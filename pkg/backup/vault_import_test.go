package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	_ "modernc.org/sqlite"
)

func setupLiveAndBackup(t *testing.T) (liveDB *sql.DB, liveQueries *db.Queries, liveKey []byte, backupDir string) {
	t.Helper()

	masterPw := "master-pw"
	recoveryPw := "recovery-pw"

	d, q, key := setupLiveVaultDB(t, masterPw)
	addTestFolder(t, d, "Social")
	addTestEntry(t, d, key, "GitHub", "alice", "gh-pass")
	addTestEntry(t, d, key, "Gmail", "bob", "gm-pass")

	backupDir = t.TempDir()
	_, err := ExportVault(context.Background(), q, key, recoveryPw, backupDir)
	if err != nil {
		t.Fatalf("ExportVault: %v", err)
	}

	// Create a fresh live DB to import into (simulating a new/empty vault).
	freshDir := t.TempDir()
	freshPath := filepath.Join(freshDir, "fresh.db")
	freshDB, err := sql.Open("sqlite", freshPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { freshDB.Close() })

	params := vaultcrypto.Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1}
	salt, _ := vaultcrypto.GenerateSalt()
	freshKey := vaultcrypto.DeriveKey("new-master", salt, params)
	verBlob, verNonce, _ := vaultcrypto.MakeVerificationBlob(freshKey)

	_, err = freshDB.Exec(`
		CREATE TABLE vault_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			salt BLOB NOT NULL,
			argon2_memory INTEGER NOT NULL,
			argon2_iterations INTEGER NOT NULL,
			argon2_parallelism INTEGER NOT NULL,
			verification_blob BLOB NOT NULL,
			verification_nonce BLOB NOT NULL,
			auto_lock_seconds INTEGER NOT NULL DEFAULT 900,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE vault_folders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			parent_id INTEGER REFERENCES vault_folders(id) ON DELETE CASCADE,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
		CREATE TABLE vault_entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			url_host TEXT NOT NULL DEFAULT '',
			folder_id INTEGER REFERENCES vault_folders(id) ON DELETE SET NULL,
			ciphertext BLOB NOT NULL,
			nonce BLOB NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = freshDB.Exec(
		`INSERT INTO vault_config (id, salt, argon2_memory, argon2_iterations, argon2_parallelism,
			verification_blob, verification_nonce) VALUES (1, ?, ?, ?, ?, ?, ?)`,
		salt, params.Memory, params.Iterations, params.Parallelism, verBlob, verNonce,
	)
	if err != nil {
		t.Fatal(err)
	}

	return freshDB, db.New(freshDB), freshKey, backupDir
}

func TestImportVault_RoundTrip(t *testing.T) {
	_, liveQ, liveKey, backupDir := setupLiveAndBackup(t)
	defer vaultcrypto.ZeroKey(liveKey)

	result, err := ImportVault(context.Background(), liveQ, liveKey, "recovery-pw", backupDir)
	if err != nil {
		t.Fatalf("ImportVault: %v", err)
	}

	if result.EntriesImported != 2 {
		t.Errorf("expected 2 entries imported, got %d", result.EntriesImported)
	}
	if result.FoldersImported != 1 {
		t.Errorf("expected 1 folder imported, got %d", result.FoldersImported)
	}

	// Verify we can decrypt imported entries with the live key.
	entries, err := liveQ.ListAllVaultEntriesForReEncrypt(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries in live vault, got %d", len(entries))
	}

	for _, e := range entries {
		plaintext, err := vaultcrypto.Decrypt(liveKey, e.Ciphertext, e.Nonce)
		if err != nil {
			t.Fatalf("decrypt imported entry %d: %v", e.ID, err)
		}
		var payload map[string]string
		json.Unmarshal(plaintext, &payload)
		if payload["username"] == "" || payload["password"] == "" {
			t.Errorf("entry %d has empty credentials: %v", e.ID, payload)
		}
	}
}

func TestImportVault_SkipDuplicates(t *testing.T) {
	liveDB, liveQ, liveKey, backupDir := setupLiveAndBackup(t)
	defer vaultcrypto.ZeroKey(liveKey)

	// Pre-populate with a matching entry name.
	payload, _ := json.Marshal(map[string]string{"url": "https://github.com", "username": "existing", "password": "pw"})
	ct, nonce, _ := vaultcrypto.Encrypt(liveKey, payload)
	liveDB.Exec(`INSERT INTO vault_entries (name, url_host, ciphertext, nonce) VALUES (?, ?, ?, ?)`,
		"GitHub", "github.com", ct, nonce)

	// Pre-populate with a matching folder name.
	liveDB.Exec(`INSERT INTO vault_folders (name, sort_order) VALUES ('Social', 0)`)

	result, err := ImportVault(context.Background(), liveQ, liveKey, "recovery-pw", backupDir)
	if err != nil {
		t.Fatalf("ImportVault: %v", err)
	}

	if result.EntriesImported != 1 {
		t.Errorf("expected 1 entry imported (Gmail), got %d", result.EntriesImported)
	}
	if result.EntriesSkipped != 1 {
		t.Errorf("expected 1 entry skipped (GitHub), got %d", result.EntriesSkipped)
	}
	if result.FoldersImported != 0 {
		t.Errorf("expected 0 folders imported, got %d", result.FoldersImported)
	}
	if result.FoldersSkipped != 1 {
		t.Errorf("expected 1 folder skipped, got %d", result.FoldersSkipped)
	}
}

func TestImportVault_WrongRecoveryPassword(t *testing.T) {
	_, liveQ, liveKey, backupDir := setupLiveAndBackup(t)
	defer vaultcrypto.ZeroKey(liveKey)

	_, err := ImportVault(context.Background(), liveQ, liveKey, "wrong-password", backupDir)
	if err == nil {
		t.Fatal("expected error with wrong recovery password")
	}
}

func TestImportVault_NoBackupFile(t *testing.T) {
	_, liveQ, liveKey, _ := setupLiveAndBackup(t)
	defer vaultcrypto.ZeroKey(liveKey)

	_, err := ImportVault(context.Background(), liveQ, liveKey, "recovery-pw", t.TempDir())
	if err == nil {
		t.Fatal("expected error when vault_backup.db is missing")
	}
}
