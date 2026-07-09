package v1_shares_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1_shares "github.com/autobutler-org/autobutler/internal/server/api/v1/shares"
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/shareutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
	"github.com/gin-gonic/gin"
)

// fakeDetector implements storageutil.Detector and returns a single internal
// device pointing at the provided temp directory (same pattern as the cirrus
// integration tests).
type fakeDetector struct {
	mountPoint string
}

func (f *fakeDetector) DetectDevices() ([]storageutil.Device, error) {
	return []storageutil.Device{
		{
			Name:       "Test Device",
			MountPoint: f.mountPoint,
			IsInternal: true,
		},
	}, nil
}

// newTestEngine creates a gin engine with the shares routes registered, a
// fake StorageService pointing at a temp cirrus dir, and the share store
// pointed at a temp shares.json.
func newTestEngine(t *testing.T) (*gin.Engine, string) {
	t.Helper()

	mountPoint := t.TempDir()
	cirrusDir := filepath.Join(mountPoint, "autobutler", "data", "cirrus")
	if err := os.MkdirAll(cirrusDir, 0755); err != nil {
		t.Fatalf("failed to create cirrus dir: %v", err)
	}

	shareutil.ResetForTesting(filepath.Join(t.TempDir(), "shares.json"))
	t.Cleanup(func() { shareutil.ResetForTesting("") })

	svc := storageutil.NewStorageService(&fakeDetector{mountPoint: mountPoint})
	deps := deputil.NewDependencies().WithStorageService(svc)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	})
	group := engine.Group("/api/v1")
	serverutil.RegisterRouterWithGroup(group, v1_shares.NewRouter())
	return engine, cirrusDir
}

func doRequest(engine *gin.Engine, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// createShare posts a share definition and returns the parsed response.
func createShare(t *testing.T, engine *gin.Engine, body map[string]any) map[string]any {
	t.Helper()
	payload, _ := json.Marshal(body)
	w := doRequest(engine, http.MethodPost, "/api/v1/shares", bytes.NewReader(payload))
	if w.Code != http.StatusOK {
		t.Fatalf("create share returned %d: %s", w.Code, w.Body.String())
	}
	var share map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &share); err != nil {
		t.Fatalf("failed to parse share response: %v", err)
	}
	return share
}

