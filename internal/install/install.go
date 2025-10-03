package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func installSystemdService() error {
	serviceFilePath := filepath.Join("/etc/systemd/system", systemdServiceName)
	if err := os.WriteFile(serviceFilePath, []byte(buildServiceFile()), 0644); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}
	if err := exec.Command("systemctl", "start", strings.Split(systemdServiceName, ".")[0]).Run(); err != nil {
		return fmt.Errorf("failed to start systemctl service: %w", err)
	}
	return nil
}

func installPlistService() error {
	serviceFilePath := filepath.Join("/Library/LaunchDaemons", plistServiceName)
	if err := os.WriteFile(serviceFilePath, []byte(buildServiceFile()), 0644); err != nil {
		return fmt.Errorf("failed to write plist service file: %w", err)
	}
	if err := exec.Command("launchctl", "load", serviceFilePath).Run(); err != nil {
		return fmt.Errorf("failed to load plist service: %w", err)
	}
	return nil
}

func Install() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("cp", "-v", executable, "/usr/local/bin/autobutler").Run(); err != nil {
			return fmt.Errorf("failed to copy binary to /usr/local/bin: %w", err)
		}
		return installSystemdService()
	case "darwin":
		if err := exec.Command("cp", "-v", executable, "/Applications/autobutler").Run(); err != nil {
			return fmt.Errorf("failed to copy binary to /Applications: %w", err)
		}
		return installPlistService()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
