package db_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// sqlc accepting a query is not SQLite accepting it: sqlc v1.30.0 generates
// code for a statement it left a parameter unexpanded in
// (sqlc-dev/sqlc#4110), and that code fails at run time with a syntax error.
// Preparing every generated statement against the migrated schema turns that
// into a test failure.
func TestEveryGeneratedQueryPrepares(t *testing.T) {
	files, err := filepath.Glob("*.sql.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no generated query files found: %v", err)
	}
	database := dbtest.NewDB(t)
	total := 0
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		checked, failures := dbtest.PrepareAll(database.Db, string(source))
		total += checked
		for _, failure := range failures {
			t.Errorf("%s: %s", file, failure)
		}
	}
	if total == 0 {
		t.Fatal("found no statements in the generated files; the generatedQuery pattern no longer matches sqlc's output")
	}
	t.Logf("prepared %d statements", total)
}

// The check above is only worth having if it can fail.
func TestPrepareAllReportsARejectedStatement(t *testing.T) {
	source := "const good = `-- name: Good :one\nSELECT install_id FROM install WHERE id = ?\n`\n" +
		"const unexpanded = `-- name: Unexpanded :exec\nUPDATE install SET install_id = sqlc.arg(install_id)\n`\n" +
		"const noColumn = `-- name: NoColumn :one\nSELECT no_such_column FROM install\n`\n"
	checked, failures := dbtest.PrepareAll(dbtest.NewDB(t).Db, source)
	if checked != 3 || len(failures) != 2 {
		t.Fatalf("checked %d statements with failures %q, want 3 checked and Unexpanded and NoColumn rejected", checked, failures)
	}
}

func TestNewInstallID(t *testing.T) {
	first, second := db.NewInstallID(), db.NewInstallID()
	if raw, err := hex.DecodeString(first); err != nil || len(raw) != 16 || first == second {
		t.Fatalf("NewInstallID() = %q then %q, want two different ids of 32 hex digits", first, second)
	}
}
