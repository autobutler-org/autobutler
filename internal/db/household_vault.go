package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// The household vault (005_vault) was one vault every account opened with a
// shared master password. Per-account vaults replaced it, and migration 038
// drops its tables. Nothing can carry it forward: its entries are encrypted
// under a key derived from that master password, and an account's new vault
// is encrypted on the client under a key the Quark never sees. So it is kept,
// never deleted, and the vault page says where (#3616).
//
// The SQL in this file is raw for the exception AGENTS.md names: it writes a
// database outside the migration set, the preserved copy.

// HouseholdVaultFilePattern matches the copies of the household vault that
// preserveHouseholdVault leaves beside quark.db, one per upgrade that found
// one: household-vault-<UTC time>.db.
const HouseholdVaultFilePattern = "household-vault-*.db"

// LegacyDriveVaultFileName is the household vault an older Quark kept in an
// external drive's data directory once the vault was moved there. Nothing
// writes it any more, and nothing removes it either.
const LegacyDriveVaultFileName = "vault.db"

// HouseholdVaultSummary is what a household vault file holds, counted
// without opening any ciphertext.
type HouseholdVaultSummary struct {
	// Entries and Folders count its rows.
	Entries int64 `json:"entries"`
	Folders int64 `json:"folders"`
	// HasConfig reports the master password's salt and check, without which
	// nothing in it can be opened.
	HasConfig bool `json:"hasConfig"`
	// DeviceSerial is the drive the vault had been moved to, when it had;
	// only a copy made from quark.db records it.
	DeviceSerial string `json:"deviceSerial,omitempty"`
}

// householdVaultTables are the household vault's tables, parents first.
var householdVaultTables = []string{"vault_config", "vault_folders", "vault_entries", "vault_location"}

// householdVaultSchema is 005_vault's schema, created in the named schema so
// the copy is the same database an older Quark read.
func householdVaultSchema(schema string) string {
	return strings.ReplaceAll(`
CREATE TABLE IF NOT EXISTS S.vault_config (
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
CREATE TABLE IF NOT EXISTS S.vault_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    parent_id INTEGER REFERENCES vault_folders(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS S.vault_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url_host TEXT NOT NULL DEFAULT '',
    folder_id INTEGER REFERENCES vault_folders(id) ON DELETE SET NULL,
    ciphertext BLOB NOT NULL,
    nonce BLOB NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS S.idx_vault_entries_folder ON vault_entries(folder_id);
CREATE INDEX IF NOT EXISTS S.idx_vault_entries_url_host ON vault_entries(url_host);
CREATE TABLE IF NOT EXISTS S.vault_location (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    device_serial TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
`, "S.", schema+".")
}

