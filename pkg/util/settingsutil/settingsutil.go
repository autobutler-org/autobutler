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
	AutoUpdate          bool   `json:"autoUpdate"`
	RemoteAccessEnabled bool   `json:"remoteAccessEnabled"`
	DevMode             bool   `json:"devMode"`
	ActiveBranch        string `json:"activeBranch,omitempty"`
	DeviceID            string `json:"deviceId,omitempty"`
}

var (
	mu     sync.Mutex
	cached *Settings
)

func settingsPath() (string, error) {
	dataDir := storageutil.GetDataDir()
	return filepath.Join(dataDir, settingsFileName), nil
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

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	snapshot := *s
	cached = &snapshot
	return nil
}

// invalidateCache clears the in-memory cache so the next Load() re-reads disk.
// Must be called with mu held.
func invalidateCache() {
	cached = nil
}

// GetAutoUpdate returns whether automatic updates are enabled.
func GetAutoUpdate() bool {
	s, err := Load()
	if err != nil {
		return false
	}
	return s.AutoUpdate
}

// SetAutoUpdate sets the auto-update preference and persists it.
func SetAutoUpdate(enabled bool) error {
	mu.Lock()
	s := cached
	mu.Unlock()

	if s == nil {
		loaded, err := Load()
		if err != nil {
			loaded = &Settings{}
		}
		s = loaded
	}
	s.AutoUpdate = enabled
	return Save(s)
}

// GetRemoteAccess returns whether remote access is enabled.
func GetRemoteAccess() bool {
	s, err := Load()
	if err != nil {
		return false
	}
	return s.RemoteAccessEnabled
}

// SetRemoteAccess sets the remote access enabled flag and persists it.
func SetRemoteAccess(enabled bool) error {
	mu.Lock()
	s := cached
	mu.Unlock()

	if s == nil {
		loaded, err := Load()
		if err != nil {
			loaded = &Settings{}
		}
		s = loaded
	}
	s.RemoteAccessEnabled = enabled
	return Save(s)
}

// GetDeviceID returns the persisted device ID, or empty string if not set.
func GetDeviceID() string {
	s, err := Load()
	if err != nil {
		return ""
	}
	return s.DeviceID
}

// SetDeviceID persists the device ID.
func SetDeviceID(id string) error {
	mu.Lock()
	s := cached
	mu.Unlock()

	if s == nil {
		loaded, err := Load()
		if err != nil {
			loaded = &Settings{}
		}
		s = loaded
	}
	s.DeviceID = id
	return Save(s)
}

// GetActiveBranch returns the active dev branch, or empty string if not set.
func GetActiveBranch() string {
	s, err := Load()
	if err != nil {
		return ""
	}
	return s.ActiveBranch
}

// SetActiveBranch persists the active dev branch.
func SetActiveBranch(branch string) error {
	mu.Lock()
	s := cached
	mu.Unlock()

	if s == nil {
		loaded, err := Load()
		if err != nil {
			loaded = &Settings{}
		}
		s = loaded
	}
	s.ActiveBranch = branch
	return Save(s)
}
