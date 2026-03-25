package storageutil

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

func getServiceDataDir() string {
	return "/var/lib/autobutler/data"
}

func isRunningAsServiceUser() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	return u.Username == "autobutler"
}

func ConstructCirrusDir(dataDir string) string {
	return filepath.Join(dataDir, "cirrus")
}

func GetCirrusDir() (string, error) {
	cirrusPath := ConstructCirrusDir(GetDataDir())
	if err := os.MkdirAll(cirrusPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cirrus directory: %w", err)
	}
	return cirrusPath, nil
}

func GetCirrusDirForDevice(mountPoint string) (string, error) {
	if mountPoint == "" {
		return "", fmt.Errorf("mount point is empty")
	}

	mountInfo, err := os.Stat(mountPoint)
	if err != nil {
		return "", err
	}
	if !mountInfo.IsDir() {
		return "", fmt.Errorf("mount point is not a directory")
	}

	dataDir := GetDataDirForDevice(mountPoint)
	cirrusPath := ConstructCirrusDir(dataDir)
	if err := os.MkdirAll(cirrusPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cirrus directory for device at %s: %v", mountPoint, err)
	}
	return cirrusPath, nil
}

// GetDataDirForDevice returns the data directory path for a specific device mount point
func GetDataDirForDevice(mountPoint string) string {
	// For the main system device (root filesystem or /System/Volumes/Data on macOS),
	// use the standard user-specific data directory location
	isSystemDevice := mountPoint == "/" || mountPoint == "/System/Volumes/Data"

	if isSystemDevice {
		// Use platform-specific user directories (~/Library/Application Support/Autobutler/data on macOS)
		switch runtime.GOOS {
		case "darwin": // coverage: ignore - Not run in CI
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/" // coverage: ignore - requires UserHomeDir to fail
			}
			return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
		case "linux": // coverage: ignore - Not run in mac dev environments
			if isRunningAsServiceUser() {
				return getServiceDataDir() // coverage: ignore
			}
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/var/lib" // coverage: ignore
			}
			return filepath.Join(homeDir, "autobutler", "data") // coverage: ignore
		}
	}

	// For external devices, use autobutler directory on the device itself
	return filepath.Join(mountPoint, "autobutler", "data")
}

func GetDataDir() string {
	// Get data directory for the system device
	switch runtime.GOOS {
	case "darwin": // coverage: ignore - Not run in CI
		// On macOS, the system mount point can be / or /System/Volumes/Data
		// Use / as the canonical reference
		return GetDataDirForDevice("/")
	case "linux": // coverage: ignore - Not run in mac dev environments
		return GetDataDirForDevice("/")
	default:
		panic(fmt.Sprintf("unsupported OS: %s", runtime.GOOS)) // coverage: ignore - panic on unsupported OS
	}
}

func GetMountsDir() (string, error) {
	mountDir := filepath.Join(GetDataDir(), "mounts")
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create mounts directory: %w", err)
	}
	return mountDir, nil
}
