package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestFramePageIsMountedAheadOfTheWebApp checks the server answers /frame with the slideshow page (#3021), not with the
// web app's index.html fallback, which would show the Flutter client on a screen too weak to run it.
func TestFramePageIsMountedAheadOfTheWebApp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := setupStaticRoutes(engine); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/frame", "/frame/"} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>Quark Frame</title>") {
			t.Errorf("GET %s = %d, want the slideshow page: %.80s", target, rec.Code, rec.Body.String())
		}
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/frame/frame.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
		t.Errorf("GET /frame/frame.js = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}
