package v1_networking

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"autobutler/pkg/headscale"
	"autobutler/pkg/networking"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

// AutoSetup creates a complete local Headscale network with one click
var autoSetupRoute = serverutil.ApiRoute(
	"POST", "/networking/auto-setup", func(c *gin.Context) *serverutil.Response {
		// Check if already configured
		configPath := getConfigPath()
		if existingCfg, _ := networking.LoadConfig(configPath); existingCfg != nil && existingCfg.HeadscaleURL != "" {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusBadRequest).
				WithData(gin.H{
					"error": "Network already configured",
				})
		}

		// Run the setup process
		result, err := performAutoSetup()
		if err != nil {
			return serverutil.NewResponse().
				WithContentType(serverutil.ContentTypeJSON).
				WithStatusCode(http.StatusInternalServerError).
				WithData(gin.H{
					"error": fmt.Sprintf("Setup failed: %v", err),
				})
		}

		return serverutil.NewResponse().
			WithContentType(serverutil.ContentTypeJSON).
			WithStatusCode(http.StatusOK).
			WithData(result)
	},
)

type SetupResult struct {
	HeadscaleURL  string `json:"headscale_url"`
	AuthKey       string `json:"auth_key"`
	NetworkName   string `json:"network_name"`
	NodeHostname  string `json:"node_hostname"`
	Instructions  string `json:"instructions"`
}

func performAutoSetup() (*SetupResult, error) {
	ctx := context.Background()
	
	// 1. Determine local IP address
	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to detect local IP: %w", err)
	}

	// 2. Setup Headscale data directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	dataDir := filepath.Join(homeDir, ".autobutler", "headscale")
	configPath := filepath.Join(dataDir, "config.yaml")
	
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 3. Ensure Headscale binary is available
	headscaleBin, err := ensureHeadscaleBinary(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to setup Headscale binary: %w", err)
	}

	// 4. Initialize Headscale database
	if err := initHeadscaleDB(headscaleBin, dataDir); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// 5. Create Headscale manager and start server
	headscaleURL := fmt.Sprintf("http://%s:8080", localIP)
	mgr := headscale.NewManager(&headscale.Config{
		ServerURL:  headscaleURL,
		ListenAddr: "0.0.0.0:8080",
		DataDir:    dataDir,
		ConfigPath: configPath,
		BinaryPath: headscaleBin,
		LogLevel:   "info",
	})

	if err := mgr.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Headscale: %w", err)
	}

	// Wait for Headscale to be ready
	time.Sleep(2 * time.Second)

	// 6. Create default user
	if err := mgr.CreateUser("home"); err != nil {
		// Ignore error if user already exists
		fmt.Printf("Note: %v\n", err)
	}

	// 7. Generate pre-auth key
	authKey, err := mgr.CreateAuthKey("home", false, 365*24*time.Hour)
	if err != nil {
		mgr.Stop()
		return nil, fmt.Errorf("failed to create auth key: %w", err)
	}

	// 8. Save networking configuration
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "autobutler-node"
	}

	netCfg := &networking.Config{
		HeadscaleURL: headscaleURL,
		AuthKey:      authKey,
		Hostname:     hostname,
		StateDir:     filepath.Join(homeDir, ".autobutler", "networking-state"),
		WebUIPort:    8443,
		Environment:  "Home",
		Features: networking.FeatureToggles{
			AdvertiseLocal: true,
			RemoteTunnel:   false,
			UsageAnalytics: false,
		},
	}

	netConfigPath := getConfigPath()
	if err := netCfg.SaveConfig(netConfigPath); err != nil {
		mgr.Stop()
		return nil, fmt.Errorf("failed to save network config: %w", err)
	}

	// 9. Start the networking node
	node, err := networking.InitNetworkingNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start networking node: %w", err)
	}
	_ = node // Node is now running

	// 10. Return connection instructions
	return &SetupResult{
		HeadscaleURL: headscaleURL,
		AuthKey:      authKey,
		NetworkName:  "home",
		NodeHostname: hostname,
		Instructions: fmt.Sprintf("Download the Tailscale app, then use these settings:\nControl Server: %s\nAuth Key: %s", headscaleURL, authKey),
	}, nil
}

func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "127.0.0.1", nil
}

func ensureHeadscaleBinary(dataDir string) (string, error) {
	// Check if headscale is in PATH
	if binPath, err := exec.LookPath("headscale"); err == nil {
		return binPath, nil
	}

	// Check in data directory
	localBin := filepath.Join(dataDir, "headscale")
	if _, err := os.Stat(localBin); err == nil {
		return localBin, nil
	}

	// TODO: Download and install Headscale binary automatically
	// For now, return error with instructions
	return "", fmt.Errorf("headscale binary not found. Please install with: brew install headscale")
}

func initHeadscaleDB(headscaleBin, dataDir string) error {
	// Check if database already exists
	dbPath := filepath.Join(dataDir, "db.sqlite")
	if _, err := os.Stat(dbPath); err == nil {
		return nil // Already initialized
	}

	// Run headscale database init
	cmd := exec.Command(headscaleBin, "database", "init", "-c", filepath.Join(dataDir, "config.yaml"))
	cmd.Dir = dataDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("database init failed: %w: %s", err, string(output))
	}

	return nil
}
