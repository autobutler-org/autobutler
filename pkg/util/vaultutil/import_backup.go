package vaultutil

import (
	"context"
	"fmt"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/backup"
)

// ImportBackupParams restores entries from a recovery backup written to a
// managed device by [github.com/autobutler-org/quark/pkg/backup.ExportVault].
type ImportBackupParams struct {
	// VaultDB is the live vault. The restore runs in one transaction on it.
	VaultDB *db.DatabaseSqlc
	// LiveKey is the unlocked vault key the recovered entries are re-sealed
	// under.
	LiveKey []byte
	// RecoveryPassword unlocks the backup file.
	RecoveryPassword string
	// BackupDir is the device directory holding the backup.
	BackupDir string
}

// ImportBackupResult carries the counts pkg/backup reports.
type ImportBackupResult struct {
	Import *backup.VaultImportResult
}

// ImportBackup restores a recovery backup into the live vault. A rejected
// backup — wrong recovery password, missing or corrupt file — comes back
// wrapped in [ErrBackupImportFailed], which the caller answers as bad input
// rather than a server fault.
func ImportBackup(ctx context.Context, params ImportBackupParams) (ImportBackupResult, error) {
	tx, err := params.VaultDB.Db.BeginTx(ctx, nil)
	if err != nil {
		return ImportBackupResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := params.VaultDB.Queries.WithTx(tx)

	result, err := backup.ImportVault(ctx, qtx, params.LiveKey, params.RecoveryPassword, params.BackupDir)
	if err != nil {
		return ImportBackupResult{}, fmt.Errorf("%w: %w", ErrBackupImportFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return ImportBackupResult{}, fmt.Errorf("commit: %w", err)
	}

	return ImportBackupResult{Import: result}, nil
}
