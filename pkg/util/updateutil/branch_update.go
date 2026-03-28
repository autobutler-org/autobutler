package updateutil

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const defaultProvisioningURL = "http://165.227.215.101:8081"

func getProvisioningURL() string {
	if url := os.Getenv("AUTOBUTLER_PROVISIONING_URL"); url != "" {
		return url
	}
	return defaultProvisioningURL
}

func UpdateFromBranch(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch cannot be empty")
	}

	_, err := backupSelf()
	if err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	url := fmt.Sprintf("%s/artifacts/%s/latest", getProvisioningURL(), branch)
	fmt.Println("Downloading branch update from", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download branch artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download branch artifact from %s: %s", url, resp.Status)
	}

	if err := replaceSelfFromBinary(resp.Body); err != nil {
		return fmt.Errorf("failed to replace self with branch artifact: %w", err)
	}

	fmt.Println("Branch update successful.")
	return nil
}

func replaceSelfFromBinary(body io.Reader) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "autobutler_branch_update_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, body); err != nil {
		return fmt.Errorf("failed to write binary to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	execDir := execPath[:strings.LastIndex(execPath, "/")]
	tmpNew, err := os.CreateTemp(execDir, ".autobutler_new_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file in target directory: %w", err)
	}
	tmpNewPath := tmpNew.Name()
	defer os.Remove(tmpNewPath)

	src, err := os.Open(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open downloaded binary: %w", err)
	}
	defer src.Close()

	if _, err := io.Copy(tmpNew, src); err != nil {
		tmpNew.Close()
		return fmt.Errorf("failed to copy new binary: %w", err)
	}

	if err := tmpNew.Sync(); err != nil {
		tmpNew.Close()
		return fmt.Errorf("failed to sync new binary: %w", err)
	}
	tmpNew.Close()

	if err := os.Chmod(tmpNewPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on new binary: %w", err)
	}

	if err := os.Rename(tmpNewPath, execPath); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	return nil
}
