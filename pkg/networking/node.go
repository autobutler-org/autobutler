package networking

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"tailscale.com/tsnet"
)

type NodeStatus string

const (
	StatusOnline   NodeStatus = "online"
	StatusOffline  NodeStatus = "offline"
	StatusDegraded NodeStatus = "degraded"
)

type Node struct {
	mu          sync.RWMutex
	server      *tsnet.Server
	config      *Config
	metrics     *Metrics
	diagnostics *Diagnostics
	status      NodeStatus
	localIP     string
	ipv6        string
	tailnetIP   string
	tlsCert     *x509.Certificate
	logger      *slog.Logger
	shutdownCh  chan struct{}
}

type NodeInfo struct {
	Name      string     `json:"name"`
	LocalIP   string     `json:"local_ip"`
	IPv6      string     `json:"ipv6"`
	TailnetIP string     `json:"tailnet_ip,omitempty"`
	Status    NodeStatus `json:"status"`
	Uptime    int64      `json:"uptime"`
}

func NewNode(cfg *Config, logger *slog.Logger) (*Node, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	if err := os.MkdirAll(cfg.StateDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	node := &Node{
		config:      cfg,
		metrics:     NewMetrics(),
		diagnostics: NewDiagnostics(),
		status:      StatusOffline,
		logger:      logger,
		shutdownCh:  make(chan struct{}),
	}

	if err := node.initTLSCertificate(); err != nil {
		logger.Warn("failed to initialize TLS certificate", "error", err)
	}

	return node, nil
}

func (n *Node) Start(ctx context.Context) error {
	n.logger.Info("starting networking node", "hostname", n.config.Hostname)

	srv := &tsnet.Server{
		Hostname: n.config.Hostname,
		Dir:      filepath.Join(n.config.StateDir, "tsnet"),
		Logf: func(format string, args ...any) {
			n.logger.Debug(fmt.Sprintf(format, args...))
		},
	}

	if n.config.ControlURL != "" {
		srv.ControlURL = n.config.ControlURL
	}

	if n.config.AuthKey != "" {
		srv.AuthKey = n.config.AuthKey
	}

	n.mu.Lock()
	n.server = srv
	n.mu.Unlock()

	if err := n.updateNetworkInfo(ctx); err != nil {
		n.logger.Warn("failed to get network info", "error", err)
	}

	n.mu.Lock()
	n.status = StatusOnline
	n.mu.Unlock()

	go n.metricsLoop(ctx)

	n.logger.Info("networking node started", "hostname", n.config.Hostname, "local_ip", n.localIP)
	return nil
}

func (n *Node) Shutdown(ctx context.Context) error {
	n.logger.Info("shutting down networking node")

	close(n.shutdownCh)

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.server != nil {
		if err := n.server.Close(); err != nil {
			return fmt.Errorf("failed to close tsnet server: %w", err)
		}
	}

	n.status = StatusOffline
	n.logger.Info("networking node stopped")
	return nil
}

func (n *Node) updateNetworkInfo(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				n.localIP = ipnet.IP.String()
			} else if ipnet.IP.To16() != nil && n.ipv6 == "" {
				n.ipv6 = ipnet.IP.String()
			}
		}
	}

	if n.server != nil {
		// Use a short timeout to prevent blocking when auth key is invalid
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		status, err := n.server.Up(timeoutCtx)
		if err == nil && len(status.TailscaleIPs) > 0 {
			n.tailnetIP = status.TailscaleIPs[0].String()
		} else if err != nil {
			// Log but don't fail - just means we're not connected yet
			n.logger.Debug("tailscale not up yet", "error", err)
		}
	}

	return nil
}

func (n *Node) metricsLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.shutdownCh:
			return
		case <-ticker.C:
			n.updateMetrics(ctx)
		}
	}
}

func (n *Node) updateMetrics(ctx context.Context) {
	if err := n.updateNetworkInfo(ctx); err != nil {
		n.logger.Warn("failed to update network info", "error", err)
	}

	latency := n.measureLatency(ctx)
	n.metrics.UpdateLatency(latency)
}

func (n *Node) measureLatency(ctx context.Context) float64 {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", "1.1.1.1:53", 2*time.Second)
	if err != nil {
		return 0
	}
	defer conn.Close()

	return float64(time.Since(start).Milliseconds())
}

func (n *Node) GetInfo() NodeInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return NodeInfo{
		Name:      n.config.Hostname,
		LocalIP:   n.localIP,
		IPv6:      n.ipv6,
		TailnetIP: n.tailnetIP,
		Status:    n.status,
		Uptime:    int64(time.Since(n.metrics.uptime).Seconds()),
	}
}

func (n *Node) GetMetrics() MetricsSnapshot {
	return n.metrics.Snapshot()
}

func (n *Node) GetDiagnostics(ctx context.Context) []DiagnosticCheck {
	return n.diagnostics.RunAll(ctx, n)
}

func (n *Node) UpdateFeatures(features FeatureToggles) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.config.Features = features
	return nil
}

func (n *Node) GetFeatures() FeatureToggles {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.config.Features
}

func (n *Node) initTLSCertificate() error {
	certPath := filepath.Join(n.config.StateDir, "cert.pem")
	keyPath := filepath.Join(n.config.StateDir, "key.pem")

	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			return n.loadTLSCertificate(certPath, keyPath)
		}
	}

	return n.generateTLSCertificate(certPath, keyPath)
}

func (n *Node) loadTLSCertificate(certPath, keyPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	if time.Now().After(cert.NotAfter.Add(-30 * 24 * time.Hour)) {
		n.logger.Info("certificate expiring soon, regenerating")
		return n.generateTLSCertificate(certPath, keyPath)
	}

	n.tlsCert = cert
	return nil
}

func (n *Node) generateTLSCertificate(certPath, keyPath string) error {
	n.logger.Info("generating new TLS certificate")

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Autobutler"},
			CommonName:   n.config.Hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{n.config.Hostname, "localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return err
	}

	n.tlsCert = cert
	n.logger.Info("generated new TLS certificate", "valid_until", cert.NotAfter)
	return nil
}

func (n *Node) GetTLSConfig() (*tls.Config, error) {
	certPath := filepath.Join(n.config.StateDir, "cert.pem")
	keyPath := filepath.Join(n.config.StateDir, "key.pem")

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
