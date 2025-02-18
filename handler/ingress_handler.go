package handler

import (
	"encoding/json"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/brutella/dnssd"
	"github.com/lab42/kdns/mdns"
	networkingv1 "k8s.io/api/networking/v1"
)

type IngressHandler interface {
	OnAdd(obj interface{})
	OnUpdate(oldObj interface{}, newObj interface{})
	OnDelete(obj interface{})
}

type IngressHandlerImpl struct {
	mdns *mdns.Manager
}

func NewIngressHandler(mdns *mdns.Manager) IngressHandlerImpl {
	return IngressHandlerImpl{
		mdns: mdns,
	}
}

func (h *IngressHandlerImpl) OnAdd(obj interface{}) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return
	}

	if val, ok := ingress.Annotations[mdnsEnabledAnnotation]; !ok || val != "true" {
		return
	}

	config := dnssd.Config{
		Type:   mdnsDefaultType,
		Domain: mdnsDefaultDomain,
		Port:   mdnsDefaultPort,
	}

	if val, ok := ingress.Annotations[mdnsNameAnnotation]; ok && val != "" {
		config.Name = val
	}

	if val, ok := ingress.Annotations[mdnsTypeAnnotation]; ok && val != "" {
		config.Type = val
	}

	if val, ok := ingress.Annotations[mdnsDomainAnnotation]; ok && val != "" {
		config.Domain = val
	}

	if val, ok := ingress.Annotations[mdnsHostAnnotation]; ok && val != "" {
		config.Host = val
	} else if len(ingress.Spec.Rules) > 0 {
		config.Host = ingress.Spec.Rules[0].Host
	}

	if val, ok := ingress.Annotations[mdnsTextAnnotation]; ok && val != "" {
		textMap := make(map[string]string)
		if err := json.Unmarshal([]byte(val), &textMap); err == nil {
			config.Text = textMap
		}
	}

	if val, ok := ingress.Annotations[mdnsIPsAnnotation]; ok && val != "" {
		ipStrings := strings.Split(val, ",")
		ips := make([]net.IP, 0, len(ipStrings))
		for _, ipStr := range ipStrings {
			if ip := net.ParseIP(strings.TrimSpace(ipStr)); ip != nil {
				ips = append(ips, ip)
			}
		}
		if len(ips) != 0 {
			config.IPs = ips
		}
	}

	if val, ok := ingress.Annotations[mdnsPortAnnotation]; ok && val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	}

	if val, ok := ingress.Annotations[mdnsIfacesAnnotation]; ok && val != "" {
		ifaces := strings.Split(val, ",")
		for i, iface := range ifaces {
			ifaces[i] = strings.TrimSpace(iface)
		}
		config.Ifaces = ifaces
	}

	log.Printf("added config: %v", config)
	h.mdns.Upsert(config)
}

func (h *IngressHandlerImpl) OnUpdate(oldObj interface{}, newObj interface{}) {
	h.OnDelete(oldObj)
	h.OnAdd(newObj)
}

func (h *IngressHandlerImpl) OnDelete(obj interface{}) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return
	}
	if val, ok := ingress.Annotations[mdnsNameAnnotation]; ok && val != "" {
		h.mdns.Remove(val)
	}
}
