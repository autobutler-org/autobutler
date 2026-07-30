package remoteutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

const hostname = "autobutler"

const defaultControlURL = "https://network.autobutler.org"

var (
	mu              sync.Mutex
	srv             *tsnet.Server
	proxyLn         net.Listener
	running         bool
	tlsProxyRunning bool // true when the :443 TLS proxy is active
)

func controlURL() string {
	u := os.Getenv("AUTOBUTLER_HEADSCALE_URL")
	if u == "" {
		u = defaultControlURL
	}
	if strings.HasPrefix(u, "http://") {
		log.Printf("[remote] WARNING: Headscale control URL is using HTTP (%s). Auth keys will be sent in plaintext. Use HTTPS in production.", u)
	}
	return u
}

// stateDir returns the path where tsnet should persist its state. On Linux it
// prefers the system service directory if the parent exists; otherwise it falls
// back to the user's home config directory. Directory creation is left to the
// caller (Start).
func stateDir() string {
	if runtime.GOOS == "linux" {
		svcDir := "/var/lib/autobutler/tsnet"
		if _, err := os.Stat(filepath.Dir(svcDir)); err == nil {
			return svcDir
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".config", "autobutler", "tsnet")
}

func Start(authKey string) error {
	mu.Lock()
	defer mu.Unlock()
	if running {
		return nil
	}
	dir := stateDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create tsnet state dir: %w", err)
	}
	srv = &tsnet.Server{
		Hostname:   hostname,
		AuthKey:    authKey,
		Dir:        dir,
		ControlURL: controlURL(),
		Logf: func(format string, args ...any) {
			log.Printf("[tsnet] "+format, args...)
		},
	}
	if err := srv.Start(); err != nil {
		srv = nil
		return fmt.Errorf("failed to start tsnet: %w", err)
	}
	running = true
	return nil
}

func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if proxyLn != nil {
		proxyLn.Close()
		proxyLn = nil
	}
	if srv != nil {
		srv.Close()
		srv = nil
	}
	running = false
}

func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return running
}

// IsTLSActive returns true when the tailnet proxy is using a Tailscale-issued
// TLS certificate (HTTPS on :443), false when falling back to HTTP on :80.
func IsTLSActive() bool {
	mu.Lock()
	defer mu.Unlock()
	return tlsProxyRunning
}

// RemoteURL returns the URL for reaching this node over the tailnet.
//
// If HTTPS (Tailscale-issued cert) is available, it returns the MagicDNS
// hostname with https:// — no browser warnings. Otherwise it falls back to
// the Tailscale IP with http://.
//
// The mutex is held only long enough to snapshot the server pointer; the
// network calls to the local Tailscale daemon happen outside the lock.
func RemoteURL() string {
	mu.Lock()
	if !running || srv == nil {
		mu.Unlock()
		return ""
	}
	s := srv
	tlsEnabled := tlsProxyRunning
	mu.Unlock()

	lc, err := s.LocalClient()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := lc.Status(ctx)
	if err != nil {
		return ""
	}

	// Prefer MagicDNS HTTPS URL when TLS is active.
	if tlsEnabled && st.CurrentTailnet != nil && st.CurrentTailnet.MagicDNSSuffix != "" {
		// DNSName from Self already contains the FQDN (e.g. "autobutler.example.ts.net.");
		// strip the trailing dot if present.
		if st.Self != nil && st.Self.DNSName != "" {
			host := strings.TrimSuffix(st.Self.DNSName, ".")
			return fmt.Sprintf("https://%s", host)
		}
	}

	// Fall back to IP-based URL.
	if st.Self == nil || len(st.Self.TailscaleIPs) == 0 {
		return ""
	}
	ip := st.Self.TailscaleIPs[0].String()
	if tlsEnabled {
		return fmt.Sprintf("https://%s", ip)
	}
	return fmt.Sprintf("http://%s:80", ip)
}

// StartProxy starts an HTTP reverse proxy on the tsnet listener at :80,
// forwarding traffic to the local butler server at localPort. It is idempotent:
// if the proxy listener is already open, it returns nil immediately.
//
// The proxy is intentionally unauthenticated at the tsnet layer — access
// control is enforced by the proxied butler server's own auth middleware. Only
// peers on the tailnet can reach this listener.
// HasPersistedState returns true if tsnet has previously stored credentials
// on disk and can reconnect without a new auth key.
func HasPersistedState() bool {
	dir := stateDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			return true
		}
	}
	return false
}

// StartProxy forwards tailnet traffic to the local butler on [localPort].
//
// It first attempts to start an HTTPS proxy on :443 using a Tailscale-issued
// Let's Encrypt certificate (requires MagicDNS + HTTPS enabled in the
// Tailscale/Headscale admin panel). If that is not available (MagicDNS
// disabled, HTTPS not enabled, or the control server does not support ACME),
// it falls back to a plain HTTP proxy on :80.
//
// [localTLS] must match how the butler is actually serving that port — see
// serverutil.ServingTLS.
func StartProxy(localPort int, localTLS bool) error {
	mu.Lock()
	defer mu.Unlock()
	if proxyLn != nil {
		return nil // already started
	}
	if srv == nil {
		return fmt.Errorf("tsnet not started")
	}

	// Build the reverse proxy target (points at the local butler).
	scheme := "http"
	if localTLS {
		scheme = "https"
	}
	target := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("localhost:%d", localPort),
	}
	rp := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
		},
	}
	if localTLS {
		// The butler presents its own self-signed cert on loopback — skip
		// verification for this private hop.
		rp.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	// Attempt TLS proxy on :443 first (Tailscale-issued cert).
	if ln, err := srv.ListenTLS("tcp", ":443"); err == nil {
		proxyLn = ln
		tlsProxyRunning = true
		log.Printf("[tsnet] HTTPS proxy active on :443 (Tailscale cert)")
		go func() {
			if err := http.Serve(ln, rp); err != nil {
				log.Printf("[tsnet] HTTPS proxy stopped: %v", err)
				mu.Lock()
				tlsProxyRunning = false
				mu.Unlock()
			}
		}()
		return nil
	} else {
		log.Printf("[tsnet] Tailscale cert not available (%v); falling back to HTTP on :80", err)
	}

	// Fall back: plain HTTP on :80.
	ln, err := srv.Listen("tcp", ":80")
	if err != nil {
		return fmt.Errorf("tsnet listen failed: %w", err)
	}
	proxyLn = ln
	go func() {
		if err := http.Serve(ln, rp); err != nil {
			log.Printf("[tsnet] HTTP proxy stopped: %v", err)
		}
	}()
	return nil
}
