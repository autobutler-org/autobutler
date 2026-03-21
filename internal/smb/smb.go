// Package smb provides utilities for configuring AutoButler as a Samba
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

	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

const (
	shareName     = "autobutler"
	smbConfigPath = "/etc/samba/smb.conf"
	smbService    = "smbd"
)

var errNotLinux = errors.New("SMB setup is only supported on Linux")

// shareBlock returns the smb.conf stanza for the AutoButler share.
func shareBlock(filesDir string) string {
	return fmt.Sprintf(`
[%s]
   comment = AutoButler Files
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

// IsConfigured returns true when the AutoButler share block exists in smb.conf.
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

// Setup installs and configures Samba for AutoButler.
// Must be run as root (or via sudo).
func Setup(username, password string) error {
	if !IsLinux() {
		return errNotLinux
	}

	filesDir, err := storageutil.GetCirrusDir()
	if err != nil {
		return fmt.Errorf("failed to locate AutoButler files directory: %w", err)
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
	fmt.Printf("\n✅ AutoButler SMB share configured.\n")
	fmt.Printf("   macOS:   smb://%s/%s\n", host, shareName)
	fmt.Printf("   Windows: \\\\%s\\%s\n", host, shareName)
	fmt.Printf("   Path:    %s\n", filepath.Join(filesDir))
	return nil
}
