package vaultutil_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
)

// householdVaultDDL is the part of 005_vault's schema a household vault file
// needs for these tests.
const householdVaultDDL = `
CREATE TABLE vault_config (id INTEGER PRIMARY KEY, salt BLOB NOT NULL, verification_blob BLOB NOT NULL, verification_nonce BLOB NOT NULL);
CREATE TABLE vault_folders (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, parent_id INTEGER);
CREATE TABLE vault_entries (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, url_host TEXT NOT NULL DEFAULT '', folder_id INTEGER, ciphertext BLOB NOT NULL, nonce BLOB NOT NULL);
`

// writeHouseholdVault makes a household vault file at path holding entries
// entries and one folder.
func writeHouseholdVault(t *testing.T, path string, entries int, extra string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("sqlite", db.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	statements := householdVaultDDL + `
		INSERT INTO vault_config (id, salt, verification_blob, verification_nonce) VALUES (1, x'01', x'02', x'03');
		INSERT INTO vault_folders (name) VALUES ('Home');` + extra
	for range entries {
		statements += `INSERT INTO vault_entries (name, ciphertext, nonce) VALUES ('entry', x'01', x'02');`
	}
	if _, err := conn.Exec(statements); err != nil {
		t.Fatalf("seed %s: %v", path, err)
	}
}

// fakeDevices is the storage service's list of managed drives.
type fakeDevices struct {
	devices []storageutil.ManagedDevice
	err     error
}

func (f fakeDevices) GetManagedDevices() ([]storageutil.ManagedDevice, error) {
	return f.devices, f.err
}

func drive(name, dataDir string) storageutil.ManagedDevice {
	return storageutil.ManagedDevice{Device: storageutil.Device{Name: name}, DataDir: dataDir}
}

// TestListLegacyVaults_FindsEveryHouseholdVaultAnUpgradeLeft lays out a data
// directory as an upgrade from before per-account vaults leaves it: the copy
// kept beside quark.db, and a drive still holding the vault.db an older Quark
// moved the vault into. Both are listed, counted, and left exactly where they
// are.
func TestListLegacyVaults_FindsEveryHouseholdVaultAnUpgradeLeft(t *testing.T) {
	dataDir := t.TempDir()
	kept := filepath.Join(dataDir, "household-vault-20261021T120000Z.db")
	writeHouseholdVault(t, kept, 3, `
		CREATE TABLE vault_location (id INTEGER PRIMARY KEY, device_serial TEXT NOT NULL DEFAULT '');
		INSERT INTO vault_location (id, device_serial) VALUES (1, 'USB-1');`)
	// A copy interrupted before it was proven is not a kept vault.
	if err := os.WriteFile(filepath.Join(dataDir, ".household-vault-20261021T120000Z.db.tmp"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	driveDir := filepath.Join(t.TempDir(), "quark", "data")
	onDrive := filepath.Join(driveDir, db.LegacyDriveVaultFileName)
	writeHouseholdVault(t, onDrive, 2, "")
	emptyDrive := filepath.Join(t.TempDir(), "quark", "data")
	brokenDir := filepath.Join(t.TempDir(), "quark", "data")
	broken := filepath.Join(brokenDir, db.LegacyDriveVaultFileName)
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := vaultutil.ListLegacyVaults(vaultutil.ListLegacyVaultsParams{
		DataDir: dataDir,
		Storage: fakeDevices{devices: []storageutil.ManagedDevice{
			// The internal drive is listed by the storage service too.
			{Device: storageutil.Device{Name: "Internal", IsInternal: true}, DataDir: dataDir},
			drive("Backup drive", driveDir),
			drive("Photos drive", emptyDrive),
			drive("Old drive", brokenDir),
		}},
	})
	if err != nil {
		t.Fatalf("ListLegacyVaults: %v", err)
	}
	want := []vaultutil.LegacyVault{
		{Kind: vaultutil.LegacyVaultKept, Path: kept, Readable: true, Entries: 3, Folders: 1, DeviceSerial: "USB-1"},
		{Kind: vaultutil.LegacyVaultOnDrive, Path: onDrive, Readable: true, Entries: 2, Folders: 1, DeviceName: "Backup drive"},
		{Kind: vaultutil.LegacyVaultOnDrive, Path: broken, DeviceName: "Old drive"},
	}
	if !reflect.DeepEqual(result.Vaults, want) {
		t.Fatalf("vaults =\n%+v\nwant\n%+v", result.Vaults, want)
	}
	for _, path := range []string{kept, onDrive, broken} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s is gone: %v", path, err)
		}
	}
}

// TestListLegacyVaults_NothingToFind is every install that never had a
// household vault: the list is empty, not nil, so it serializes as [].
func TestListLegacyVaults_NothingToFind(t *testing.T) {
	result, err := vaultutil.ListLegacyVaults(vaultutil.ListLegacyVaultsParams{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("ListLegacyVaults: %v", err)
	}
	if result.Vaults == nil || len(result.Vaults) != 0 {
		t.Fatalf("vaults = %#v, want an empty list", result.Vaults)
	}
}

// TestListLegacyVaults_StillListsThisQuarkWhenDrivesFail keeps the kept copy
// visible when the drives cannot be listed.
func TestListLegacyVaults_StillListsThisQuarkWhenDrivesFail(t *testing.T) {
	dataDir := t.TempDir()
	writeHouseholdVault(t, filepath.Join(dataDir, "household-vault-20261021T120000Z.db"), 1, "")
	result, err := vaultutil.ListLegacyVaults(vaultutil.ListLegacyVaultsParams{
		DataDir: dataDir,
		Storage: fakeDevices{err: errors.New("no drives")},
	})
	if err != nil {
		t.Fatalf("ListLegacyVaults: %v", err)
	}
	if len(result.Vaults) != 1 || result.Vaults[0].Entries != 1 {
		t.Fatalf("vaults = %+v, want the kept copy", result.Vaults)
	}
}
