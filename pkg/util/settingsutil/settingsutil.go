package settingsutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

const settingsFileName = "settings.json"

// Settings holds application-level user-configurable settings.
type Settings struct {
	AutoUpdate bool `json:"autoUpdate"`
}

var (
	mu     sync.Mutex
	cached *Settings
)

func settingsPath() (string, error) {
	dataDir, err := storageutil.GetCirrusDir()
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}
	// Store settings.json alongside cirrus dir, in the parent data dir.
	return filepath.Join(filepath.Dir(dataDir), settingsFileName), nil
}

// Load reads settings from disk (or returns defaults if not present).
// The result is cached for the lifetime of the process.
// Returns a copy of the cached settings to prevent callers from mutating
// the shared state without holding the lock.
func Load() (*Settings, error) {
	mu.Lock()
	defer mu.Unlock()

	if cached != nil {
		copy := *cached
		return &copy, nil
	}

	path, err := settingsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cached = &Settings{}
		copy := *cached
		return &copy, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	s := &Settings{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	cached = s
	copy := *cached
	return &copy, nil
}

// Save writes settings to disk and updates the in-process cache.
// Stores a copy so the caller's pointer cannot mutate the cache.
func Save(s *Settings) error {
	mu.Lock()
	defer mu.Unlock()

	path, err := settingsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	snapshot := *s
	cached = &snapshot
	return nil
}

// GetAutoUpdate returns whether automatic updates are enabled.
// Defaults to false if settings cannot be loaded.
func GetAutoUpdate() bool {
	s, err := Load()
	if err != nil {
		return false
	}
	return s.AutoUpdate
}

// SetAutoUpdate sets the auto-update preference and persists it.
func SetAutoUpdate(enabled bool) error {
	s, err := Load()
	if err != nil {
		s = &Settings{}
	}
	s.AutoUpdate = enabled
	return Save(s)
}
