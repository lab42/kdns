package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/brutella/dnssd"
	"github.com/lab42/kdns/mdns"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	gatewayGVR   = schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	httpRouteGVR = schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
)

type GatewayAPIHandlerImpl struct {
	mdns       *mdns.Manager
	client     dynamic.Interface
	mu         sync.Mutex
	advertised map[string]string
}

func NewGatewayAPIHandler(mdnsManager *mdns.Manager, client dynamic.Interface) GatewayAPIHandlerImpl {
	return GatewayAPIHandlerImpl{
		mdns:       mdnsManager,
		client:     client,
		advertised: make(map[string]string),
	}
}

func (h *GatewayAPIHandlerImpl) Reconcile(ctx context.Context) error {
	routes, err := h.client.Resource(httpRouteGVR).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list HTTPRoutes: %w", err)
	}
	gatewayList, err := h.client.Resource(gatewayGVR).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list Gateways: %w", err)
	}

	desired := gatewayAPIConfigs(routes.Items, gatewayList.Items)
	h.mu.Lock()
	defer h.mu.Unlock()

	for name := range h.advertised {
		if _, exists := desired[name]; !exists {
			if err := h.mdns.Remove(name); err != nil {
				log.Printf("failed to remove Gateway API mDNS config %s: %v", name, err)
			} else {
				log.Printf("removed Gateway API mDNS config: %s", name)
			}
			delete(h.advertised, name)
		}
	}

	for name, config := range desired {
		fingerprint := configFingerprint(config)
		if h.advertised[name] == fingerprint {
			continue
		}
		if err := h.mdns.Upsert(config); err != nil {
			log.Printf("failed to publish Gateway API mDNS config %s: %v", name, err)
			continue
		}
		h.advertised[name] = fingerprint
		log.Printf("published Gateway API mDNS config: %s -> %v:%d", config.Host, config.IPs, config.Port)
	}
	return nil
}

func gatewayAPIConfigs(routes, gateways []unstructured.Unstructured) map[string]dnssd.Config {
	gatewayIndex := make(map[string]unstructured.Unstructured, len(gateways))
	for _, gateway := range gateways {
		gatewayIndex[gateway.GetNamespace()+"/"+gateway.GetName()] = gateway
	}

	result := make(map[string]dnssd.Config)
	for _, route := range routes {
		annotations := route.GetAnnotations()
		if annotations[mdnsEnabledAnnotation] != "true" {
			continue
		}
		hostnames, _, _ := unstructured.NestedStringSlice(route.Object, "spec", "hostnames")
		if len(hostnames) == 0 {
			continue
		}
		parentRefs, _, _ := unstructured.NestedSlice(route.Object, "spec", "parentRefs")
		for _, rawParent := range parentRefs {
			parent, ok := rawParent.(map[string]interface{})
			if !ok || stringValue(parent["kind"], "Gateway") != "Gateway" || stringValue(parent["group"], "gateway.networking.k8s.io") != "gateway.networking.k8s.io" {
				continue
			}
			name, _ := parent["name"].(string)
			if name == "" {
				continue
			}
			namespace := stringValue(parent["namespace"], route.GetNamespace())
			sectionName, _ := parent["sectionName"].(string)
			if !routeParentAccepted(route, namespace, name, sectionName) {
				continue
			}
			gateway, exists := gatewayIndex[namespace+"/"+name]
			if !exists {
				continue
			}
			ips := gatewayAddresses(gateway)
			if override := parseIPs(annotations[mdnsIPsAnnotation]); len(override) > 0 {
				ips = override
			}
			if len(ips) == 0 {
				continue
			}
			port, serviceType := gatewayListener(gateway, sectionName)
			if value, err := strconv.Atoi(annotations[mdnsPortAnnotation]); err == nil && value > 0 {
				port = value
			}
			if annotations[mdnsTypeAnnotation] != "" {
				serviceType = annotations[mdnsTypeAnnotation]
			}
			domain := strings.Trim(strings.TrimSpace(stringValue(annotations[mdnsDomainAnnotation], mdnsDefaultDomain)), ".")
			for _, rawHostname := range hostnames {
				hostname := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(rawHostname)), ".")
				suffix := "." + strings.ToLower(domain)
				if !strings.HasSuffix(hostname, suffix) {
					continue
				}
				configName := strings.TrimSuffix(hostname, suffix)
				if len(hostnames) == 1 && annotations[mdnsNameAnnotation] != "" {
					configName = annotations[mdnsNameAnnotation]
				}
				host := hostname
				if len(hostnames) == 1 && annotations[mdnsHostAnnotation] != "" {
					host = annotations[mdnsHostAnnotation]
				}
				config := dnssd.Config{
					Name:   configName,
					Host:   host,
					Type:   serviceType,
					Domain: domain,
					Port:   port,
					IPs:    ips,
				}
				if text := annotations[mdnsTextAnnotation]; text != "" {
					_ = json.Unmarshal([]byte(text), &config.Text)
				}
				if rawIfaces := annotations[mdnsIfacesAnnotation]; rawIfaces != "" {
					for _, iface := range strings.Split(rawIfaces, ",") {
						if trimmed := strings.TrimSpace(iface); trimmed != "" {
							config.Ifaces = append(config.Ifaces, trimmed)
						}
					}
				}
				if config.Name != "" {
					result[config.Name] = config
				}
			}
		}
	}
	return result
}

