// Package mdnsutil advertises the AutoButler server on the local network using
// DNS-SD / mDNS (RFC 6762 / RFC 6763). This lets devices on the same LAN
// reach the butler at autobutler.local:<port> without knowing its IP address.
//
// The advertisement is a PTR/SRV/TXT triple under the service type _autobutler._tcp.
// Most OSes also resolve the host record so that plain HTTP/HTTPS connections
// to "autobutler.local" work transparently in browsers and native apps.
package mdnsutil

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/grandcat/zeroconf"
)

const (
	// ServiceType is the DNS-SD service type advertised on the LAN.
	// Registered under the autobutler.local domain namespace.
	ServiceType = "_autobutler._tcp"

	// Domain is the mDNS local domain.
	Domain = "local."

	// DefaultInstanceName is the human-readable service instance name shown in
	// discovery UIs (e.g. "AutoButler on pi5").
	defaultInstanceName = "AutoButler"
)

var (
	mu     sync.Mutex
	server *zeroconf.Server
)

// AdvertiseOptions controls the mDNS advertisement.
type AdvertiseOptions struct {
	// Port is the TCP port the butler is listening on. Required.
	Port int
	// TXT is an optional list of key=value metadata strings attached to the
	// service record. Useful for versioning and capabilities.
	TXT []string
	// InstanceName overrides the default instance name. If empty, the
	// machine hostname is appended: "AutoButler on <hostname>".
	InstanceName string
}

// Advertise starts the mDNS advertisement. It is idempotent: calling it again
// while already running is a no-op. Call Stop to deregister.
func Advertise(opts AdvertiseOptions) error {
	mu.Lock()
	defer mu.Unlock()
	if server != nil {
		return nil // already advertising
	}

	instance := opts.InstanceName
	if instance == "" {
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			// Strip any existing domain suffix for a clean display name.
			short := strings.Split(hostname, ".")[0]
			instance = fmt.Sprintf("%s on %s", defaultInstanceName, short)
		} else {
			instance = defaultInstanceName
		}
	}

	txt := opts.TXT
	if txt == nil {
		txt = []string{}
	}

	srv, err := zeroconf.Register(
		instance,
		ServiceType,
		Domain,
		opts.Port,
		txt,
		nil, // interfaces: nil = all interfaces
	)
	if err != nil {
		return fmt.Errorf("mdns register: %w", err)
	}

	server = srv
	log.Printf("[mdns] advertising %q on %s.%s port %d", instance, ServiceType, Domain, opts.Port)
	return nil
}

// Stop deregisters the mDNS advertisement and releases the underlying socket.
// Safe to call even if Advertise was never called or already stopped.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return
	}
	server.Shutdown()
	server = nil
	log.Printf("[mdns] advertisement stopped")
}

// IsAdvertising returns true if mDNS is currently active.
func IsAdvertising() bool {
	mu.Lock()
	defer mu.Unlock()
	return server != nil
}

// DiscoverLocal probes the local network for AutoButler instances and returns
// the first result. [ctx] controls the discovery timeout — pass a context with
// a timeout (e.g. 3 seconds) to bound the search. Returns an error if nothing
// is found within the context deadline.
//
// This is primarily useful in tests and CLI tooling; the mobile onboarding
// flow uses platform-specific mDNS resolution instead.
func DiscoverLocal(ctx context.Context) (*zeroconf.ServiceEntry, error) {
	entries := make(chan *zeroconf.ServiceEntry, 8)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("mdns resolver: %w", err)
	}
	if err := resolver.Browse(ctx, ServiceType, Domain, entries); err != nil {
		return nil, fmt.Errorf("mdns browse: %w", err)
	}
	select {
	case entry := <-entries:
		return entry, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("no AutoButler found on the local network within the timeout")
	}
}
