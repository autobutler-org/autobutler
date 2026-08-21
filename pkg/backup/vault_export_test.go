package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	_ "modernc.org/sqlite"
)

func setupLiveVaultDB(t *testing.T, masterPassword string) (*sql.DB, *db.Queries, []byte) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	_, err = d.Exec(`
		CREATE TABLE vault_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			salt BLOB NOT NULL,
			argon2_memory INTEGER NOT NULL DEFAULT 65536,
			argon2_iterations INTEGER NOT NULL DEFAULT 3,
			argon2_parallelism INTEGER NOT NULL DEFAULT 4,
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
		CREATE INDEX idx_vault_entries_folder ON vault_entries(folder_id);
	`)
	if err != nil {
		t.Fatal(err)
	}

	params := vaultcrypto.Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1}
	salt, _ := vaultcrypto.GenerateSalt()
	key := vaultcrypto.DeriveKey(masterPassword, salt, params)
	verBlob, verNonce, _ := vaultcrypto.MakeVerificationBlob(key)

	_, err = d.Exec(
		`INSERT INTO vault_config (id, salt, argon2_memory, argon2_iterations, argon2_parallelism,
			verification_blob, verification_nonce, auto_lock_seconds)
		VALUES (1, ?, ?, ?, ?, ?, ?, 900)`,
		salt, params.Memory, params.Iterations, params.Parallelism, verBlob, verNonce,
	)
	if err != nil {
		t.Fatal(err)
	}

	queries := db.New(d)
	return d, queries, key
}

func addTestEntry(t *testing.T, d *sql.DB, key []byte, name, username, password string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{
		"url":      "https://example.com",
		"username": username,
		"password": password,
	})
	ct, nonce, _ := vaultcrypto.Encrypt(key, payload)
	_, err := d.Exec(
		`INSERT INTO vault_entries (name, url_host, ciphertext, nonce) VALUES (?, ?, ?, ?)`,
		name, "example.com", ct, nonce,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func addTestFolder(t *testing.T, d *sql.DB, name string) {
	t.Helper()
	_, err := d.Exec(`INSERT INTO vault_folders (name, sort_order) VALUES (?, 0)`, name)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExportVault_RoundTrip(t *testing.T) {
	masterPw := "master-password-123"
	recoveryPw := "recovery-password-456"

	d, queries, liveKey := setupLiveVaultDB(t, masterPw)
	defer vaultcrypto.ZeroKey(liveKey)

	addTestFolder(t, d, "Social")
	addTestFolder(t, d, "Banking")
	addTestEntry(t, d, liveKey, "GitHub", "alice", "gh-secret")
	addTestEntry(t, d, liveKey, "Gmail", "bob", "gm-secret")

	targetDir := t.TempDir()
	dbPath, err := ExportVault(context.Background(), queries, liveKey, recoveryPw, targetDir)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("backup vault DB not found: %v", err)
	}

	// Open backup DB and verify.
	backupDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backupDB.Close()

	// Verify vault_config exists and recovery password works.
	var salt []byte
	var argMem, argIter, argPar int64
	var vBlob, vNonce []byte
	err = backupDB.QueryRow(
		`SELECT salt, argon2_memory, argon2_iterations, argon2_parallelism,
			verification_blob, verification_nonce FROM vault_config WHERE id = 1`,
	).Scan(&salt, &argMem, &argIter, &argPar, &vBlob, &vNonce)
	if err != nil {
		t.Fatalf("query backup vault_config: %v", err)
	}

	recoveryParams := vaultcrypto.Argon2Params{
		Memory:      uint32(argMem),
		Iterations:  uint32(argIter),
		Parallelism: uint8(argPar),
	}
	recoveryKey := vaultcrypto.DeriveKey(recoveryPw, salt, recoveryParams)
	defer vaultcrypto.ZeroKey(recoveryKey)

	if !vaultcrypto.CheckVerificationBlob(recoveryKey, vBlob, vNonce) {
		t.Fatal("recovery password should verify against backup vault")
	}

	// Master password should NOT work on the backup.
	masterKey := vaultcrypto.DeriveKey(masterPw, salt, recoveryParams)
	if vaultcrypto.CheckVerificationBlob(masterKey, vBlob, vNonce) {
		t.Fatal("master password should NOT verify against backup vault")
	}
	vaultcrypto.ZeroKey(masterKey)

	// Verify entries can be decrypted with recovery key.
	rows, err := backupDB.Query(`SELECT name, ciphertext, nonce FROM vault_entries ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type decryptedEntry struct {
		name     string
		username string
		password string
	}
	var got []decryptedEntry
	for rows.Next() {
		var name string
		var ct, nonce []byte
		if err := rows.Scan(&name, &ct, &nonce); err != nil {
			t.Fatal(err)
		}
		plaintext, err := vaultcrypto.Decrypt(recoveryKey, ct, nonce)
		if err != nil {
			t.Fatalf("decrypt backup entry %q: %v", name, err)
		}
		var payload map[string]string
		json.Unmarshal(plaintext, &payload)
		got = append(got, decryptedEntry{name, payload["username"], payload["password"]})
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].name != "GitHub" || got[0].password != "gh-secret" {
		t.Errorf("entry 0: got %+v", got[0])
	}
	if got[1].name != "Gmail" || got[1].password != "gm-secret" {
		t.Errorf("entry 1: got %+v", got[1])
	}

	// Verify folders were copied.
	var folderCount int
	backupDB.QueryRow(`SELECT COUNT(*) FROM vault_folders`).Scan(&folderCount)
	if folderCount != 2 {
		t.Errorf("expected 2 folders, got %d", folderCount)
	}
}

func TestExportVault_EmptyVault(t *testing.T) {
	_, queries, liveKey := setupLiveVaultDB(t, "password")
	defer vaultcrypto.ZeroKey(liveKey)

	targetDir := t.TempDir()
	dbPath, err := ExportVault(context.Background(), queries, liveKey, "recovery", targetDir)
	if err != nil {
		t.Fatalf("ExportVault failed: %v", err)
	}

	backupDB, _ := sql.Open("sqlite", dbPath)
	defer backupDB.Close()

	var entryCount int
	backupDB.QueryRow(`SELECT COUNT(*) FROM vault_entries`).Scan(&entryCount)
	if entryCount != 0 {
		t.Errorf("expected 0 entries, got %d", entryCount)
	}
}

func TestExportVault_WrongLiveKey(t *testing.T) {
	_, queries, liveKey := setupLiveVaultDB(t, "correct-password")
	vaultcrypto.ZeroKey(liveKey)

	wrongKey := vaultcrypto.DeriveKey("wrong", []byte("saltsaltsaltsalt"),
		vaultcrypto.Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})

	targetDir := t.TempDir()
	_, err := ExportVault(context.Background(), queries, wrongKey, "recovery", targetDir)
	if err == nil {
		t.Fatal("expected error with wrong live key")
	}
}

func TestBackupVaultChecksum(t *testing.T) {
	_, queries, liveKey := setupLiveVaultDB(t, "password")
	defer vaultcrypto.ZeroKey(liveKey)

	targetDir := t.TempDir()
	dbPath, _ := ExportVault(context.Background(), queries, liveKey, "recovery", targetDir)

	checksum, err := BackupVaultChecksum(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(checksum) != 64 {
		t.Errorf("expected 64-char hex sha256, got %d chars", len(checksum))
	}

	// Running again should produce same checksum for same file.
	checksum2, _ := BackupVaultChecksum(dbPath)
	if checksum != checksum2 {
		t.Error("checksum should be deterministic for the same file")
	}
}
