package watcher

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/lab42/kdns/handler"
	"github.com/lab42/kdns/mdns"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sWatcher struct {
	ingressHandler  *handler.IngressHandlerImpl
	serviceHandler  *handler.ServiceHandlerImpl
	gatewayHandler  *handler.GatewayAPIHandlerImpl
	clientset       *kubernetes.Clientset
	stopCh          chan struct{}
	informerFactory informers.SharedInformerFactory
}

func NewK8sWatcher(ingressHandler handler.IngressHandlerImpl, serviceHandler handler.ServiceHandlerImpl, mdnsManager *mdns.Manager) (*K8sWatcher, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join("/root/.kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load Kubernetes configuration: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gateway API client: %w", err)
	}
	gatewayHandler := handler.NewGatewayAPIHandler(mdnsManager, dynamicClient)

	informerFactory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)

	return &K8sWatcher{
		ingressHandler:  &ingressHandler,
		serviceHandler:  &serviceHandler,
		gatewayHandler:  &gatewayHandler,
		clientset:       clientset,
		stopCh:          make(chan struct{}),
		informerFactory: informerFactory,
	}, nil
}

func (w *K8sWatcher) Run() {
	// Set up Ingress informer
	ingressInformer := w.informerFactory.Networking().V1().Ingresses().Informer()
	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    w.ingressHandler.OnAdd,
		UpdateFunc: w.ingressHandler.OnUpdate,
		DeleteFunc: w.ingressHandler.OnDelete,
	})

	// Set up Service informer
	serviceInformer := w.informerFactory.Core().V1().Services().Informer()
	serviceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    w.serviceHandler.OnAdd,
		UpdateFunc: w.serviceHandler.OnUpdate,
		DeleteFunc: w.serviceHandler.OnDelete,
	})

	w.informerFactory.Start(w.stopCh)
	w.informerFactory.WaitForCacheSync(w.stopCh)
	go w.runGatewayReconciler()
	log.Println("K8sWatcher is running and listening for Ingress, Service, Gateway, and HTTPRoute changes.")
}

func (w *K8sWatcher) runGatewayReconciler() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if err := w.gatewayHandler.Reconcile(ctx); err != nil {
			log.Printf("Gateway API mDNS reconciliation failed: %v", err)
		}
		select {
		case <-ticker.C:
		case <-w.stopCh:
			return
		}
	}
}

func (w *K8sWatcher) Stop() {
	close(w.stopCh)
	log.Println("K8sWatcher has stopped.")
}
