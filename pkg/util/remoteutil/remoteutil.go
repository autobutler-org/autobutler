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
	"sync"

	"tailscale.com/tsnet"
)

const hostname = "autobutler"

const defaultControlURL = "http://165.227.215.101:8080"

var (
	mu      sync.Mutex
	srv     *tsnet.Server
	proxyLn net.Listener
	running bool
)

func controlURL() string {
	if u := os.Getenv("AUTOBUTLER_HEADSCALE_URL"); u != "" {
		return u
	}
	return defaultControlURL
}

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

func RemoteURL() string {
	mu.Lock()
	defer mu.Unlock()
	if !running || srv == nil {
		return ""
	}
	lc, err := srv.LocalClient()
	if err != nil {
		return ""
	}
	st, err := lc.Status(context.Background())
	if err != nil {
		return ""
	}
	if st.Self == nil || len(st.Self.TailscaleIPs) == 0 {
		return ""
	}
	ip := st.Self.TailscaleIPs[0].String()
	return fmt.Sprintf("http://%s:80", ip)
}

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
	if srv == nil {
		return fmt.Errorf("tsnet not started")
	}
	ln, err := srv.Listen("tcp", ":80")
	if err != nil {
		return fmt.Errorf("tsnet listen failed: %w", err)
	}
	proxyLn = ln
	rp := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", localPort),
	})
	go func() {
		if err := http.Serve(ln, rp); err != nil {
			log.Printf("[tsnet] proxy stopped: %v", err)
		}
	}()
	return nil
}
