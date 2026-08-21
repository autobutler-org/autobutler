// Package smb provides utilities for configuring Quark as a Samba
// (SMB/CIFS) network share on Linux, allowing file access directly from
// macOS Finder, Windows Explorer, and Linux file managers on the same LAN.
package smb

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

const (
	shareName     = "quark"
	smbConfigPath = "/etc/samba/smb.conf"
	smbService    = "smbd"
)

var errNotLinux = errors.New("SMB setup is only supported on Linux")

// shareBlock returns the smb.conf stanza for the Quark share.
func shareBlock(filesDir string) string {
	return fmt.Sprintf(`
[%s]
   comment = Quark Files
   path = %s
   writeable = yes
   create mask = 0775
   directory mask = 0775
   public = no
`, shareName, filesDir)
}

// IsLinux returns true when running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsInstalled returns true when the samba package is installed.
func IsInstalled() bool {
	_, err := exec.LookPath("smbpasswd")
	return err == nil
}

// IsConfigured returns true when the Quark share block exists in smb.conf.
func IsConfigured() bool {
	data, err := os.ReadFile(smbConfigPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), fmt.Sprintf("[%s]", shareName))
}

// IsRunning returns true when the smbd service is active.
func IsRunning() bool {
	out, err := exec.Command("systemctl", "is-active", smbService).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// Status reports the current SMB setup state.
type Status struct {
	Linux      bool
	Installed  bool
	Configured bool
	Running    bool
	FilesDir   string
}

// GetStatus returns the current SMB setup state.
func GetStatus() (*Status, error) {
	filesDir, err := storageutil.GetCirrusDir()
	if err != nil {
		filesDir = ""
	}
	return &Status{
		Linux:      IsLinux(),
		Installed:  IsInstalled(),
		Configured: IsConfigured(),
		Running:    IsRunning(),
		FilesDir:   filesDir,
	}, nil
}

// Setup installs and configures Samba for Quark.
// Must be run as root (or via sudo).
func Setup(username, password string) error {
	if !IsLinux() {
		return errNotLinux
	}

	filesDir, err := storageutil.GetCirrusDir()
	if err != nil {
		return fmt.Errorf("failed to locate Quark files directory: %w", err)
	}

	// Ensure the files directory exists.
	if err := os.MkdirAll(filesDir, 0o775); err != nil {
		return fmt.Errorf("failed to create files directory: %w", err)
	}

	// Install samba if not already present.
	if !IsInstalled() {
		fmt.Println("Installing samba...")
		if err := exec.Command("apt-get", "install", "-y", "samba", "samba-common").Run(); err != nil {
			return fmt.Errorf("failed to install samba: %w", err)
		}
	}

	// Append the share block to smb.conf if not already there.
	if !IsConfigured() {
		fmt.Printf("Adding [%s] share to %s...\n", shareName, smbConfigPath)
		f, err := os.OpenFile(smbConfigPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open %s (try running with sudo): %w", smbConfigPath, err)
		}
		if _, err := f.WriteString(shareBlock(filesDir)); err != nil {
			f.Close()
			return fmt.Errorf("failed to write share config: %w", err)
		}
		f.Close()
	}

	// Set the Samba password for the user.
	fmt.Printf("Setting Samba password for user %q...\n", username)
	cmd := exec.Command("smbpasswd", "-a", username, "-s")
	cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set Samba password: %w", err)
	}

	// Enable and restart smbd.
	fmt.Println("Enabling and starting smbd...")
	for _, args := range [][]string{
		{"systemctl", "enable", smbService},
		{"systemctl", "restart", smbService},
	} {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %s: %w", strings.Join(args, " "), err)
		}
	}

	hostname, _ := os.Hostname()
	host := hostname
	if !strings.HasSuffix(host, ".local") {
		host += ".local"
	}
	fmt.Printf("\n✅ Quark SMB share configured.\n")
	fmt.Printf("   macOS:   smb://%s/%s\n", host, shareName)
	fmt.Printf("   Windows: \\\\%s\\%s\n", host, shareName)
	fmt.Printf("   Path:    %s\n", filepath.Join(filesDir))
	return nil
}

// Teardown removes the Quark share from smb.conf and stops the smbd service.
// Must be run as root (or via sudo).
func Teardown() error {
	if !IsLinux() {
		return errNotLinux
	}

	// Remove the share block from smb.conf.
	if IsConfigured() {
		data, err := os.ReadFile(smbConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", smbConfigPath, err)
		}
		block := shareBlock("")
		// Strip any variant of the share block (path may differ), so match on the header.
		lines := strings.Split(string(data), "\n")
		var kept []string
		skip := false
		for _, line := range lines {
			if strings.TrimSpace(line) == fmt.Sprintf("[%s]", shareName) {
				skip = true
			} else if skip && strings.HasPrefix(strings.TrimSpace(line), "[") {
				skip = false
			}
			if !skip {
				kept = append(kept, line)
			}
		}
		_ = block // suppress unused warning
		if err := os.WriteFile(smbConfigPath, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
			return fmt.Errorf("failed to write %s (try running with sudo): %w", smbConfigPath, err)
		}
	}

	// Stop and disable the smbd service.
	for _, args := range [][]string{
		{"systemctl", "stop", smbService},
		{"systemctl", "disable", smbService},
	} {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %s: %w", strings.Join(args, " "), err)
		}
	}

	return nil
}
