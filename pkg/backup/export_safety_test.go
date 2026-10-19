package backup

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

// assertNoTemps fails when an export left a temp file in dir.
func assertNoTemps(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), storageutil.WriteTempPrefix) {
			t.Errorf("export left %s behind", entry.Name())
		}
	}
}

// Two chat exports used to share one temp name and remove each other's
// (#3084, F13).
func TestExportChat_ConcurrentExportsBothSucceed(t *testing.T) {
	database := dbtest.NewDB(t)
	seedChat(t, database.Db)
	targetDir := t.TempDir()

	errs := make([]error, 4)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Go(func() {
			_, errs[i] = ExportChat(context.Background(), database.Db, targetDir)
		})
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Errorf("concurrent export: %v", err)
		}
	}

	live := dbtest.NewDB(t).Db
	addChatUsers(t, live, "alice", "bob", "carol")
	result, err := ImportChat(context.Background(), live, targetDir)
	if err != nil {
		t.Fatalf("the backup two exports raced for does not import: %v", err)
	}
	if result.Messages != 4 {
		t.Errorf("restored %d messages, want 4", result.Messages)
	}
	assertNoTemps(t, targetDir)
}
