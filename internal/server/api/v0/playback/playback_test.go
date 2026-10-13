package v0_playback_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_playback "github.com/autobutler-org/quark/internal/server/api/v0/playback"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/playbackutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// systemDevice is the internal drive at "/", whose files directory is the one
// storageutil.GetFilesDir resolves under HOME, served as the "files"
// namespace.
type systemDevice struct{}

func (systemDevice) DetectDevices() ([]storageutil.Device, error) {
	return []storageutil.Device{{Name: "Internal", MountPoint: "/", IsInternal: true}}, nil
}

// playbackHarness serves the playback routes over a real files directory and
// a migrated database, acting as whoever principal names.
type playbackHarness struct {
	engine    *gin.Engine
	principal *accessutil.Principal
	founder   accessutil.Principal
	user      accessutil.Principal
}

func newPlaybackHarness(t *testing.T) playbackHarness {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	if _, err := storageutil.GetFilesDir(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	database := dbtest.NewDB(t)
	founder, err := database.Queries.CreateUser(ctx, db.CreateUserParams{Username: "founder", PasswordHash: "h", RecoveryPhraseHash: "r"})
	if err != nil {
		t.Fatal(err)
	}
	user, err := database.Queries.CreateUser(ctx, db.CreateUserParams{Username: "bob", PasswordHash: "h", RecoveryPhraseHash: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Queries.SetUserPathAccess(ctx, db.SetUserPathAccessParams{
		RelPath: "shared",
		UserID:  sql.NullInt64{Int64: user.ID, Valid: true},
		Level:   accessutil.Read.String(),
	}); err != nil {
		t.Fatal(err)
	}
	storage := storageutil.NewStorageService(systemDevice{})
	files := vfs.NewStorageServiceVFS(storage, vfs.FilesNamespace(""))
	registry := vfs.NewRegistry()
	if err := registry.Register(vfs.Namespace{ID: vfs.FilesNamespace("")}, files); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"shared/clip.mp4", "private/clip.mp4"} {
		if err := files.Write(ctx, rel, strings.NewReader("not really a video"), vfs.WriteOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	deps := deputil.NewDependencies().
		WithStorageService(storage).
		WithVFSRegistry(registry).
		WithMetadataStore(vfs.NewSQLiteMetadataStore(database.Db)).
		WithDatabase(database)

	h := playbackHarness{
		principal: &accessutil.Principal{},
		founder:   accessutil.Principal{UserID: founder.ID, IsAdmin: true},
		user:      accessutil.Principal{UserID: user.ID},
	}
	*h.principal = h.founder
	gin.SetMode(gin.TestMode)
	h.engine = gin.New()
	h.engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c = ctxutil.With(c, "principal", *h.principal)
		c.Next()
	})
	serverutil.RegisterRouterWithGroup(h.engine.Group("/api/v0"), v0_playback.NewRouter())
	return h
}

func (h playbackHarness) do(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.engine.ServeHTTP(w, req)
	return w
}

// state sends a request that has to succeed and decodes the state it answers
// with.
func (h playbackHarness) state(t *testing.T, method, path, body string) playbackutil.State {
	t.Helper()
	w := h.do(method, path, body)
	var state playbackutil.State
	if err := json.Unmarshal(w.Body.Bytes(), &state); w.Code != http.StatusOK || err != nil {
		t.Fatalf("%s %s = %d %s: %v", method, path, w.Code, w.Body.String(), err)
	}
	return state
}

const getShared = "/api/v0/playback?relPath=shared/clip.mp4"

func TestPlayback_RoundTrip(t *testing.T) {
	h := newPlaybackHarness(t)

	w := h.do(http.MethodGet, getShared, "")
	if w.Code != http.StatusOK || strings.TrimSpace(w.Body.String()) != `{"positionMs":0,"durationMs":0,"watched":false}` {
		t.Fatalf("GET with nothing stored = %d %s, want the zero state without updatedAt", w.Code, w.Body.String())
	}

	saved := h.state(t, http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","deviceSerial":"","positionMs":30000,"durationMs":100000}`)
	if saved.PositionMs != 30000 || saved.DurationMs != 100000 || saved.Watched || saved.UpdatedAt == "" {
		t.Errorf("PUT = %+v, want 30000 of 100000, unwatched, with updatedAt", saved)
	}
	if got := h.state(t, http.MethodGet, getShared, ""); got != saved {
		t.Errorf("GET after PUT = %+v, want %+v", got, saved)
	}

	if got := h.state(t, http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","positionMs":96000,"durationMs":100000}`); !got.Watched || got.PositionMs != 0 {
		t.Errorf("PUT near the end = %+v, want watched at 0", got)
	}

	if got := h.state(t, http.MethodPut, "/api/v0/playback/watched", `{"relPath":"shared/clip.mp4","watched":false}`); got != (playbackutil.State{}) {
		t.Errorf("PUT watched=false = %+v, want the zero state", got)
	}
	if got := h.state(t, http.MethodGet, getShared, ""); got != (playbackutil.State{}) {
		t.Errorf("GET after marking unwatched = %+v, want the zero state", got)
	}

	h.state(t, http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","positionMs":30000,"durationMs":100000}`)
	marked := h.state(t, http.MethodPut, "/api/v0/playback/watched", `{"relPath":"shared/clip.mp4","watched":true}`)
	if !marked.Watched || marked.PositionMs != 0 || marked.DurationMs != 100000 {
		t.Errorf("PUT watched=true = %+v, want watched at 0 of 100000", marked)
	}
}

func TestPlayback_StateIsPerUser(t *testing.T) {
	h := newPlaybackHarness(t)
	saved := h.state(t, http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","positionMs":30000,"durationMs":100000}`)

	*h.principal = h.user
	if got := h.state(t, http.MethodGet, getShared, ""); got != (playbackutil.State{}) {
		t.Errorf("another user sees %+v, want the zero state", got)
	}
	h.state(t, http.MethodPut, "/api/v0/playback/watched", `{"relPath":"shared/clip.mp4","watched":true}`)

	*h.principal = h.founder
	if got := h.state(t, http.MethodGet, getShared, ""); got != saved {
		t.Errorf("state after another user's change = %+v, want %+v", got, saved)
	}
}

func TestPlayback_BadRequest(t *testing.T) {
	h := newPlaybackHarness(t)
	for name, req := range map[string][3]string{
		"get without relPath":     {http.MethodGet, "/api/v0/playback", ""},
		"save without relPath":    {http.MethodPut, "/api/v0/playback", `{"positionMs":1,"durationMs":2}`},
		"save with a bad body":    {http.MethodPut, "/api/v0/playback", `{"relPath":`},
		"save with a string":      {http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","positionMs":"1"}`},
		"negative position":       {http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","positionMs":-1,"durationMs":100}`},
		"negative duration":       {http.MethodPut, "/api/v0/playback", `{"relPath":"shared/clip.mp4","positionMs":1,"durationMs":-100}`},
		"watched without relPath": {http.MethodPut, "/api/v0/playback/watched", `{"watched":true}`},
		"watched with a bad body": {http.MethodPut, "/api/v0/playback/watched", `nope`},
	} {
		if w := h.do(req[0], req[1], req[2]); w.Code != http.StatusBadRequest {
			t.Errorf("%s = %d %s, want 400", name, w.Code, w.Body.String())
		}
	}
	if got := h.state(t, http.MethodGet, getShared, ""); got != (playbackutil.State{}) {
		t.Errorf("state after rejected requests = %+v, want none", got)
	}
}

func TestPlayback_NotFound(t *testing.T) {
	h := newPlaybackHarness(t)
	notFound := func(who string, relPath string) {
		t.Helper()
		for name, req := range map[string][3]string{
			"get":     {http.MethodGet, "/api/v0/playback?relPath=" + relPath, ""},
			"save":    {http.MethodPut, "/api/v0/playback", `{"relPath":"` + relPath + `","positionMs":1,"durationMs":100}`},
			"watched": {http.MethodPut, "/api/v0/playback/watched", `{"relPath":"` + relPath + `","watched":true}`},
		} {
			if w := h.do(req[0], req[1], req[2]); w.Code != http.StatusNotFound {
				t.Errorf("%s %s as %s = %d %s, want 404", name, relPath, who, w.Code, w.Body.String())
			}
		}
	}
	// A file that is not there, a folder, a path that climbs out of the files
	// directory, and a device that is not attached.
	notFound("the founder", "shared/missing.mp4")
	notFound("the founder", "shared")
	notFound("the founder", "../../etc/passwd")
	if w := h.do(http.MethodGet, getShared+"&deviceSerial=GONE", ""); w.Code != http.StatusNotFound {
		t.Errorf("get on an unknown device = %d %s, want 404", w.Code, w.Body.String())
	}

	// A file the caller may not read answers exactly as a missing one does.
	*h.principal = h.user
	notFound("a user without access", "private/clip.mp4")
	notFound("a user without access", "shared/missing.mp4")
	if got := h.state(t, http.MethodGet, getShared, ""); got != (playbackutil.State{}) {
		t.Errorf("a readable video = %+v, want the zero state", got)
	}
	*h.principal = h.founder
	if got := h.state(t, http.MethodGet, "/api/v0/playback?relPath=private/clip.mp4", ""); got != (playbackutil.State{}) {
		t.Errorf("refused writes left %+v behind, want none", got)
	}
}
