package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func GetDataDir() string {
	// Depending on OS, return the appropriate data directory.
	// For example, on Unix-like systems, it could be "/var/lib/autobutler/data",
	// and on Windows, it could be "C:\\ProgramData\\Autobutler\\data", and MacOS could be "~/Library/Application Support/Autobutler/data".

	// switch on os
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get user home directory: %v", err))
	}
	switch os := runtime.GOOS; os {
	case "linux":
		return filepath.Join(homeDir, ".autobutler", "data")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
	default:
		panic(fmt.Sprintf("unsupported OS: %s", os))
	}
}
