package mdns

import (
	"fmt"
	"sync"

	"github.com/hashicorp/mdns"
	"github.com/miekg/dns"
)

type MDNSZone struct {
	mdns.Zone
	entries map[string]*mdns.ServiceEntry
	mu      sync.RWMutex
}

// NewMDNSZone initializes a new MDNSZone.
func NewMDNSZone() *MDNSZone {
	return &MDNSZone{
		entries: make(map[string]*mdns.ServiceEntry),
	}
}

// Add registers a new service entry to the zone.
func (z *MDNSZone) Add(entry *mdns.ServiceEntry) {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.entries[entry.Name] = entry
}

// Remove deregisters a service entry from the zone.
func (z *MDNSZone) Remove(entry *mdns.ServiceEntry) {
	z.mu.Lock()
	defer z.mu.Unlock()
	delete(z.entries, entry.Name)
}

// Get retrieves a service entry by name.
func (z *MDNSZone) Get(name string) (*mdns.ServiceEntry, error) {
	z.mu.RLock()
	defer z.mu.RUnlock()
	if entry, exists := z.entries[name]; exists {
		return entry, nil
	}
	return nil, fmt.Errorf("entry not found: %s", name)
}

// Records returns DNS records in response to a DNS question.
func (z *MDNSZone) Records(q dns.Question) []dns.RR {
	z.mu.RLock()
	defer z.mu.RUnlock()

	var records []dns.RR
	for _, entry := range z.entries {
		if entry.Name == q.Name && entry.Port > 0 {
			// Create A record for IPv4 addresses
			if entry.AddrV4 != nil {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   entry.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
					},
					A: entry.AddrV4,
				}
				records = append(records, rr)
			}
			// You can also handle AAAA records for IPv6 addresses
			if entry.AddrV6 != nil {
				rr := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   entry.Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
					},
					AAAA: entry.AddrV6,
				}
				records = append(records, rr)
			}
		}
	}
	return records
}
