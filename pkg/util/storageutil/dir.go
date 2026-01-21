package storageutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func ConstructCirrusDir(dataDir string) string {
	return filepath.Join(dataDir, "cirrus")
}

func GetCirrusDir() string {
	// TODO: Probably should not panic here
	cirrusPath := ConstructCirrusDir(GetDataDir())
	if err := os.MkdirAll(cirrusPath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create cirrus directory: %v", err)) // coverage: ignore - panic on filesystem error
	}
	return cirrusPath
}

func GetCirrusDirForDevice(mountPoint string) string {
	// TODO: Probably should not panic here
	dataDir := GetDataDirForDevice(mountPoint)
	cirrusPath := ConstructCirrusDir(dataDir)
	if err := os.MkdirAll(cirrusPath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create cirrus directory for device at %s: %v", mountPoint, err)) // coverage: ignore - panic on filesystem error
	}
	return cirrusPath
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
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = "/var/lib" // coverage: ignore - requires UserHomeDir to fail
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

func GetMountsDir() string {
	mountDir := filepath.Join(GetDataDir(), "mounts")
	// TODO: Probably should not panic here
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create mount directory: %v", err)) // coverage: ignore - panic on filesystem error
	}
	return mountDir
}
