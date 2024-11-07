// k8swatcher.go
package watcher

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/lab42/kdns/mdns"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type K8sWatcher struct {
	mDNS            *mdns.MDNSServer
	clientset       *kubernetes.Clientset
	stopCh          chan struct{}
	informerFactory informers.SharedInformerFactory
}

// NewK8sWatcher initializes the Kubernetes watcher.
func NewK8sWatcher(mDNS *mdns.MDNSServer) (*K8sWatcher, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create a shared informer factory with a resync interval
	informerFactory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)

	return &K8sWatcher{
		mDNS:            mDNS,
		clientset:       clientset,
		stopCh:          make(chan struct{}),
		informerFactory: informerFactory,
	}, nil
}

// Run starts the watcher and listens for changes in Ingress resources.
func (w *K8sWatcher) Run() {
	// Ingress Informer
	ingressInformer := w.informerFactory.Networking().V1().Ingresses().Informer()
	ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    w.onIngressAdd,
		UpdateFunc: w.onIngressUpdate,
		DeleteFunc: w.onIngressDelete,
	})

	// Start the informer factory to begin listening for changes
	w.informerFactory.Start(w.stopCh)

	// Wait for the initial cache sync to complete
	w.informerFactory.WaitForCacheSync(w.stopCh)
	log.Println("K8sWatcher is running and listening for changes.")
}

// Stop stops the watcher and all associated informers.
func (w *K8sWatcher) Stop() {
	close(w.stopCh)
	log.Println("K8sWatcher has stopped.")
}

// Event Handlers for Ingress
func (w *K8sWatcher) onIngressAdd(obj interface{}) {
	ingress := obj.(*networkingv1.Ingress)

	// Check if there are any addresses in the Ingress status
	var ingressIP net.IP
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		ingressIP = net.ParseIP(ingress.Status.LoadBalancer.Ingress[0].IP)
		fmt.Printf("Ingress IP: %s\n", ingressIP)
	} else {
		fmt.Println("No Ingress IP found.")
		return // Exit if no IP is found
	}

	// Extract the service details from the Ingress
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			serviceName := path.Backend.Service.Name
			servicePort := path.Backend.Service.Port.Number
			namespace := ingress.Namespace // Get the namespace from the Ingress

			// Create a service name for mDNS (adjust as needed)
			serviceID := fmt.Sprintf("%s.%s.local", serviceName, namespace)
			fmt.Printf("Registering mDNS service: %s on port %d\n", serviceID, servicePort)

			// Add the service to the mDNS server
			err := w.mDNS.AddService(serviceName, "http", serviceName, int(servicePort), ingressIP) // serviceType is set to "http"
			if err != nil {
				fmt.Printf("Error adding service to mDNS: %s\n", err)
			}
		}
	}
}

func (w *K8sWatcher) onIngressUpdate(oldObj, newObj interface{}) {
	oldIngress := oldObj.(*networkingv1.Ingress)
	newIngress := newObj.(*networkingv1.Ingress)

	// Check if there are any meaningful changes between old and new Ingress
	if !w.ingressChanged(oldIngress, newIngress) {
		log.Printf("No significant changes in Ingress %s/%s, skipping update", newIngress.Namespace, newIngress.Name)
		return
	}

	// Remove old Ingress services from mDNS
	w.onIngressDelete(oldIngress)

	// Register new Ingress services to mDNS
	w.onIngressAdd(newIngress)

	log.Printf("Ingress updated: %s/%s", newIngress.Namespace, newIngress.Name)
}

func (w *K8sWatcher) onIngressDelete(obj interface{}) {
	ingress := obj.(*networkingv1.Ingress)

	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			serviceName := path.Backend.Service.Name
			w.mDNS.RemoveService(serviceName)
		}
	}
}

func (w *K8sWatcher) ingressChanged(oldIngress, newIngress *networkingv1.Ingress) bool {
	// Compare LoadBalancer IPs
	if len(oldIngress.Status.LoadBalancer.Ingress) != len(newIngress.Status.LoadBalancer.Ingress) {
		return true
	}
	for i, oldEntry := range oldIngress.Status.LoadBalancer.Ingress {
		if oldEntry.IP != newIngress.Status.LoadBalancer.Ingress[i].IP {
			return true
		}
	}

	// Compare Ingress rules and paths
	if len(oldIngress.Spec.Rules) != len(newIngress.Spec.Rules) {
		return true
	}
	for i, oldRule := range oldIngress.Spec.Rules {
		newRule := newIngress.Spec.Rules[i]
		if oldRule.Host != newRule.Host {
			return true
		}

		// Compare HTTP paths
		if oldRule.HTTP == nil || newRule.HTTP == nil {
			if oldRule.HTTP != newRule.HTTP { // one is nil, the other is not
				return true
			}
			continue
		}
		if len(oldRule.HTTP.Paths) != len(newRule.HTTP.Paths) {
			return true
		}
		for j, oldPath := range oldRule.HTTP.Paths {
			newPath := newRule.HTTP.Paths[j]
			if oldPath.Path != newPath.Path ||
				oldPath.Backend.Service.Name != newPath.Backend.Service.Name ||
				oldPath.Backend.Service.Port.Number != newPath.Backend.Service.Port.Number {
				return true
			}
		}
	}

	// If no significant changes detected, return false
	return false
}
