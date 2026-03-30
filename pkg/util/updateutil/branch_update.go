package updateutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
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

	secret := os.Getenv("AUTOBUTLER_PROVISIONING_SECRET")
	if secret == "" {
		return fmt.Errorf("AUTOBUTLER_PROVISIONING_SECRET is not set")
	}

	_, err := backupSelf()
	if err != nil {
		return fmt.Errorf("failed to backup current binary: %w", err)
	}

	url := fmt.Sprintf("%s/artifacts/%s/latest", getProvisioningURL(), branch)
	fmt.Println("Downloading branch update from", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Provisioning-Secret", secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download branch artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download branch artifact from %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read branch artifact body: %w", err)
	}

	h := sha256.Sum256(body)
	checksum := hex.EncodeToString(h[:])

	if expectedChecksum := resp.Header.Get("X-Content-SHA256"); expectedChecksum != "" {
		if checksum != expectedChecksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, checksum)
		}
	} else {
		log.Printf("[branch-update] warning: server did not return X-Content-SHA256 header, skipping checksum verification")
	}

	log.Printf("[branch-update] installing binary with SHA256: %s", checksum)

	if err := replaceSelfFromBinary(bytes.NewReader(body)); err != nil {
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
