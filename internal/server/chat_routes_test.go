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

// TestChatRoutesAnswerWhenFlagIsOff checks chat, the calendar, and slides
// answer with their flags stored off (#2577, #2610, #2869).
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
	if err := settingsutil.SetFeatureFlag("slides", false); err != nil {
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

	do := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+admin.SessionToken)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		return w
	}

	chat := do(http.MethodGet, "/api/v0/chat/channels")
	if chat.Code != http.StatusOK || strings.Contains(chat.Body.String(), "turned off") {
		t.Fatalf("GET /chat/channels = %d %s, want 200 with the flag stored off", chat.Code, chat.Body.String())
	}
	calendar := do(http.MethodGet, "/api/v0/calendar/events?from=2026-10-01T00:00:00Z&to=2026-10-08T00:00:00Z")
	if calendar.Code != http.StatusOK || strings.Contains(calendar.Body.String(), "turned off") {
		t.Fatalf("GET /calendar/events = %d %s, want 200 with the flag stored off", calendar.Code, calendar.Body.String())
	}
	// No filePath: the handler runs and refuses the request. A flag gate
	// would have answered 404 "turned off" before that.
	exportSlides := do(http.MethodGet, "/api/v0/files/export/pptx")
	if exportSlides.Code != http.StatusBadRequest || strings.Contains(exportSlides.Body.String(), "turned off") {
		t.Fatalf("GET /files/export/pptx = %d %s, want 400 with the flag gone", exportSlides.Code, exportSlides.Body.String())
	}
	importSlides := do(http.MethodPost, "/api/v0/files/import/pptx")
	if importSlides.Code != http.StatusBadRequest || strings.Contains(importSlides.Body.String(), "turned off") {
		t.Fatalf("POST /files/import/pptx = %d %s, want 400 with the flag gone", importSlides.Code, importSlides.Body.String())
	}
}
