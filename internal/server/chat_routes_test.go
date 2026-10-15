package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/internal/server/middleware"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// TestChatRoutesAnswerWhenFlagIsOff checks chat answers with the flag stored
// off (#2577). Calendar, still in beta, stays a 404 while its flag is off.
func TestChatRoutesAnswerWhenFlagIsOff(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	t.Cleanup(func() { settingsutil.ResetForTesting("") })
	if err := settingsutil.SetFeatureFlag("chat", false); err != nil {
		t.Fatal(err)
	}
	if err := settingsutil.SetFeatureFlag("calendar", false); err != nil {
		t.Fatal(err)
	}

	database := dbtest.NewDB(t)
	admin, err := authutil.Setup(context.Background(), authutil.SetupParams{
		Database: database, Files: vfs.NewMemVFS("files"), Username: "admin",
		AuthKey: dbtest.AuthKey("admin-password"), SaltSecret: dbtest.SaltSecret,
	})
	if err != nil {
		t.Fatalf("authutil.Setup: %v", err)
	}

	deps := deputil.NewDependencies().WithDatabase(database)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, _ any) {
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	middleware.Use(engine, deps)
	setupRouters(engine, nil, deps)
	t.Cleanup(deps.Background().Wait)

	do := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+admin.SessionToken)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}

	chat := do("/api/v0/chat/channels")
	if chat.Code != http.StatusOK || strings.Contains(chat.Body.String(), "turned off") {
		t.Fatalf("GET /chat/channels = %d %s, want 200 with the flag stored off", chat.Code, chat.Body.String())
	}
	calendar := do("/api/v0/calendar/events")
	if calendar.Code != http.StatusNotFound || !strings.Contains(calendar.Body.String(), "turned off") {
		t.Fatalf("GET /calendar/events = %d %s, want 404 while the calendar flag is off", calendar.Code, calendar.Body.String())
	}
}
