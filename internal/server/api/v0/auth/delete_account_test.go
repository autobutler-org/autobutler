package v0_auth_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	v0_auth "github.com/autobutler-org/quark/internal/server/api/v0/auth"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

const deleteAccountUser = "testuser"

// newDeleteAccountEngine builds a gin engine wired to a real on-disk database
// carrying the real migration set, with HOME redirected at a temporary
// directory so storageutil.GetDataDir resolves inside the test sandbox rather
// than at the developer's actual data directory. It returns the engine, the
// database handle, and the files directory the handler will delete.
func newDeleteAccountEngine(t *testing.T) (*gin.Engine, *sql.DB, string) {
	t.Helper()

	// Redirected before GetDataDir is called anywhere below: every platform
	// branch of GetDataDirForDevice derives from os.UserHomeDir.
	t.Setenv("HOME", t.TempDir())

	sqlDB, err := sql.Open("sqlite", db.DSN(filepath.Join(t.TempDir(), "quark.db")))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	database := &db.DatabaseSqlc{Db: sqlDB}
	// On an empty file the drop half is a no-op, so this is just "run every
	// migration" — the same schema production boots with.
	if err := db.ResetDatabase(database); err != nil {
		t.Fatalf("build schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	database.Queries = db.New(conn)

	if _, err := authutil.Setup(context.Background(), database.Queries, authutil.SetupParams{
		Username: deleteAccountUser,
		Password: "TestPassword123!",
	}); err != nil {
		t.Fatalf("authutil.Setup: %v", err)
	}

	filesDir, err := storageutil.GetFilesDir()
	if err != nil {
		t.Fatalf("GetFilesDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filesDir, "keep.txt"), []byte("data"), 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(database)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		// Production's requireAuth sets "username" and nothing else.
		c = ctxutil.With(c, "username", deleteAccountUser)
		c.Next()
	})
	group := engine.Group("/api/v0")
	serverutil.RegisterRouterWithGroup(group, v0_auth.NewRouter())
	return engine, sqlDB, filesDir
}

func deleteAccountRequest(engine *gin.Engine, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, "/api/v0/auth/account?"+query, nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// decodeDeleted reads the {"deleted": {...}} envelope a successful call returns.
func decodeDeleted(t *testing.T, w *httptest.ResponseRecorder) (bool, bool, bool) {
	t.Helper()
	var body struct {
		Deleted struct {
			Database bool `json:"database"`
			Files    bool `json:"files"`
			Devices  bool `json:"devices"`
		} `json:"deleted"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, w.Body.String())
	}
	return body.Deleted.Database, body.Deleted.Files, body.Deleted.Devices
}

func userCount(t *testing.T, sqlDB *sql.DB) int {
	t.Helper()
	var count int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	return count
}

// TestDeleteAccount_NoAspectSelected verifies a request that selects nothing is
// rejected rather than treated as a full wipe.
func TestDeleteAccount_NoAspectSelected(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	for _, query := range []string{
		"confirm=" + deleteAccountUser,
		"database=false&files=false&confirm=" + deleteAccountUser,
		"database=false&files=false&devices=false&confirm=" + deleteAccountUser,
	} {
		w := deleteAccountRequest(engine, query)
		if w.Code != http.StatusBadRequest {
			t.Errorf("query %q: expected 400, got %d: %s", query, w.Code, w.Body.String())
		}
	}

	if _, err := os.Stat(filesDir); err != nil {
		t.Errorf("files directory should be untouched: %v", err)
	}
	if userCount(t, sqlDB) != 1 {
		t.Error("database should be untouched")
	}
}

// TestDeleteAccount_ConfirmationMissing verifies the confirmation guard rejects
// a request that omits it, even with a valid aspect selected.
func TestDeleteAccount_ConfirmationMissing(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "database=true&files=true")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filesDir); err != nil {
		t.Errorf("files directory should be untouched: %v", err)
	}
	if userCount(t, sqlDB) != 1 {
		t.Error("database should be untouched")
	}
}

// TestDeleteAccount_ConfirmationMismatched verifies a confirmation naming some
// other user is rejected.
func TestDeleteAccount_ConfirmationMismatched(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "files=true&confirm=other-user")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filesDir); err != nil {
		t.Errorf("files directory should be untouched: %v", err)
	}
	if userCount(t, sqlDB) != 1 {
		t.Error("database should be untouched")
	}
}

// TestDeleteAccount_RejectsNonBoolean verifies an unparseable aspect is a 400
// rather than a silent false.
func TestDeleteAccount_RejectsNonBoolean(t *testing.T) {
	engine, _, _ := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "files=not-a-bool&confirm="+deleteAccountUser)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteAccount_RejectsNonBooleanDevices verifies the new aspect is
// validated like the other two rather than falling through as false.
func TestDeleteAccount_RejectsNonBooleanDevices(t *testing.T) {
	engine, _, _ := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "devices=sure&confirm="+deleteAccountUser)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestDeleteAccount_DevicesOnlyIsAValidSelection verifies devices=true on its
// own satisfies the "select something" guard and leaves the appliance's own
// data alone. No drive is attached in the test environment, so this covers the
// plumbing; the path behavior is covered in authutil's own tests.
func TestDeleteAccount_DevicesOnlyIsAValidSelection(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "devices=true&confirm="+deleteAccountUser)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	databaseDeleted, filesDeleted, devicesDeleted := decodeDeleted(t, w)
	if databaseDeleted || filesDeleted || !devicesDeleted {
		t.Errorf("expected database=false files=false devices=true, got database=%v files=%v devices=%v",
			databaseDeleted, filesDeleted, devicesDeleted)
	}

	if _, err := os.Stat(filepath.Join(filesDir, "keep.txt")); err != nil {
		t.Errorf("local files must survive a devices-only reset: %v", err)
	}
	if userCount(t, sqlDB) != 1 {
		t.Error("database must survive a devices-only reset")
	}
}

// TestDeleteAccount_FilesOnly verifies files=true removes the files directory
// and leaves the database intact.
func TestDeleteAccount_FilesOnly(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "files=true&confirm="+deleteAccountUser)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	databaseDeleted, filesDeleted, devicesDeleted := decodeDeleted(t, w)
	if databaseDeleted || !filesDeleted || devicesDeleted {
		t.Errorf("expected database=false files=true devices=false, got database=%v files=%v devices=%v",
			databaseDeleted, filesDeleted, devicesDeleted)
	}

	if _, err := os.Stat(filesDir); !os.IsNotExist(err) {
		t.Errorf("files directory should be gone, stat returned %v", err)
	}
	if userCount(t, sqlDB) != 1 {
		t.Error("database should have survived a files-only delete")
	}
}

// TestDeleteAccount_DatabaseOnly verifies database=true resets the schema to
// first-boot state and leaves stored files alone.
func TestDeleteAccount_DatabaseOnly(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "database=true&confirm="+deleteAccountUser)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	databaseDeleted, filesDeleted, devicesDeleted := decodeDeleted(t, w)
	if !databaseDeleted || filesDeleted || devicesDeleted {
		t.Errorf("expected database=true files=false devices=false, got database=%v files=%v devices=%v",
			databaseDeleted, filesDeleted, devicesDeleted)
	}

	if userCount(t, sqlDB) != 0 {
		t.Error("users should be gone after a database reset")
	}
	// The schema is back, not merely emptied: sessions is queryable again.
	var sessions int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Errorf("schema should have been re-migrated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filesDir, "keep.txt")); err != nil {
		t.Errorf("files should have survived a database-only delete: %v", err)
	}
}

// TestDeleteAccount_Both verifies the two aspects combine in one call.
func TestDeleteAccount_Both(t *testing.T) {
	engine, sqlDB, filesDir := newDeleteAccountEngine(t)

	w := deleteAccountRequest(engine, "database=true&files=true&confirm="+deleteAccountUser)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	databaseDeleted, filesDeleted, devicesDeleted := decodeDeleted(t, w)
	if !databaseDeleted || !filesDeleted {
		t.Errorf("expected both true, got database=%v files=%v", databaseDeleted, filesDeleted)
	}
	if devicesDeleted {
		t.Error("devices must stay false when it was not requested")
	}

	if userCount(t, sqlDB) != 0 {
		t.Error("users should be gone")
	}
	if _, err := os.Stat(filesDir); !os.IsNotExist(err) {
		t.Errorf("files directory should be gone, stat returned %v", err)
	}
}

// TestDeleteAccount_RepeatIsIdempotent verifies deleting what is already gone
// is a 200 rather than a 500. Files-only, because after a database reset the
// caller no longer has an account to authenticate a second call with.
func TestDeleteAccount_RepeatIsIdempotent(t *testing.T) {
	engine, _, filesDir := newDeleteAccountEngine(t)

	first := deleteAccountRequest(engine, "files=true&confirm="+deleteAccountUser)
	if first.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d: %s", first.Code, first.Body.String())
	}
	second := deleteAccountRequest(engine, "files=true&confirm="+deleteAccountUser)
	if second.Code != http.StatusOK {
		t.Fatalf("second call: expected 200, got %d: %s", second.Code, second.Body.String())
	}
	if _, err := os.Stat(filesDir); !os.IsNotExist(err) {
		t.Errorf("files directory should still be gone, stat returned %v", err)
	}
}

// TestDeleteAccount_RevokesSessions verifies every session is gone after a
// files-only delete, where the sessions table itself survives.
func TestDeleteAccount_RevokesSessions(t *testing.T) {
	engine, sqlDB, _ := newDeleteAccountEngine(t)

	var before int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM sessions`).Scan(&before); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if before == 0 {
		t.Fatal("setup should have created a session to revoke")
	}

	w := deleteAccountRequest(engine, "files=true&confirm="+deleteAccountUser)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var after int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM sessions`).Scan(&after); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if after != 0 {
		t.Errorf("expected all sessions revoked, %d remain", after)
	}
}
