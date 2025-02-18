package watcher

import (
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/lab42/kdns/handler"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sWatcher struct {
	ingressHandler  *handler.IngressHandlerImpl
	serviceHandler  *handler.ServiceHandlerImpl
	clientset       *kubernetes.Clientset
	stopCh          chan struct{}
	informerFactory informers.SharedInformerFactory
}

func NewK8sWatcher(ingressHandler handler.IngressHandlerImpl, serviceHandler handler.ServiceHandlerImpl) (*K8sWatcher, error) {
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

	informerFactory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)

	return &K8sWatcher{
		ingressHandler:  &ingressHandler,
		serviceHandler:  &serviceHandler,
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
	log.Println("K8sWatcher is running and listening for Ingress and Service changes.")
}

func (w *K8sWatcher) Stop() {
	close(w.stopCh)
	log.Println("K8sWatcher has stopped.")
}