// InspectHouseholdVault counts what the household vault file at path holds.
// It opens the file read-only and changes nothing in it. A table the file
// lacks counts as empty.
func InspectHouseholdVault(path string) (HouseholdVaultSummary, error) {
	if _, err := os.Stat(path); err != nil {
		return HouseholdVaultSummary{}, err
	}
	conn, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return HouseholdVaultSummary{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer conn.Close()
	return summarizeHouseholdVault(context.Background(), conn, "main")
}

// queryRower is the one method summarizeHouseholdVault needs, which a pool
// and a single connection both have.
type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// summarizeHouseholdVault counts the household vault's rows in schema.
func summarizeHouseholdVault(ctx context.Context, q queryRower, schema string) (HouseholdVaultSummary, error) {
	var summary HouseholdVaultSummary
	present := map[string]bool{}
	for _, table := range householdVaultTables {
		var n int
		if err := q.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM `+schema+`.sqlite_master WHERE type = 'table' AND name = ?`, table,
		).Scan(&n); err != nil {
			return summary, fmt.Errorf("look for %s: %w", table, err)
		}
		present[table] = n > 0
	}
	counts := map[string]*int64{"vault_entries": &summary.Entries, "vault_folders": &summary.Folders}
	for table, into := range counts {
		if !present[table] {
			continue
		}
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+schema+`.`+table).Scan(into); err != nil {
			return summary, fmt.Errorf("count %s: %w", table, err)
		}
	}
	if present["vault_config"] {
		var n int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+schema+`.vault_config`).Scan(&n); err != nil {
			return summary, fmt.Errorf("count vault_config: %w", err)
		}
		summary.HasConfig = n > 0
	}
	if present["vault_location"] {
		err := q.QueryRowContext(ctx, `SELECT device_serial FROM `+schema+`.vault_location WHERE id = 1`).
			Scan(&summary.DeviceSerial)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return summary, fmt.Errorf("read vault_location: %w", err)
		}
	}
	return summary, nil
}

// isEmpty reports a household vault that was never set up: 005 seeds
// vault_location with no serial, and nothing else.
func (s HouseholdVaultSummary) isEmpty() bool {
	return s == HouseholdVaultSummary{}
}

// preserveHouseholdVault copies the household vault's tables out of the
// database at path, into a new household-vault-<time>.db beside it, before
// migration 038 drops them. It returns the copy's path, or "" when there was
// nothing to keep: the tables are gone already, or the vault was never set up.
//
// The copy is written under a temporary name, counted against the source, and
// only then renamed into place, so a household-vault-*.db file is always a
// whole copy. Any failure is returned and the caller must not migrate: a
// server that refuses to start loses nothing, and 038 would.
func preserveHouseholdVault(sqlDB *sql.DB, path string) (string, error) {
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	source, err := summarizeHouseholdVault(ctx, conn, "main")
	if err != nil {
		return "", err
	}
	if source.isEmpty() {
		return "", nil
	}
	if path == "" {
		return "", errors.New("the household vault is in a database with no file to keep a copy beside")
	}

	dir := filepath.Dir(path)
	stamp := time.Now().UTC().Format("20060102T150405Z")
	final := filepath.Join(dir, "household-vault-"+stamp+".db")
	temp := filepath.Join(dir, ".household-vault-"+stamp+".db.tmp")
	// A crash mid-copy leaves the temp file, which nothing reads; start over.
	if err := os.Remove(temp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := copyHouseholdVault(ctx, conn, temp); err != nil {
		return "", errors.Join(err, os.Remove(temp))
	}
	copied, err := InspectHouseholdVault(temp)
	if err == nil && copied != source {
		err = fmt.Errorf("the copy holds %+v, the database %+v", copied, source)
	}
	if err == nil {
		err = storageutil.SyncFile(temp)
	}
	if err == nil {
		err = renameWithoutOverwriting(temp, final)
	}
	if err != nil {
		return "", errors.Join(fmt.Errorf("copy the household vault to %s: %w", final, err), os.Remove(temp))
	}
	if err := storageutil.SyncDir(dir); err != nil {
		return "", err
	}
	return final, nil
}

// copyHouseholdVault writes every household vault row on conn into a new
// database at path. The foreign keys are checked at commit rather than per
// row, because a folder can name a parent with a higher id.
func copyHouseholdVault(ctx context.Context, conn *sql.Conn, path string) error {
	if _, err := conn.ExecContext(ctx, `ATTACH DATABASE ? AS household_copy`, path); err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	// Not ctx: the connection goes back to the pool and must not keep the file.
	defer func() { _, _ = conn.ExecContext(context.WithoutCancel(ctx), `DETACH DATABASE household_copy`) }()

	statements := []string{`BEGIN`, `PRAGMA defer_foreign_keys = ON`, householdVaultSchema("household_copy")}
	for _, table := range householdVaultTables {
		statements = append(statements, `INSERT INTO household_copy.`+table+` SELECT * FROM main.`+table)
	}
	for _, statement := range append(statements, `COMMIT`) {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			_, _ = conn.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
			return fmt.Errorf("copy the household vault: %w", err)
		}
	}
	return nil
}
