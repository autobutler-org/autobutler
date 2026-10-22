package db

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// dropHouseholdVaultVersion is 038_drop_household_vault, which drops the
// household vault's tables (#3616).
const dropHouseholdVaultVersion = 38

// householdVaultSecret is an entry name the household vault kept in
// plaintext. After the upgrade it must be in the preserved copy and nowhere
// in quark.db, its freed pages included.
const householdVaultSecret = "household-bank-login-3616"

// seedHouseholdVault fills the household vault's tables the way an older
// Quark left them: a config row, a folder with a child folder made before its
// parent's id, two entries, and the vault on the drive serial names.
func seedHouseholdVault(t *testing.T, conn *sql.DB, serial string) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO vault_config (id, salt, verification_blob, verification_nonce) VALUES (1, x'0102', x'0304', x'0506')`,
		`INSERT INTO vault_folders (id, name, parent_id) VALUES (1, 'Banking', NULL)`,
		`INSERT INTO vault_folders (id, name, parent_id) VALUES (3, 'Root', NULL)`,
		`UPDATE vault_folders SET parent_id = 3 WHERE id = 1`,
		`INSERT INTO vault_entries (name, url_host, folder_id, ciphertext, nonce) VALUES ('` + householdVaultSecret + `', 'bank.example', 1, x'aa', x'bb')`,
		`INSERT INTO vault_entries (name, url_host, folder_id, ciphertext, nonce) VALUES ('Mail', 'mail.example', NULL, x'cc', x'dd')`,
	} {
		if _, err := conn.Exec(statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	if _, err := conn.Exec(`UPDATE vault_location SET device_serial = ? WHERE id = 1`, serial); err != nil {
		t.Fatalf("set the vault location: %v", err)
	}
}

// preservedHouseholdVaults lists the household vault copies in dir.
func preservedHouseholdVaults(t *testing.T, dir string) []string {
	t.Helper()
	found, err := filepath.Glob(filepath.Join(dir, HouseholdVaultFilePattern))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	return found
}

// TestInitSchema_PreservesTheHouseholdVaultBeforeDroppingIt upgrades a
// database that still holds the household vault. Migration 038 drops its
// tables; before it runs, the rows are copied whole into a file beside
// quark.db, and once the copy is proven the freed pages are cleared.
func TestInitSchema_PreservesTheHouseholdVaultBeforeDroppingIt(t *testing.T) {
	conn, _ := migrateTo(t, dropHouseholdVaultVersion-1)
	seedHouseholdVault(t, conn, "USB-SERIAL-1")
	path, err := databaseFilePath(conn)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(path)

	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initSchema: %v", err)
	}

	if n := count(t, conn, `SELECT COUNT(*) FROM sqlite_master WHERE name = 'vault_entries'`); n != 0 {
		t.Fatalf("vault_entries is still in quark.db after the upgrade")
	}
	kept := preservedHouseholdVaults(t, dir)
	if len(kept) != 1 {
		t.Fatalf("preserved copies = %v, want one", kept)
	}
	summary, err := InspectHouseholdVault(kept[0])
	if err != nil {
		t.Fatalf("inspect the preserved copy: %v", err)
	}
	want := HouseholdVaultSummary{Entries: 2, Folders: 2, HasConfig: true, DeviceSerial: "USB-SERIAL-1"}
	if summary != want {
		t.Fatalf("preserved copy = %+v, want %+v", summary, want)
	}

	copied, err := sql.Open("sqlite", DSN(kept[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer copied.Close()
	var name, host string
	var ciphertext []byte
	if err := copied.QueryRow(
		`SELECT name, url_host, ciphertext FROM vault_entries WHERE folder_id = 1`,
	).Scan(&name, &host, &ciphertext); err != nil {
		t.Fatalf("read the preserved entry: %v", err)
	}
	if name != householdVaultSecret || host != "bank.example" || !bytes.Equal(ciphertext, []byte{0xaa}) {
		t.Fatalf("preserved entry = %q %q %x", name, host, ciphertext)
	}
	if n := count(t, copied, `SELECT COUNT(*) FROM vault_folders WHERE id = 1 AND parent_id = 3`); n != 1 {
		t.Fatalf("the folder tree was not preserved")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(householdVaultSecret)) {
		t.Fatalf("quark.db still holds the household vault's plaintext in its free pages")
	}
}

// TestInitSchema_MakesNoCopyOfAnEmptyHouseholdVault covers every install
// that never set the household vault up: 005 seeds vault_location and nothing
// else, and there is nothing to keep.
func TestInitSchema_MakesNoCopyOfAnEmptyHouseholdVault(t *testing.T) {
	conn, _ := migrateTo(t, dropHouseholdVaultVersion-1)
	path, err := databaseFilePath(conn)
	if err != nil {
		t.Fatal(err)
	}
	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initSchema: %v", err)
	}
	if kept := preservedHouseholdVaults(t, filepath.Dir(path)); len(kept) != 0 {
		t.Fatalf("preserved copies = %v, want none", kept)
	}
}

// TestInitSchema_KeepsAnEarlierCopy upgrades a second database into a
// directory that already holds a preserved copy, which must survive as it was.
func TestInitSchema_KeepsAnEarlierCopy(t *testing.T) {
	conn, _ := migrateTo(t, dropHouseholdVaultVersion-1)
	seedHouseholdVault(t, conn, "")
	path, err := databaseFilePath(conn)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(path)
	earlier := filepath.Join(dir, "household-vault-20260101T000000Z.db")
	if err := os.WriteFile(earlier, []byte("an earlier copy"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := initSchema(&DatabaseSqlc{Db: conn}); err != nil {
		t.Fatalf("initSchema: %v", err)
	}
	if kept := preservedHouseholdVaults(t, dir); len(kept) != 2 {
		t.Fatalf("preserved copies = %v, want the earlier one and a new one", kept)
	}
	if raw, err := os.ReadFile(earlier); err != nil || string(raw) != "an earlier copy" {
		t.Fatalf("the earlier copy changed: %q, %v", raw, err)
	}
}

// TestInspectHouseholdVault_ReadsADrivesVaultFile reads the vault.db an
// older Quark kept on an external drive, which has no vault_location table.
func TestInspectHouseholdVault_ReadsADrivesVaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), LegacyDriveVaultFileName)
	conn, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(householdVaultSchema("main") + `
		INSERT INTO vault_entries (name, ciphertext, nonce) VALUES ('Wifi', x'01', x'02');`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	summary, err := InspectHouseholdVault(path)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if want := (HouseholdVaultSummary{Entries: 1}); summary != want {
		t.Fatalf("summary = %+v, want %+v", summary, want)
	}
}
