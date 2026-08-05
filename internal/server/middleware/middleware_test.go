package middleware_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/internal/server/middleware"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

const authSchema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	recovery_phrase_hash TEXT NOT NULL,
	is_admin INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
`

func newMiddlewareTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if _, err := sqlDB.Exec(authSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get conn: %v", err)
	}
	return sqlDB, db.New(conn)
}

// newMiddlewareEngine creates a minimal gin engine with the middleware stack
// applied and a single /api/v0/protected GET that returns 200 when reached.
func newMiddlewareEngine(t *testing.T, deps deputil.Dependencies) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.Use(engine, deps)
	engine.GET("/api/v0/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	// /api/v0/events is the canonical path for ?token= query-param auth (WebSocket).
	engine.GET("/api/v0/events", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	engine.GET("/api/v0/auth/status", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	engine.GET("/non-api/page", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return engine
}

func doMiddlewareReq(engine *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// TestRequireAuth_NoDatabaseReturns503 verifies that when no database is
// configured, API routes return 503 (fail closed).
func TestRequireAuth_NoDatabaseReturns503(t *testing.T) {
	// Deps with no database set.
	deps := deputil.NewDependencies()
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 (no DB), got %d", w.Code)
	}
}

// TestRequireAuth_ExemptPathsPassThrough verifies that setup/login/recover/status
// do not require authentication even with a configured DB.
func TestRequireAuth_ExemptPathsPassThrough(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	for _, path := range []string{
		"/api/v0/auth/status",
		// setup/login/recover aren't registered in the test engine so we only
		// test /auth/status which is registered above.
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := doMiddlewareReq(engine, req)
		if w.Code == http.StatusUnauthorized {
			t.Errorf("exempt path %q returned 401; should be exempt", path)
		}
	}
}

// TestRequireAuth_NonAPIPathPassesThrough verifies that non-/api/ paths bypass
// auth entirely (e.g. static assets served by the Flutter web app).
func TestRequireAuth_NonAPIPathPassesThrough(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/non-api/page", nil)
	w := doMiddlewareReq(engine, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("non-API path returned 401; should be exempt from auth")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestRequireAuth_SetupNotCompletePassesThrough verifies that API routes are
// accessible before setup is complete (the setup wizard must be reachable).
func TestRequireAuth_SetupNotCompletePassesThrough(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	// No users created → setup not complete.
	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	w := doMiddlewareReq(engine, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("pre-setup API request returned 401; should be permitted before setup is complete")
	}
}

// TestRequireAuth_UnauthorizedAfterSetup verifies that after setup is complete,
// unauthenticated requests to /api/ routes are rejected with 401.
func TestRequireAuth_UnauthorizedAfterSetup(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	ctx := context.Background()
	_, err := authutil.Setup(ctx, queries, authutil.SetupParams{
		Username: "admin",
		Password: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 after setup without auth, got %d", w.Code)
	}
}

// TestRequireAuth_BearerTokenGrantsAccess verifies a valid Bearer token in the
// Authorization header grants access to a protected route.
func TestRequireAuth_BearerTokenGrantsAccess(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	ctx := context.Background()
	result, err := authutil.Setup(ctx, queries, authutil.SetupParams{
		Username: "admin",
		Password: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	req.Header.Set("Authorization", "Bearer "+result.SessionToken)
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid Bearer token, got %d: %s", w.Code, w.Body.String())
	}
}

// TestRequireAuth_CookieGrantsAccess verifies a valid session cookie grants
// access to a protected route.
func TestRequireAuth_CookieGrantsAccess(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	ctx := context.Background()
	result, err := authutil.Setup(ctx, queries, authutil.SetupParams{
		Username: "admin",
		Password: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: result.SessionToken})
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with session cookie, got %d", w.Code)
	}
}

// TestRequireAuth_QueryTokenGrantsAccess verifies a valid ?token= query param
// grants access to a protected route.
func TestRequireAuth_QueryTokenGrantsAccess(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	ctx := context.Background()
	result, err := authutil.Setup(ctx, queries, authutil.SetupParams{
		Username: "admin",
		Password: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	// Use /api/v0/events — the only path where ?token= query-param auth is permitted.
	req := httptest.NewRequest(http.MethodGet,
		"/api/v0/events?token="+result.SessionToken, nil)
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with ?token=, got %d", w.Code)
	}
}

// TestRequireAuth_InvalidTokenReturns401 verifies a bogus token is rejected.
func TestRequireAuth_InvalidTokenReturns401(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	ctx := context.Background()
	_, err := authutil.Setup(ctx, queries, authutil.SetupParams{
		Username: "admin",
		Password: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	req.Header.Set("Authorization", "Bearer totally-fake-token-that-does-not-exist")
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", w.Code)
	}
}

// TestRequireAuth_BasicAuthGrantsAccess verifies HTTP Basic Auth works as a
// fallback authentication method.
func TestRequireAuth_BasicAuthGrantsAccess(t *testing.T) {
	sqlDB, queries := newMiddlewareTestDB(t)
	ctx := context.Background()
	_, err := authutil.Setup(ctx, queries, authutil.SetupParams{
		Username: "admin",
		Password: "SecurePass1!",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(&db.DatabaseSqlc{Db: sqlDB, Queries: queries})
	engine := newMiddlewareEngine(t, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/protected", nil)
	req.SetBasicAuth("admin", "SecurePass1!")
	w := doMiddlewareReq(engine, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with Basic Auth, got %d", w.Code)
	}
}