func routeParentAccepted(route unstructured.Unstructured, namespace, name, sectionName string) bool {
	parents, _, _ := unstructured.NestedSlice(route.Object, "status", "parents")
	for _, rawParent := range parents {
		parent, ok := rawParent.(map[string]interface{})
		if !ok {
			continue
		}
		parentRef, _, _ := unstructured.NestedMap(parent, "parentRef")
		if stringValue(parentRef["name"], "") != name || stringValue(parentRef["namespace"], route.GetNamespace()) != namespace || stringValue(parentRef["sectionName"], "") != sectionName {
			continue
		}
		conditions, _, _ := unstructured.NestedSlice(parent, "conditions")
		for _, rawCondition := range conditions {
			condition, ok := rawCondition.(map[string]interface{})
			if ok && condition["type"] == "Accepted" && condition["status"] == "True" {
				return true
			}
		}
	}
	return false
}

func gatewayAddresses(gateway unstructured.Unstructured) []net.IP {
	addresses, _, _ := unstructured.NestedSlice(gateway.Object, "status", "addresses")
	result := make([]net.IP, 0, len(addresses))
	for _, rawAddress := range addresses {
		address, ok := rawAddress.(map[string]interface{})
		if !ok || stringValue(address["type"], "IPAddress") != "IPAddress" {
			continue
		}
		if parsed := net.ParseIP(stringValue(address["value"], "")); parsed != nil {
			result = append(result, parsed)
		}
	}
	return result
}

func gatewayListener(gateway unstructured.Unstructured, sectionName string) (int, string) {
	listeners, _, _ := unstructured.NestedSlice(gateway.Object, "spec", "listeners")
	if sectionName == "" {
		for _, rawListener := range listeners {
			listener, ok := rawListener.(map[string]interface{})
			if ok && strings.EqualFold(stringValue(listener["protocol"], ""), "HTTPS") {
				return listenerConfig(listener)
			}
		}
	}
	for _, rawListener := range listeners {
		listener, ok := rawListener.(map[string]interface{})
		if !ok || sectionName != "" && stringValue(listener["name"], "") != sectionName {
			continue
		}
		return listenerConfig(listener)
	}
	return 443, "_https._tcp"
}

func listenerConfig(listener map[string]interface{}) (int, string) {
	port := 443
	switch value := listener["port"].(type) {
	case int64:
		port = int(value)
	case float64:
		port = int(value)
	}
	serviceType := mdnsDefaultType
	if strings.EqualFold(stringValue(listener["protocol"], ""), "HTTPS") {
		serviceType = "_https._tcp"
	}
	return port, serviceType
}

func parseIPs(value string) []net.IP {
	result := make([]net.IP, 0)
	for _, candidate := range strings.Split(value, ",") {
		if parsed := net.ParseIP(strings.TrimSpace(candidate)); parsed != nil {
			result = append(result, parsed)
		}
	}
	return result
}

func configFingerprint(config dnssd.Config) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d|%v|%v|%v", config.Name, config.Host, config.Type, config.Domain, config.Port, config.IPs, config.Ifaces, config.Text)
}

func stringValue(value interface{}, fallback string) string {
	if result, ok := value.(string); ok && result != "" {
		return result
	}
	return fallback
}
