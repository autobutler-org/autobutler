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
	mu         sync.Mutex
	srv        *tsnet.Server
	proxyLn    net.Listener // :80 plain HTTP proxy
	proxyTLSLn net.Listener // :443 HTTPS proxy (Tailscale cert)
	running    bool
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
	if proxyTLSLn != nil {
		proxyTLSLn.Close()
		proxyTLSLn = nil
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

// RemoteURL returns the Tailscale IP-based URL for the tsnet node, or "" if
// not running. The mutex is held only long enough to snapshot the server
// pointer; the network call to the local Tailscale daemon happens outside the
// lock so that Stop() and IsRunning() are never blocked by I/O.
func RemoteURL() string {
	mu.Lock()
	if !running || srv == nil {
		mu.Unlock()
		return ""
	}
	s := srv
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
	if st.Self == nil || len(st.Self.TailscaleIPs) == 0 {
		return ""
	}
	ip := st.Self.TailscaleIPs[0].String()
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
// [localTLS] must match how the butler is actually serving that port.
//
// This starts two listeners on the tailnet:
//   - :80  plain HTTP proxy (always started)
//   - :443 HTTPS proxy using Tailscale-provisioned Let's Encrypt certs
//     (started only when MagicDNS + HTTPS are enabled in the tailnet admin console)
//
// The :443 listener removes the "your connection is not private" browser
// warning for family members accessing via Tailscale. The :80 listener stays
// as a fallback and for redirect purposes.
func StartProxy(localPort int, localTLS bool) error {
	mu.Lock()
	defer mu.Unlock()
	if proxyLn != nil {
		return nil // already started
	}
	if srv == nil {
		return fmt.Errorf("tsnet not started")
	}

	rp := buildReverseProxy(localPort, localTLS)

	// :80 — plain HTTP; always works regardless of MagicDNS/cert settings.
	ln, err := srv.Listen("tcp", ":80")
	if err != nil {
		return fmt.Errorf("tsnet listen :80 failed: %w", err)
	}
	proxyLn = ln
	go func() {
		if err := http.Serve(ln, rp); err != nil {
			log.Printf("[tsnet] :80 proxy stopped: %v", err)
		}
	}()

	// :443 — HTTPS via Tailscale-provisioned Let's Encrypt cert.
	// Best-effort: if MagicDNS or HTTPS is not enabled, log and skip.
	tlsLn, err := srv.ListenTLS("tcp", ":443")
	if err != nil {
		log.Printf("[tsnet] HTTPS proxy not started (MagicDNS/HTTPS may not be enabled): %v", err)
	} else {
		proxyTLSLn = tlsLn
		go func() {
			if err := http.Serve(tlsLn, rp); err != nil {
				log.Printf("[tsnet] :443 proxy stopped: %v", err)
			}
		}()
		log.Printf("[tsnet] HTTPS proxy started on tailnet :443 (Tailscale cert)")
	}

	return nil
}

// buildReverseProxy creates an httputil.ReverseProxy targeting the local
// butler at localhost:[localPort]. If localTLS is true, TLS server verification
// is skipped — the loopback cert is self-signed by the butler itself.
func buildReverseProxy(localPort int, localTLS bool) *httputil.ReverseProxy {
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
		rp.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return rp
}

// TailscaleDomain returns the *.ts.net (or tailnet DNS) hostname for the tsnet
// node if HTTPS certs are available, or "" otherwise. Callers can surface this
// in the UI so users know which hostname to use for valid-cert access.
func TailscaleDomain() string {
	mu.Lock()
	if !running || srv == nil {
		mu.Unlock()
		return ""
	}
	s := srv
	mu.Unlock()

	lc, err := s.LocalClient()
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := lc.Status(ctx)
	if err != nil || len(st.CertDomains) == 0 {
		return ""
	}
	return st.CertDomains[0]
}
