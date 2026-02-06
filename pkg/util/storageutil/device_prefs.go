package storageutil

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

// DevicePrefs holds persisted preferences about devices (e.g., which are backups)
type DevicePrefs struct {
	BackupDevices map[string]bool `json:"backupDevices,omitempty"`
	DefaultBackup string          `json:"defaultBackup,omitempty"`
}

func prefsFilePath() string {
	return filepath.Join(GetDataDir(), "device_prefs.json")
}

// LoadDevicePrefs returns persisted device preferences; missing file yields empty prefs
func LoadDevicePrefs() (DevicePrefs, error) {
	var p DevicePrefs
	p.BackupDevices = make(map[string]bool)
	path := prefsFilePath()
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	if p.BackupDevices == nil {
		p.BackupDevices = make(map[string]bool)
	}
	return p, nil
}

// SaveDevicePrefs persists device preferences to disk
func SaveDevicePrefs(p DevicePrefs) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(prefsFilePath(), b, 0644)
}

// SetDeviceBackup marks/unmarks a device serial as a backup device
func SetDeviceBackup(serial string, isBackup bool) error {
	p, err := LoadDevicePrefs()
	if err != nil {
		return err
	}
	if p.BackupDevices == nil {
		p.BackupDevices = make(map[string]bool)
	}
	if isBackup {
		p.BackupDevices[serial] = true
	} else {
		delete(p.BackupDevices, serial)
	}
	return SaveDevicePrefs(p)
}

// IsBackup checks whether a serial is recorded as a backup device
func (p DevicePrefs) IsBackup(serial string) bool {
	if p.BackupDevices == nil {
		return false
	}
	return p.BackupDevices[serial]
}
