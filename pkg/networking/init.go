package networking

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

func InitNetworkingNode(ctx context.Context) (*Node, error) {
	// Try home config first, then system config
	home := os.Getenv("HOME")
	configPaths := []string{}
	if home != "" {
		configPaths = append(configPaths, home+"/.config/autobutler/networking.json")
	}
	configPaths = append(configPaths, "/etc/autobutler/networking.json")
	if envPath := os.Getenv("NETWORKING_CONFIG_PATH"); envPath != "" {
		configPaths = []string{envPath}
	}

	var cfg *Config
	var err error
	for _, path := range configPaths {
		cfg, err = LoadConfig(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.HeadscaleURL == "" || cfg.AuthKey == "" {
		slog.Info("networking node not configured, skipping initialization")
		return nil, nil
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	node, err := NewNode(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create networking node: %w", err)
	}

	if err := node.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start networking node: %w", err)
	}

	logger.Info("networking node initialized successfully")
	return node, nil
}
