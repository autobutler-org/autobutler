package settingsutil

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

type settings struct {
	DevMode             bool   `json:"dev_mode"`
	ActiveBranch        string `json:"active_branch,omitempty"`
	RemoteAccessEnabled bool   `json:"remote_access_enabled,omitempty"`
	DeviceID            string `json:"device_id,omitempty"`
}

var mu sync.Mutex

func isRunningAsServiceUser() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	return u.Username == "autobutler"
}

func baseDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "/"
		}
		return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
	case "linux":
		if isRunningAsServiceUser() {
			return "/var/lib/autobutler/data"
		}
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "/var/lib"
		}
		return filepath.Join(homeDir, "autobutler", "data")
	default:
		panic(fmt.Sprintf("unsupported OS: %s", runtime.GOOS))
	}
}

func settingsPath() string {
	return filepath.Join(baseDataDir(), "settings.json")
}

func load() (*settings, error) {
	s := &settings{}
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	return s, nil
}

func save(s *settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(settingsPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath(), data, 0600)
}

func GetDevMode() bool {
	if env := os.Getenv("AUTOBUTLER_DEV_MODE"); env != "" {
		val, err := strconv.ParseBool(env)
		if err == nil {
			return val
		}
	}
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		return false
	}
	return s.DevMode
}

func SetDevMode(enabled bool) error {
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		s = &settings{}
	}
	s.DevMode = enabled
	return save(s)
}

func GetActiveBranch() string {
	if branch := os.Getenv("AUTOBUTLER_BRANCH"); branch != "" {
		return branch
	}
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		return ""
	}
	return s.ActiveBranch
}

func SetActiveBranch(branch string) error {
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		s = &settings{}
	}
	s.ActiveBranch = branch
	return save(s)
}

func GetRemoteAccess() bool {
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		return false
	}
	return s.RemoteAccessEnabled
}

func SetRemoteAccess(enabled bool) error {
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		s = &settings{}
	}
	s.RemoteAccessEnabled = enabled
	return save(s)
}

func GetDeviceID() string {
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		return ""
	}
	return s.DeviceID
}

func SetDeviceID(id string) error {
	mu.Lock()
	defer mu.Unlock()
	s, err := load()
	if err != nil {
		s = &settings{}
	}
	s.DeviceID = id
	return save(s)
}
