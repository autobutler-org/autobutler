package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	_ "modernc.org/sqlite"
)

const backupVaultFilename = "vault_backup.db"

func ExportVault(ctx context.Context, queries *db.Queries, liveKey []byte, recoveryPassword string, targetDir string) (string, error) {
	config, err := queries.GetVaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("get vault config: %w", err)
	}

	if !vaultcrypto.CheckVerificationBlob(liveKey, config.VerificationBlob, config.VerificationNonce) {
		return "", errors.New("live vault key verification failed")
	}

	newSalt, err := vaultcrypto.GenerateSalt()
	if err != nil {
		return "", fmt.Errorf("generate recovery salt: %w", err)
	}

	recoveryParams := vaultcrypto.DefaultParams()
	recoveryKey := vaultcrypto.DeriveKey(recoveryPassword, newSalt, recoveryParams)
	defer vaultcrypto.ZeroKey(recoveryKey)

	verBlob, verNonce, err := vaultcrypto.MakeVerificationBlob(recoveryKey)
	if err != nil {
		return "", fmt.Errorf("make recovery verification blob: %w", err)
	}

	dbPath := filepath.Join(targetDir, backupVaultFilename)
	os.Remove(dbPath)

	backupDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", fmt.Errorf("open backup vault db: %w", err)
	}
	defer backupDB.Close()

	if err := createBackupVaultSchema(backupDB); err != nil {
		return "", fmt.Errorf("create backup schema: %w", err)
	}

	_, err = backupDB.ExecContext(ctx,
		`INSERT INTO vault_config (id, salt, argon2_memory, argon2_iterations, argon2_parallelism,
			verification_blob, verification_nonce, auto_lock_seconds)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)`,
		newSalt, recoveryParams.Memory, recoveryParams.Iterations, recoveryParams.Parallelism,
		verBlob, verNonce, config.AutoLockSeconds,
	)
	if err != nil {
		return "", fmt.Errorf("insert backup vault config: %w", err)
	}

	folders, err := queries.ListVaultFolders(ctx)
	if err != nil {
		return "", fmt.Errorf("list vault folders: %w", err)
	}
	for _, f := range folders {
		_, err := backupDB.ExecContext(ctx,
			`INSERT INTO vault_folders (id, name, parent_id, sort_order, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			f.ID, f.Name, f.ParentID, f.SortOrder, f.CreatedAt,
		)
		if err != nil {
			return "", fmt.Errorf("insert backup folder %d: %w", f.ID, err)
		}
	}

	entries, err := queries.ListAllVaultEntriesForReEncrypt(ctx)
	if err != nil {
		return "", fmt.Errorf("list vault entries: %w", err)
	}

	fullEntries := make([]db.VaultEntry, 0, len(entries))
	for _, e := range entries {
		full, err := queries.GetVaultEntry(ctx, e.ID)
		if err != nil {
			return "", fmt.Errorf("get full entry %d: %w", e.ID, err)
		}
		fullEntries = append(fullEntries, full)
	}

	for _, entry := range fullEntries {
		plaintext, err := vaultcrypto.Decrypt(liveKey, entry.Ciphertext, entry.Nonce)
		if err != nil {
			return "", fmt.Errorf("decrypt entry %d: %w", entry.ID, err)
		}

		newCiphertext, newNonce, err := vaultcrypto.Encrypt(recoveryKey, plaintext)
		if err != nil {
			return "", fmt.Errorf("re-encrypt entry %d: %w", entry.ID, err)
		}

		_, err = backupDB.ExecContext(ctx,
			`INSERT INTO vault_entries (id, name, url_host, folder_id, ciphertext, nonce, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID, entry.Name, entry.UrlHost, entry.FolderID,
			newCiphertext, newNonce, entry.CreatedAt, entry.UpdatedAt,
		)
		if err != nil {
			return "", fmt.Errorf("insert backup entry %d: %w", entry.ID, err)
		}
	}

	return dbPath, nil
}

// BackupVaultChecksum returns the hex-encoded SHA-256 of the backup vault DB file.
func BackupVaultChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

func createBackupVaultSchema(d *sql.DB) error {
	return db.InitVaultSchema(d)
}