func writeTestFile(t *testing.T, cirrusDir, name, content string) {
	t.Helper()
	full := filepath.Join(cirrusDir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestShareLifecycle_File(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	writeTestFile(t, cirrusDir, "docs/report.txt", "hello share")

	share := createShare(t, engine, map[string]any{"filePath": "docs/report.txt"})
	urlPath, _ := share["urlPath"].(string)
	if urlPath == "" {
		t.Fatal("expected urlPath in create response")
	}

	// Public info
	w := doRequest(engine, http.MethodGet, urlPath+"/info", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("info returned %d: %s", w.Code, w.Body.String())
	}
	var info map[string]any
	json.Unmarshal(w.Body.Bytes(), &info)
	if info["name"] != "report.txt" || info["isFolder"] != false {
		t.Errorf("unexpected info: %v", info)
	}

	// Public download
	w = doRequest(engine, http.MethodGet, urlPath, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("download returned %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "hello share" {
		t.Errorf("unexpected download body: %q", w.Body.String())
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected nosniff, got %q", got)
	}
	if got := w.Header().Get("Content-Security-Policy"); got != "sandbox" {
		t.Errorf("expected CSP sandbox, got %q", got)
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="report.txt"` {
		t.Errorf("unexpected disposition: %q", got)
	}

	// List shows the share with access count
	w = doRequest(engine, http.MethodGet, "/api/v1/shares", nil)
	var shares []map[string]any
	json.Unmarshal(w.Body.Bytes(), &shares)
	if len(shares) != 1 {
		t.Fatalf("expected 1 share, got %d", len(shares))
	}
	if shares[0]["accessCount"].(float64) != 1 {
		t.Errorf("expected accessCount 1, got %v", shares[0]["accessCount"])
	}
	if _, leaked := shares[0]["passwordHash"]; leaked {
		t.Error("passwordHash must not appear in API responses")
	}

	// Revoke, then the public link dies
	id, _ := share["id"].(string)
	w = doRequest(engine, http.MethodDelete, "/api/v1/shares/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete returned %d: %s", w.Code, w.Body.String())
	}
	w = doRequest(engine, http.MethodGet, urlPath, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after revoke, got %d", w.Code)
	}
}

func TestShareDownload_FolderZips(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	writeTestFile(t, cirrusDir, "album/one.txt", "1")
	writeTestFile(t, cirrusDir, "album/two.txt", "2")

	share := createShare(t, engine, map[string]any{"filePath": "album"})
	w := doRequest(engine, http.MethodGet, share["urlPath"].(string), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("folder download returned %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename="album.zip"` {
		t.Errorf("unexpected disposition: %q", got)
	}
	// Zip magic bytes
	if body := w.Body.Bytes(); len(body) < 4 || body[0] != 'P' || body[1] != 'K' {
		t.Error("expected zip content")
	}
}

func TestShare_PasswordFlow(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	writeTestFile(t, cirrusDir, "secret.txt", "classified")

	share := createShare(t, engine, map[string]any{"filePath": "secret.txt", "password": "letmein1"})
	urlPath := share["urlPath"].(string)

	// No password → 401
	w := doRequest(engine, http.MethodGet, urlPath, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without password, got %d", w.Code)
	}

	// Wrong password → 403
	req := httptest.NewRequest(http.MethodGet, urlPath, nil)
	req.Header.Set("X-Share-Password", "wrong")
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 with wrong password, got %d", w.Code)
	}

	// Info without password must not leak the filename
	w = doRequest(engine, http.MethodGet, urlPath+"/info", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("info returned %d", w.Code)
	}
	var info map[string]any
	json.Unmarshal(w.Body.Bytes(), &info)
	if name, ok := info["name"]; ok && name != "" {
		t.Errorf("info must not leak name without password, got %v", name)
	}
	if info["passwordProtected"] != true {
		t.Error("expected passwordProtected true")
	}

	// Correct password via header → content
	req = httptest.NewRequest(http.MethodGet, urlPath, nil)
	req.Header.Set("X-Share-Password", "letmein1")
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "classified" {
		t.Fatalf("expected content with correct password, got %d: %s", w.Code, w.Body.String())
	}

	// Correct password via query also works
	w = doRequest(engine, http.MethodGet, urlPath+"?password=letmein1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with query password, got %d", w.Code)
	}
}

func TestShare_Expiry(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	writeTestFile(t, cirrusDir, "old.txt", "stale")

	// Create an already-expired share directly in the store.
	past := time.Now().Add(-time.Minute)
	res, err := shareutil.Create(shareutil.CreateShareParams{FilePath: "old.txt", ExpiresAt: &past})
	if err != nil {
		t.Fatal(err)
	}

	w := doRequest(engine, http.MethodGet, "/api/v1/public/shares/"+res.Share.Token, nil)
	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 for expired share, got %d", w.Code)
	}
	w = doRequest(engine, http.MethodGet, "/api/v1/public/shares/"+res.Share.Token+"/info", nil)
	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 for expired info, got %d", w.Code)
	}

	// Expired shares still appear in the owner's list, flagged.
	w = doRequest(engine, http.MethodGet, "/api/v1/shares", nil)
	var shares []map[string]any
	json.Unmarshal(w.Body.Bytes(), &shares)
	if len(shares) != 1 || shares[0]["expired"] != true {
		t.Errorf("expected one expired share in list, got %v", shares)
	}
}

func TestCreateShare_Validation(t *testing.T) {
	engine, cirrusDir := newTestEngine(t)
	writeTestFile(t, cirrusDir, "real.txt", "x")

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing filePath", map[string]any{}, http.StatusBadRequest},
		{"negative expiry", map[string]any{"filePath": "real.txt", "expiresInHours": -1}, http.StatusBadRequest},
		{"nonexistent file", map[string]any{"filePath": "ghost.txt"}, http.StatusNotFound},
		{"traversal", map[string]any{"filePath": "../../etc/passwd"}, http.StatusBadRequest},
		{"absolute path", map[string]any{"filePath": "/etc/passwd"}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.body)
			w := doRequest(engine, http.MethodPost, "/api/v1/shares", bytes.NewReader(payload))
			if w.Code != tc.want {
				t.Errorf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestPublicShare_UnknownToken(t *testing.T) {
	engine, _ := newTestEngine(t)
	w := doRequest(engine, http.MethodGet, "/api/v1/public/shares/doesnotexist", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPublicShare_RejectsTamperedStorePath(t *testing.T) {
	engine, _ := newTestEngine(t)

	// Simulate a hand-edited shares.json containing a traversal path — the
	// public endpoints must reject it before touching the filesystem.
	res, err := shareutil.Create(shareutil.CreateShareParams{FilePath: "../../etc/passwd"})
	if err != nil {
		t.Fatal(err)
	}
	w := doRequest(engine, http.MethodGet, "/api/v1/public/shares/"+res.Share.Token, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for tampered path on download, got %d", w.Code)
	}
	w = doRequest(engine, http.MethodGet, "/api/v1/public/shares/"+res.Share.Token+"/info", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for tampered path on info, got %d", w.Code)
	}
}
