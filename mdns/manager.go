package mdns

import (
	"context"
	"fmt"
	"sync"

	"github.com/brutella/dnssd"
)

type Manager struct {
	Responder      dnssd.Responder
	ServiceHandles map[string]dnssd.ServiceHandle
	mu             sync.RWMutex
}

func NewManager() (manager Manager, err error) {
	responder, err := dnssd.NewResponder()
	if err != nil {
		return
	}

	manager.Responder = responder
	manager.ServiceHandles = make(map[string]dnssd.ServiceHandle)
	return
}

func (m *Manager) Exists(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.ServiceHandles[name]
	return exists
}

// Upsert creates or updates a DNS-SD service with the provided configuration.
// If a service with the same name exists, it will be replaced.
func (m *Manager) Upsert(config dnssd.Config) error {
	service, err := dnssd.NewService(config)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// If service exists, remove it first
	if handle, exists := m.ServiceHandles[config.Name]; exists {
		m.Responder.Remove(handle)
	}

	// Add the new service
	handle, err := m.Responder.Add(service)
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	m.ServiceHandles[config.Name] = handle
	return nil
}

func (m *Manager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	handle, exists := m.ServiceHandles[name]
	if !exists {
		return fmt.Errorf("service with name %s not found", name)
	}

	m.Responder.Remove(handle)
	delete(m.ServiceHandles, name)
	return nil
}

func (m *Manager) Respond(ctx context.Context) error {
	return m.Responder.Respond(ctx)
}
