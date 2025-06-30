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
	switch runtime.GOOS {
	case "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Assume instead we are running as a system service, and default to a system-wide data directory.
			homeDir = "/var/lib"
		}
		return filepath.Join(homeDir, "autobutler", "data")
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Assume instead we are running as a system service, and default to a system-wide data directory.
			homeDir = "/"
		}
		return filepath.Join(homeDir, "Library", "Application Support", "Autobutler", "data")
	default:
		panic(fmt.Sprintf("unsupported OS: %s", runtime.GOOS))
	}
}
