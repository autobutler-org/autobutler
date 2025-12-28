package networking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	HeadscaleURL string         `json:"headscale_url"`
	AuthKey      string         `json:"auth_key"`
	Hostname     string         `json:"hostname"`
	StateDir     string         `json:"state_dir"`
	WebUIPort    int            `json:"webui_port"`
	Features     FeatureToggles `json:"features"`
	Environment  string         `json:"environment"`
}

type FeatureToggles struct {
	AdvertiseLocal bool `json:"advertise_local"`
	RemoteTunnel   bool `json:"remote_tunnel"`
	UsageAnalytics bool `json:"usage_analytics"`
}

func DefaultConfig() *Config {
	return &Config{
		HeadscaleURL: os.Getenv("HEADSCALE_URL"),
		AuthKey:      os.Getenv("HEADSCALE_AUTH_KEY"),
		Hostname:     os.Getenv("NODE_HOSTNAME"),
		StateDir:     "/var/lib/networking-node",
		WebUIPort:    8443,
		Environment:  "Home",
		Features: FeatureToggles{
			AdvertiseLocal: true,
			RemoteTunnel:   true,
			UsageAnalytics: true,
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if stateDir := os.Getenv("STATE_DIR"); stateDir != "" {
		cfg.StateDir = stateDir
	}
	if port := os.Getenv("WEBUI_PORT"); port != "" {
		var p int
		fmt.Sscanf(port, "%d", &p)
		if p > 0 {
			cfg.WebUIPort = p
		}
	}

	return cfg, nil
}

func (c *Config) SaveConfig(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) Validate() error {
	if c.HeadscaleURL == "" {
		return fmt.Errorf("headscale_url is required")
	}
	if c.AuthKey == "" {
		return fmt.Errorf("auth_key is required")
	}
	if c.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if c.WebUIPort <= 0 || c.WebUIPort > 65535 {
		return fmt.Errorf("webui_port must be between 1 and 65535")
	}
	return nil
}
