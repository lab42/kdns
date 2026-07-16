package handler

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGatewayAPIConfigsPublishesAcceptedLocalRoute(t *testing.T) {
	gateway := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata": map[string]interface{}{
			"name":      "main",
			"namespace": "gateway-system",
		},
		"spec": map[string]interface{}{
			"listeners": []interface{}{map[string]interface{}{
				"name": "https", "protocol": "HTTPS", "port": int64(443),
			}},
		},
		"status": map[string]interface{}{
			"addresses": []interface{}{map[string]interface{}{
				"type": "IPAddress", "value": "192.0.2.10",
			}},
		},
	}}
	route := unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "HTTPRoute",
		"metadata": map[string]interface{}{
			"name":      "dashboard",
			"namespace": "apps",
			"annotations": map[string]interface{}{
				mdnsEnabledAnnotation: "true",
			},
		},
		"spec": map[string]interface{}{
			"hostnames": []interface{}{"dashboard.example.local"},
			"parentRefs": []interface{}{map[string]interface{}{
				"name": "main", "namespace": "gateway-system", "sectionName": "https",
			}},
		},
		"status": map[string]interface{}{
			"parents": []interface{}{map[string]interface{}{
				"parentRef": map[string]interface{}{
					"name": "main", "namespace": "gateway-system", "sectionName": "https",
				},
				"conditions": []interface{}{map[string]interface{}{
					"type": "Accepted", "status": "True",
				}},
			}},
		},
	}}

	configs := gatewayAPIConfigs([]unstructured.Unstructured{route}, []unstructured.Unstructured{gateway})
	config, exists := configs["dashboard.example"]
	if !exists {
		t.Fatalf("expected dashboard.example config, got %#v", configs)
	}
	if config.Host != "dashboard.example.local" || config.Port != 443 || config.Type != "_https._tcp" {
		t.Fatalf("unexpected config: %#v", config)
	}
	if got := config.IPs[0].String(); got != "192.0.2.10" {
		t.Fatalf("expected gateway address 192.0.2.10, got %s", got)
	}
}

func TestGatewayAPIConfigsIgnoresUnapprovedRoutesAndPublicHosts(t *testing.T) {
	route := unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "public", "namespace": "apps",
			"annotations": map[string]interface{}{mdnsEnabledAnnotation: "true"},
		},
		"spec": map[string]interface{}{
			"hostnames":  []interface{}{"public.example.com"},
			"parentRefs": []interface{}{map[string]interface{}{"name": "main"}},
		},
	}}
	if configs := gatewayAPIConfigs([]unstructured.Unstructured{route}, nil); len(configs) != 0 {
		t.Fatalf("expected no configs, got %#v", configs)
	}
}
