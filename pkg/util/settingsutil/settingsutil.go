package settingsutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

type settings struct {
	DevMode bool `json:"dev_mode"`
}

func settingsPath() string {
	return filepath.Join(storageutil.GetDataDir(), "settings.json")
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
	return os.WriteFile(settingsPath(), data, 0644)
}

func GetDevMode() bool {
	if env := os.Getenv("AUTOBUTLER_DEV_MODE"); env != "" {
		val, err := strconv.ParseBool(env)
		if err == nil {
			return val
		}
	}
	s, err := load()
	if err != nil {
		return false
	}
	return s.DevMode
}

func SetDevMode(enabled bool) error {
	s, err := load()
	if err != nil {
		s = &settings{}
	}
	s.DevMode = enabled
	return save(s)
}
