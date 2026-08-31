package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// The checksum used to read the whole DB file into memory to hash it (#1723).
// Streaming has to produce exactly the same digest — a wrong hash here would
// silently invalidate every backup.
func TestBackupVaultChecksumMatchesTheWholeFileHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.db")
	// Larger than the io.Copy buffer, so the streaming path takes more than one
	// read and a mishandled loop would show up.
	content := make([]byte, 200*1024)
	for i := range content {
		content[i] = byte(i % 251)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := BackupVaultChecksum(path)
	if err != nil {
		t.Fatalf("BackupVaultChecksum: %v", err)
	}
	want := sha256.Sum256(content)
	if got != hex.EncodeToString(want[:]) {
		t.Errorf("checksum: got %s, want %s", got, hex.EncodeToString(want[:]))
	}
}

func TestBackupVaultChecksumReportsAMissingFile(t *testing.T) {
	if _, err := BackupVaultChecksum(filepath.Join(t.TempDir(), "absent.db")); err == nil {
		t.Error("a missing file must be an error, not an empty checksum")
	}
}
