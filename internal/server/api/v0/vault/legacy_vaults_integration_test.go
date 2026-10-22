package v0_vault_test

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
)

// TestListLegacyVaults_TellsEveryAccountAboutTheKeptHouseholdVault serves
// GET /vault/legacy over a data directory an upgrade left a household vault
// copy in (#3616). Every account used the household vault, so every account
// is told, and the file is still there afterwards.
func TestListLegacyVaults_TellsEveryAccountAboutTheKeptHouseholdVault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	h := newHarness(t)

	var empty vaultutil.ListLegacyVaultsResult
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/legacy", "bob", "", &empty)
	if empty.Vaults == nil || len(empty.Vaults) != 0 {
		t.Fatalf("before any copy: %+v, want an empty list", empty.Vaults)
	}

	dataDir := storageutil.GetDataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(dataDir, "household-vault-20261021T120000Z.db")
	conn, err := sql.Open("sqlite", db.DSN(kept))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`
		CREATE TABLE vault_entries (id INTEGER PRIMARY KEY, name TEXT NOT NULL, ciphertext BLOB NOT NULL, nonce BLOB NOT NULL);
		INSERT INTO vault_entries (name, ciphertext, nonce) VALUES ('Bank', x'01', x'02'), ('Mail', x'03', x'04');`); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	for _, as := range []string{"bob", "admin"} {
		var listed vaultutil.ListLegacyVaultsResult
		h.expect(t, http.StatusOK, http.MethodGet, "/vault/legacy", as, "", &listed)
		want := vaultutil.LegacyVault{Kind: vaultutil.LegacyVaultKept, Path: kept, Readable: true, Entries: 2}
		if len(listed.Vaults) != 1 || listed.Vaults[0] != want {
			t.Fatalf("as %s: %+v, want [%+v]", as, listed.Vaults, want)
		}
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("the kept copy is gone: %v", err)
	}

	h.expect(t, http.StatusUnauthorized, http.MethodGet, "/vault/legacy", "nobody", "", nil)
}
