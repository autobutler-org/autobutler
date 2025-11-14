package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sys/unix"
)

func GetDataDir() string {
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

func GetFilesDir() string {
	filesPath := filepath.Join(GetDataDir(), "files")
	if err := os.MkdirAll(filesPath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create files directory: %v", err))
	}
	return filesPath
}

func getAvailableSpaceInBytes(fileDir string) uint64 {
	var stat unix.Statfs_t
	unix.Statfs(fileDir, &stat)
	return stat.Bavail * uint64(stat.Bsize)
}
