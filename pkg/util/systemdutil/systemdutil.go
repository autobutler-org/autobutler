package systemdutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const dropInDir = "/etc/systemd/system/autobutler.service.d"
const dropInFile = "branch.conf"

func dropInPath() string {
	return filepath.Join(dropInDir, dropInFile)
}

func SetBranchOverride(branch string) error {
	if branch == "" {
		if err := os.Remove(dropInPath()); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove branch override: %w", err)
		}
		daemonReload()
		return nil
	}

	if err := os.MkdirAll(dropInDir, 0755); err != nil {
		return fmt.Errorf("failed to create drop-in directory: %w", err)
	}

	content := fmt.Sprintf("[Service]\nEnvironment=\"AUTOBUTLER_BRANCH=%s\"\n", branch)
	if err := os.WriteFile(dropInPath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write branch override: %w", err)
	}

	daemonReload()
	return nil
}

func daemonReload() {
	_ = exec.Command("systemctl", "daemon-reload").Run()
}
