package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	// Registers the "sqlite" driver with database/sql.
	_ "modernc.org/sqlite"
)

func ImportVault(ctx context.Context, liveTx *db.Queries, liveKey []byte, recoveryPassword string, backupDir string) (*VaultImportResult, error) {
	dbPath := filepath.Join(backupDir, backupVaultFilename)
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("vault_backup.db not found on device: %w", err)
	}

	backupDB, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open backup vault: %w", err)
	}
	defer backupDB.Close()

	var salt []byte
	var argMem, argIter, argPar int64
	var vBlob, vNonce []byte
	err = backupDB.QueryRowContext(ctx,
		`SELECT salt, argon2_memory, argon2_iterations, argon2_parallelism,
			verification_blob, verification_nonce FROM vault_config WHERE id = 1`,
	).Scan(&salt, &argMem, &argIter, &argPar, &vBlob, &vNonce)
	if err != nil {
		return nil, fmt.Errorf("read backup vault config: %w", err)
	}

	recoveryParams := vaultcrypto.Argon2Params{
		Memory:      uint32(argMem),
		Iterations:  uint32(argIter),
		Parallelism: uint8(argPar),
	}
	recoveryKey := vaultcrypto.DeriveKey(recoveryPassword, salt, recoveryParams)
	defer vaultcrypto.ZeroKey(recoveryKey)

	if !vaultcrypto.CheckVerificationBlob(recoveryKey, vBlob, vNonce) {
		return nil, errors.New("incorrect recovery password")
	}

	result := &VaultImportResult{}

	// Build a set of existing folder names for dedup.
	existingFolders, err := liveTx.ListVaultFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list live folders: %w", err)
	}
	folderNameSet := make(map[string]int64)
	for _, f := range existingFolders {
		folderNameSet[f.Name] = f.ID
	}

	// Import folders, building an old-ID → new-ID map.
	folderIDMap := make(map[int64]int64)
	backupFolderRows, err := backupDB.QueryContext(ctx,
		`SELECT id, name, parent_id, sort_order FROM vault_folders ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("read backup folders: %w", err)
	}

	type backupFolder struct {
		id        int64
		name      string
		parentID  sql.NullInt64
		sortOrder int64
	}
	var backupFolders []backupFolder
	for backupFolderRows.Next() {
		var f backupFolder
		if err := backupFolderRows.Scan(&f.id, &f.name, &f.parentID, &f.sortOrder); err != nil {
			backupFolderRows.Close()
			return nil, fmt.Errorf("scan backup folder: %w", err)
		}
		backupFolders = append(backupFolders, f)
	}
	backupFolderRows.Close()

	for _, f := range backupFolders {
		if existingID, exists := folderNameSet[f.name]; exists {
			folderIDMap[f.id] = existingID
			result.FoldersSkipped++
			continue
		}

		parentID := sql.NullInt64{}
		if f.parentID.Valid {
			if mapped, ok := folderIDMap[f.parentID.Int64]; ok {
				parentID = sql.NullInt64{Int64: mapped, Valid: true}
			}
		}

		newFolder, err := liveTx.CreateVaultFolder(ctx, db.CreateVaultFolderParams{
			Name:      f.name,
			ParentID:  parentID,
			SortOrder: f.sortOrder,
		})
		if err != nil {
			return nil, fmt.Errorf("create folder %q: %w", f.name, err)
		}
		folderIDMap[f.id] = newFolder.ID
		folderNameSet[f.name] = newFolder.ID
		result.FoldersImported++
	}

	// Build a set of existing entry names for dedup.
	existingEntries, err := liveTx.ListVaultEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list live entries: %w", err)
	}
	entryNameSet := make(map[string]bool)
	for _, e := range existingEntries {
		entryNameSet[e.Name] = true
	}

	// Import entries.
	entryRows, err := backupDB.QueryContext(ctx,
		`SELECT id, name, url_host, folder_id, ciphertext, nonce FROM vault_entries`)
	if err != nil {
		return nil, fmt.Errorf("read backup entries: %w", err)
	}
	defer entryRows.Close()

	for entryRows.Next() {
		var id int64
		var name, urlHost string
		var folderID sql.NullInt64
		var ct, nonce []byte
		if err := entryRows.Scan(&id, &name, &urlHost, &folderID, &ct, &nonce); err != nil {
			return nil, fmt.Errorf("scan backup entry: %w", err)
		}

		if entryNameSet[name] {
			result.EntriesSkipped++
			continue
		}

		plaintext, err := vaultcrypto.Decrypt(recoveryKey, ct, nonce)
		if err != nil {
			return nil, fmt.Errorf("decrypt backup entry %q: %w", name, err)
		}

		newCt, newNonce, err := vaultcrypto.Encrypt(liveKey, plaintext)
		if err != nil {
			return nil, fmt.Errorf("re-encrypt entry %q: %w", name, err)
		}

		newFolderID := sql.NullInt64{}
		if folderID.Valid {
			if mapped, ok := folderIDMap[folderID.Int64]; ok {
				newFolderID = sql.NullInt64{Int64: mapped, Valid: true}
			}
		}

		_, err = liveTx.CreateVaultEntry(ctx, db.CreateVaultEntryParams{
			Name:       name,
			UrlHost:    urlHost,
			FolderID:   newFolderID,
			Ciphertext: newCt,
			Nonce:      newNonce,
		})
		if err != nil {
			return nil, fmt.Errorf("create entry %q: %w", name, err)
		}
		entryNameSet[name] = true
		result.EntriesImported++
	}

	return result, nil
}
