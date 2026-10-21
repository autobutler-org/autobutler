package v0_photos_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_photos "github.com/autobutler-org/quark/internal/server/api/v0/photos"
	"github.com/autobutler-org/quark/internal/server/middleware"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/photoutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// newIndexEngine mounts the photos admin router behind RequireAdmin, the way
// routes.go does, with requests coming from *username.
func newIndexEngine(t *testing.T, username *string) (*gin.Engine, deputil.Dependencies) {
	t.Helper()
	ctx := context.Background()
	database := dbtest.NewDB(t)
	for _, name := range []string{"admin", "member"} {
		if _, err := database.Queries.CreateUser(ctx, db.CreateUserParams{Username: name, PasswordHash: "h", RecoveryPhraseHash: "r"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.Queries.PromoteToAdmin(ctx, "admin"); err != nil {
		t.Fatal(err)
	}
	deps := deputil.NewDependencies().WithDatabase(database).WithEventBus(eventbus.New())
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c = ctxutil.With(c, "principal", accessutil.Principal{IsAdmin: *username == "admin"})
		c = ctxutil.With(c, "username", *username)
		c.Next()
	})
	group := engine.Group("/api/v0")
	serverutil.RegisterRouterWithGroup(group.Group("", middleware.RequireAdmin(deps)), v0_photos.NewAdminRouter())
	return engine, deps
}

func doIndexReq(t *testing.T, engine *gin.Engine, method, path string) (int, photoutil.IndexStatusResult) {
	t.Helper()
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	var status photoutil.IndexStatusResult
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatalf("decode %s: %v", w.Body.String(), err)
		}
	}
	return w.Code, status
}

// TestPhotoIndex_StatusPauseResume: an admin reads the index status, pauses
// the pass and resumes it, and each change is announced; a member is refused
// all three (#3556).
func TestPhotoIndex_StatusPauseResume(t *testing.T) {
	username := "member"
	engine, deps := newIndexEngine(t, &username)
	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/api/v0/photos/index-status"},
		{http.MethodPost, "/api/v0/photos/index/pause"},
		{http.MethodPost, "/api/v0/photos/index/resume"},
	} {
		if code, _ := doIndexReq(t, engine, r.method, r.path); code != http.StatusForbidden {
			t.Errorf("member %s %s = %d, want 403", r.method, r.path, code)
		}
	}

	username = "admin"
	events, unsubscribe := deps.EventBus().Subscribe("test")
	defer unsubscribe()
	expectEvent := func(after string) {
		t.Helper()
		select {
		case evt := <-events:
			if evt.Kind != eventbus.EventPhotoIndexChanged {
				t.Errorf("after %s: event %q, want photo_index_changed", after, evt.Kind)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("after %s: no photo_index_changed", after)
		}
	}

	code, status := doIndexReq(t, engine, http.MethodGet, "/api/v0/photos/index-status")
	if code != http.StatusOK || status.Running || status.Paused || status.LastRunAt != nil || status.Failures == nil {
		t.Fatalf("status = %d %+v, want 200, idle, never run, an empty failure list", code, status)
	}
	if code, status = doIndexReq(t, engine, http.MethodPost, "/api/v0/photos/index/pause"); code != http.StatusOK || !status.Paused {
		t.Fatalf("pause = %d %+v, want 200 and paused", code, status)
	}
	expectEvent("pause")
	if _, status = doIndexReq(t, engine, http.MethodGet, "/api/v0/photos/index-status"); !status.Paused {
		t.Errorf("status after pause = %+v, want paused", status)
	}
	if code, status = doIndexReq(t, engine, http.MethodPost, "/api/v0/photos/index/resume"); code != http.StatusOK || status.Paused {
		t.Fatalf("resume = %d %+v, want 200 and not paused", code, status)
	}
	expectEvent("resume")
}

// TestPhotoIndex_StatusReportsThePass: what the background pass counted is
// what the status route returns.
func TestPhotoIndex_StatusReportsThePass(t *testing.T) {
	username := "admin"
	engine, deps := newIndexEngine(t, &username)
	ctx := context.Background()
	reg := vfs.NewRegistry()
	mem := vfs.NewMemVFS("files")
	if err := reg.Register(vfs.Namespace{ID: "files", Description: "files"}, mem); err != nil {
		t.Fatal(err)
	}
	var photo bytes.Buffer
	if err := jpeg.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 16, 16)), nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"/a.jpg", "/broken.jpg"} {
		body := photo.Bytes()
		if name == "/broken.jpg" {
			body = []byte("not a jpeg")
		}
		if err := mem.Write(ctx, name, bytes.NewReader(body), vfs.WriteOptions{ContentType: "image/jpeg"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := photoutil.BackfillHashes(photoutil.BackfillHashesParams{
		Ctx: ctx, Queries: deps.Database().Queries, Registry: reg,
		Control: deps.PhotoIndex(), EventBus: deps.EventBus(),
	}); err != nil {
		t.Fatal(err)
	}
	code, status := doIndexReq(t, engine, http.MethodGet, "/api/v0/photos/index-status")
	want := []photoutil.IndexFailure{{RelPath: "broken.jpg", Reason: photoutil.IndexFailureUndecodable}}
	if code != http.StatusOK || status.Running || status.LastRunAt == nil ||
		status.Total != 2 || status.Hashed != 1 || status.DatesRead != 2 || status.Failed != 1 ||
		!reflect.DeepEqual(status.Failures, want) {
		t.Errorf("status = %d %+v, want 200 and a finished pass: 2 photos, 1 hashed, 2 dated, broken.jpg undecodable", code, status)
	}
}
