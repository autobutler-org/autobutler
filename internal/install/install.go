package install

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

func installSystemdService() error {
	serviceFilePath := filepath.Join("/etc/systemd/system", systemdServiceName)
	if err := os.WriteFile(serviceFilePath, []byte(buildServiceFile()), 0644); err != nil {
		return fmt.Errorf("failed to write systemd service file: %w", err)
	}
	// Reload systemd to recognize the new service
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	// Enable the service to start on boot
	if err := exec.Command("systemctl", "enable", strings.Split(systemdServiceName, ".")[0]).Run(); err != nil {
		return fmt.Errorf("failed to enable systemctl service: %w", err)
	}
	// Start the service immediately
	if err := exec.Command("systemctl", "restart", strings.Split(systemdServiceName, ".")[0]).Run(); err != nil {
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

const sudoersDropInPath = "/etc/sudoers.d/quark"

func createServiceUser() error {
	if _, err := user.Lookup(serviceUserName); err == nil {
		return nil
	}
	return exec.Command(
		"useradd",
		"--system",
		"--no-create-home",
		"--shell", "/usr/sbin/nologin",
		"--comment", "Quark service account",
		serviceUserName,
	).Run()
}

func createServiceDataDir() error {
	if err := os.MkdirAll(serviceDataDir, 0750); err != nil {
		return fmt.Errorf("failed to create service data dir: %w", err)
	}
	svcUser, err := user.Lookup(serviceUserName)
	if err != nil {
		return fmt.Errorf("failed to look up service user: %w", err)
	}
	return exec.Command("chown", "-R",
		fmt.Sprintf("%s:%s", svcUser.Uid, svcUser.Gid),
		serviceDataDir,
	).Run()
}

func installSudoersRule() error {
	mountsDir := filepath.Join(serviceDataDir, "data", "mounts")
	content := fmt.Sprintf(
		"%s ALL=(root) NOPASSWD: /bin/mount * %s/*, /bin/umount %s/*\n",
		serviceUserName, mountsDir, mountsDir,
	)
	return os.WriteFile(sudoersDropInPath, []byte(content), 0440)
}

func Install() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("cp", "-v", executable, "/usr/local/bin/quark").Run(); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}
		if err := createServiceUser(); err != nil {
			return fmt.Errorf("failed to create service user: %w", err)
		}
		if err := createServiceDataDir(); err != nil {
			return fmt.Errorf("failed to create service data directory: %w", err)
		}
		if err := installSudoersRule(); err != nil {
			return fmt.Errorf("failed to install sudoers rule: %w", err)
		}
		if err := installSudoersRule(); err != nil {
			return err
		}
		return installSystemdService()
	case "darwin": // coverage: ignore - Not run in CI
		if err := exec.Command("cp", "-v", executable, "/Applications/quark").Run(); err != nil {
			return fmt.Errorf("failed to copy binary to /Applications: %w", err)
		}
		return installPlistService()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
