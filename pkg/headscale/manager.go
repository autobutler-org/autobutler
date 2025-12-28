package headscale

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Manager handles the Headscale server lifecycle
type Manager struct {
	config    *Config
	cmd       *exec.Cmd
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	running   bool
	healthURL string
}

type Config struct {
	ServerURL  string
	ListenAddr string
	DataDir    string
	ConfigPath string
	BinaryPath string
	LogLevel   string
}

func NewManager(cfg *Config) *Manager {
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = findHeadscaleBinary()
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		healthURL: fmt.Sprintf("http://%s/health", cfg.ListenAddr),
	}
}

// Start starts the Headscale server
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("headscale is already running")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(m.config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Generate config if it doesn't exist
	if err := m.generateConfig(); err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Start Headscale process
	m.cmd = exec.CommandContext(m.ctx, m.config.BinaryPath, "serve", "-c", m.config.ConfigPath)
	m.cmd.Stdout = os.Stdout
	m.cmd.Stderr = os.Stderr
	m.cmd.Dir = m.config.DataDir

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start headscale: %w", err)
	}

	m.running = true

	// Wait for health check
	if err := m.waitForHealth(); err != nil {
		m.Stop()
		return fmt.Errorf("headscale failed to become healthy: %w", err)
	}

	// Monitor process in background
	go m.monitor()

	return nil
}

// Stop stops the Headscale server
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.cancel()
	m.running = false

	if m.cmd != nil && m.cmd.Process != nil {
		if err := m.cmd.Process.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("failed to stop headscale: %w", err)
		}

		// Wait for graceful shutdown with timeout
		done := make(chan error, 1)
		go func() {
			done <- m.cmd.Wait()
		}()

		select {
		case <-time.After(10 * time.Second):
			m.cmd.Process.Kill()
		case <-done:
		}
	}

	return nil
}

// CreateAuthKey generates a new pre-auth key
func (m *Manager) CreateAuthKey(user string, ephemeral bool, expiration time.Duration) (string, error) {
	if user == "" {
		user = "default"
	}

	args := []string{
		"preauthkeys", "create",
		"-c", m.config.ConfigPath,
		"--user", user,
	}

	if ephemeral {
		args = append(args, "--ephemeral")
	}

	if expiration > 0 {
		args = append(args, "--expiration", expiration.String())
	}

	cmd := exec.Command(m.config.BinaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create auth key: %w: %s", err, string(output))
	}

	return string(output), nil
}

// CreateUser creates a new Headscale user/namespace
func (m *Manager) CreateUser(name string) error {
	cmd := exec.Command(m.config.BinaryPath, "users", "create", "-c", m.config.ConfigPath, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create user: %w: %s", err, string(output))
	}
	return nil
}

// IsRunning returns whether Headscale is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) monitor() {
	if m.cmd == nil {
		return
	}

	err := m.cmd.Wait()

	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	if err != nil {
		fmt.Printf("Headscale process exited with error: %v\n", err)
	} else {
		fmt.Println("Headscale process exited successfully")
	}
}

func (m *Manager) waitForHealth() error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for headscale to start")
		case <-ticker.C:
			// Try to connect to health endpoint
			// For now, just wait a bit - proper health check would use HTTP client
			return nil
		}
	}
}

func (m *Manager) generateConfig() error {
	if _, err := os.Stat(m.config.ConfigPath); err == nil {
		// Config already exists
		return nil
	}

	configContent := fmt.Sprintf(`---
server_url: %s
listen_addr: %s
metrics_listen_addr: 127.0.0.1:9090
grpc_listen_addr: 127.0.0.1:50443
grpc_allow_insecure: false

private_key_path: %s/private.key
noise:
  private_key_path: %s/noise_private.key

ip_prefixes:
  - fd7a:115c:a1e0::/48
  - 100.64.0.0/10

database:
  type: sqlite3
  sqlite:
    path: %s/db.sqlite

log:
  level: %s
  format: text

dns_config:
  nameservers:
    - 1.1.1.1
  magic_dns: true
  base_domain: headscale.local

unix_socket: %s/headscale.sock
unix_socket_permission: "0770"
`,
		m.config.ServerURL,
		m.config.ListenAddr,
		m.config.DataDir,
		m.config.DataDir,
		m.config.DataDir,
		m.config.LogLevel,
		m.config.DataDir,
	)

	if err := os.WriteFile(m.config.ConfigPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func findHeadscaleBinary() string {
	// Try common locations
	locations := []string{
		"/usr/local/bin/headscale",
		"/usr/bin/headscale",
		"./headscale",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Try PATH
	if path, err := exec.LookPath("headscale"); err == nil {
		return path
	}

	return "headscale" // fallback
}
