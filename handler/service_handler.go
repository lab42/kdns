// service_handler.go
package handler

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/brutella/dnssd"
	"github.com/lab42/kdns/mdns"
	corev1 "k8s.io/api/core/v1"
)

type ServiceHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj interface{}, newObj interface{})
	OnDelete(obj interface{})
}

type ServiceHandlerImpl struct {
	mdns *mdns.Manager
}

func NewServiceHandler(mdns *mdns.Manager) ServiceHandlerImpl {
	return ServiceHandlerImpl{
		mdns: mdns,
	}
}

func (h *ServiceHandlerImpl) OnAdd(obj interface{}) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return
	}

	// Only process LoadBalancer services
	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return
	}

	if val, ok := service.Annotations[mdnsEnabledAnnotation]; !ok || val != "true" {
		return
	}

	config := dnssd.Config{
		Type:   mdnsDefaultType,
		Domain: mdnsDefaultDomain,
		Port:   mdnsDefaultPort,
	}

	// Use service name if mdns name not specified
	if val, ok := service.Annotations[mdnsNameAnnotation]; ok && val != "" {
		config.Name = val
	} else {
		config.Name = service.Name
	}

	if val, ok := service.Annotations[mdnsTypeAnnotation]; ok && val != "" {
		config.Type = val
	}

	if val, ok := service.Annotations[mdnsDomainAnnotation]; ok && val != "" {
		config.Domain = val
	}

	if val, ok := service.Annotations[mdnsHostAnnotation]; ok && val != "" {
		config.Host = val
	}

	if val, ok := service.Annotations[mdnsTextAnnotation]; ok && val != "" {
		textMap := make(map[string]string)
		if err := json.Unmarshal([]byte(val), &textMap); err == nil {
			config.Text = textMap
		}
	}

	// Handle LoadBalancer IPs
	if val, ok := service.Annotations[mdnsIPsAnnotation]; ok && val != "" {
		ipStrings := strings.Split(val, ",")
		ips := make([]net.IP, 0, len(ipStrings))
		for _, ipStr := range ipStrings {
			if ip := net.ParseIP(strings.TrimSpace(ipStr)); ip != nil {
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			config.IPs = ips
		}
	} else {
		// Use LoadBalancer ingress IPs if no specific IPs are provided
		ips := make([]net.IP, 0)
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				if ip := net.ParseIP(ingress.IP); ip != nil {
					ips = append(ips, ip)
				}
			}
		}
		if len(ips) > 0 {
			config.IPs = ips
		}
	}

	if val, ok := service.Annotations[mdnsPortAnnotation]; ok && val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	} else if len(service.Spec.Ports) > 0 {
		// Use the first port if not specified
		config.Port = int(service.Spec.Ports[0].Port)
	}

	if val, ok := service.Annotations[mdnsIfacesAnnotation]; ok && val != "" {
		ifaces := strings.Split(val, ",")
		for i, iface := range ifaces {
			ifaces[i] = strings.TrimSpace(iface)
		}
		config.Ifaces = ifaces
	}

	log.Printf("added LoadBalancer service config: %v", config)
	h.mdns.Upsert(config)
}

func (h *ServiceHandlerImpl) OnUpdate(oldObj interface{}, newObj interface{}) {
	h.OnDelete(oldObj)
	h.OnAdd(newObj)
}

func (h *ServiceHandlerImpl) OnDelete(obj interface{}) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return
	}
	if val, ok := service.Annotations[mdnsNameAnnotation]; ok && val != "" {
		h.mdns.Remove(val)
	} else {
		h.mdns.Remove(service.Name)
	}
}
