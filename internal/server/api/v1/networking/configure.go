package v1_networking

import (
	"net/http"
	"os"
	"path/filepath"

	"autobutler/pkg/networking"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

type ConfigRequest struct {
	HeadscaleURL   string `json:"headscale_url" binding:"required"`
	AuthKey        string `json:"auth_key" binding:"required"`
	Hostname       string `json:"hostname" binding:"required"`
	StateDir       string `json:"state_dir"`
	WebUIPort      int    `json:"webui_port"`
	Environment    string `json:"environment"`
	AdvertiseLocal bool   `json:"advertise_local"`
	RemoteTunnel   bool   `json:"remote_tunnel"`
	UsageAnalytics bool   `json:"usage_analytics"`
}

// GetConfiguration returns the current configuration
var getConfigurationRoute = serverutil.ApiRoute(
	"GET", "/networking/configuration", func(c *gin.Context) *serverutil.Response {
		node := getNodeFromContext(c)

		// Load current config from file
		configPath := getConfigPath()
		cfg, err := networking.LoadConfig(configPath)
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error": "Failed to load configuration",
				})
		}

		// Mask the auth key for security
		response := gin.H{
			"headscale_url":   cfg.HeadscaleURL,
			"auth_key":        maskAuthKey(cfg.AuthKey),
			"hostname":        cfg.Hostname,
			"state_dir":       cfg.StateDir,
			"webui_port":      cfg.WebUIPort,
			"environment":     cfg.Environment,
			"advertise_local": cfg.Features.AdvertiseLocal,
			"remote_tunnel":   cfg.Features.RemoteTunnel,
			"usage_analytics": cfg.Features.UsageAnalytics,
			"configured":      node != nil,
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(response)
	},
)

// UpdateConfiguration saves new configuration and restarts the networking node
var updateConfigurationRoute = serverutil.ApiRoute(
	"POST", "/networking/configuration", func(c *gin.Context) *serverutil.Response {
		var req ConfigRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithData(gin.H{
					"error":   "Invalid configuration data",
					"details": err.Error(),
				})
		}

		// Build the configuration
		cfg := &networking.Config{
			HeadscaleURL: req.HeadscaleURL,
			AuthKey:      req.AuthKey,
			Hostname:     req.Hostname,
			StateDir:     req.StateDir,
			WebUIPort:    req.WebUIPort,
			Environment:  req.Environment,
			Features: networking.FeatureToggles{
				AdvertiseLocal: req.AdvertiseLocal,
				RemoteTunnel:   req.RemoteTunnel,
				UsageAnalytics: req.UsageAnalytics,
			},
		}

		// Set defaults if not provided
		if cfg.StateDir == "" {
			cfg.StateDir = "/var/lib/networking-node"
		}
		if cfg.WebUIPort == 0 {
			cfg.WebUIPort = 8443
		}
		if cfg.Environment == "" {
			cfg.Environment = "Home"
		}

		// Validate the configuration
		if err := cfg.Validate(); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithData(gin.H{
					"error":   "Invalid configuration",
					"details": err.Error(),
				})
		}

		// Save to file
		configPath := getConfigPath()
		if err := cfg.SaveConfig(configPath); err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error":   "Failed to save configuration",
					"details": err.Error(),
				})
		}

		// Return success - node will need to be restarted
		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(gin.H{
				"message":          "Configuration saved successfully. Restart the service to apply changes.",
				"configured":       true,
				"restart_required": true,
			})
	},
)

func getConfigPath() string {
	// Use a config file in the home directory or system config location
	home := os.Getenv("HOME")
	if home == "" {
		home = "/tmp"
	}
	return filepath.Join(home, ".config", "autobutler", "networking.json")
}

func maskAuthKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
