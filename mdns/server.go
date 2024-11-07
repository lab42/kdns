package mdns

import (
	"fmt"
	"net"

	"github.com/hashicorp/mdns"
)

// MDNSServer struct with zone implementation
type MDNSServer struct {
	server *mdns.Server
	zone   *MDNSZone
}

// NewMDNSServer initializes the mDNS server with the custom zone.
func NewMDNSServer() (*MDNSServer, error) {
	return &MDNSServer{
		zone: NewMDNSZone(),
	}, nil
}

// Start starts the mDNS server with the custom zone.
func (s *MDNSServer) Start() error {
	var err error
	s.server, err = mdns.NewServer(&mdns.Config{Zone: s.zone})
	if err != nil {
		return fmt.Errorf("failed to start mDNS server: %w", err)
	}
	return nil
}

// Stop stops the mDNS server.
func (s *MDNSServer) Stop() {
	if s.server != nil {
		s.server.Shutdown()
	}
}

//	if err := server.AddServiceWithIPAndTXT("my-service", "_http._tcp", "my-service.local.", 8080, ip); err != nil {
//		log.Fatalf("Failed to add service: %v", err)
//	}
//
// AddServiceWithIP registers a service with a specific IP address.
func (s *MDNSServer) AddService(serviceName, serviceType, hostname string, port int, ip net.IP) error {
	entry := &mdns.ServiceEntry{
		Name:   fmt.Sprintf("%s.%s", serviceName, serviceType),
		Host:   hostname,
		Port:   port,
		AddrV4: ip,
	}

	if ip.To4() != nil {
		entry.AddrV4 = ip
	}

	if ip.To16() != nil {
		entry.AddrV6 = ip
	}

	s.zone.Add(entry)

	return nil
}

// RemoveService removes a registered service by its name.
func (s *MDNSServer) RemoveService(serviceName string) error {
	if _, found := s.zone.entries[serviceName]; !found {
		return fmt.Errorf("service not found: %s", serviceName)
	}
	delete(s.zone.entries, serviceName)
	return nil
}
