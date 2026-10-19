package v0_thumbnails_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_thumbnails "github.com/autobutler-org/quark/internal/server/api/v0/thumbnails"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/iosemutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// TestGetThumbnail_RetryContract pins the response a client uses to decide
// whether to retry: only a temporarily busy thumbnail carries Retry-After.
func TestGetThumbnail_RetryContract(t *testing.T) {
	for _, tt := range []struct {
		name       string
		source     []byte
		status     int
		retryAfter string
		busy       bool
	}{
		{name: "missing", status: http.StatusNotFound},
		{name: "corrupt", source: []byte("not a PNG"), status: http.StatusInternalServerError},
		{name: "busy", source: pngBytes(t), status: http.StatusServiceUnavailable, retryAfter: "5", busy: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			fsys := vfs.NewMemVFS(vfs.FilesNamespace(""))
			if tt.source != nil {
				if err := fsys.Write(context.Background(), "photo.png", bytes.NewReader(tt.source), vfs.WriteOptions{}); err != nil {
					t.Fatal(err)
				}
			}
			registry := vfs.NewRegistry()
			if err := registry.Register(vfs.Namespace{ID: vfs.FilesNamespace("")}, fsys); err != nil {
				t.Fatal(err)
			}
			deps := deputil.NewDependencies().
				WithDatabase(dbtest.NewDB(t)).
				WithVFSRegistry(registry)
			if tt.busy {
				sem := iosemutil.NewWithConcurrency(1)
				if !sem.AcquireDefault(context.Background()) {
					t.Fatal("failed to occupy IO semaphore")
				}
				t.Cleanup(sem.Release)
				deps = deps.WithIOSemaphore(sem)
			}

			gin.SetMode(gin.TestMode)
			engine := gin.New()
			engine.Use(func(c *gin.Context) {
				c = ctxutil.With(c, "deps", deps)
				c = ctxutil.With(c, "principal", accessutil.System)
				c.Next()
			})
			serverutil.RegisterRouterWithGroup(engine.Group("/api/v0"), v0_thumbnails.NewRouter())

			req := httptest.NewRequest(http.MethodGet, "/api/v0/thumbnails/photo.png", nil)
			if tt.busy {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", w.Code, tt.status, w.Body.String())
			}
			if got := w.Header().Get("Retry-After"); got != tt.retryAfter {
				t.Errorf("Retry-After = %q, want %q", got, tt.retryAfter)
			}
		})
	}
}
