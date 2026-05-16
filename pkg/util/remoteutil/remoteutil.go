package remoteutil

import (
	"context"
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
	mu      sync.Mutex
	srv     *tsnet.Server
	proxyLn net.Listener
	running bool
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

	go verifyControlPlanePQ(controlURL())

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

func StartProxy(localPort int) error {
	mu.Lock()
	defer mu.Unlock()
	if proxyLn != nil {
		return nil // already started
	}
	if srv == nil {
		return fmt.Errorf("tsnet not started")
	}
	ln, err := srv.Listen("tcp", ":80")
	if err != nil {
		return fmt.Errorf("tsnet listen failed: %w", err)
	}
	proxyLn = ln
	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", localPort),
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	// Rewrite the Host header so the proxied server sees the local address
	// rather than the Tailscale IP, which would confuse virtual-host routing.
	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	go func() {
		if err := http.Serve(ln, rp); err != nil {
			log.Printf("[tsnet] proxy stopped: %v", err)
		}
	}()
	return nil
}
