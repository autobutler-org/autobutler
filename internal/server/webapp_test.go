package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
)

// TestServeWebApp pins how the web build goes out (#3208): compressed by
// Accept-Encoding, revalidated by ETag, and index.html for any other path.
func TestServeWebApp(t *testing.T) {
	mainJS := strings.Repeat("console.log('quark');\n", 1000)
	index := "<!doctype html><title>Quark</title>"
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.NoRoute(serveWebApp(fstest.MapFS{
		"index.html":   {Data: []byte(index)},
		"main.dart.js": {Data: []byte(mainJS)},
		"icon.png":     {Data: []byte("\x89PNG not really")},
	}))
	get := func(path string, header map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		for k, v := range header {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		return rec
	}
	decode := map[string]func(io.Reader) (io.Reader, error){
		"gzip": func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) },
		"br":   func(r io.Reader) (io.Reader, error) { return brotli.NewReader(r), nil },
	}

	for encoding, newReader := range decode {
		t.Run("main.dart.js as "+encoding, func(t *testing.T) {
			rec := get("/main.dart.js", map[string]string{"Accept-Encoding": encoding})
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d", rec.Code)
			}
			if got := rec.Header().Get("Content-Encoding"); got != encoding {
				t.Fatalf("Content-Encoding %q, want %q", got, encoding)
			}
			if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Errorf("Vary %q", got)
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") {
				t.Errorf("Content-Type %q", got)
			}
			if rec.Body.Len() >= len(mainJS)/4 {
				t.Errorf("served %d bytes of %d, not compressed", rec.Body.Len(), len(mainJS))
			}
			r, err := newReader(rec.Body)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(r)
			if err != nil || string(body) != mainJS {
				t.Fatalf("decoded body differs (err %v)", err)
			}
		})
	}

	t.Run("q=0 refuses an encoding", func(t *testing.T) {
		rec := get("/main.dart.js", map[string]string{"Accept-Encoding": "br;q=0, gzip;q=0.5"})
		if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("Content-Encoding %q, want gzip", got)
		}
	})

	t.Run("identity without Accept-Encoding", func(t *testing.T) {
		rec := get("/main.dart.js", nil)
		if rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != mainJS {
			t.Fatalf("Content-Encoding %q, %d bytes", rec.Header().Get("Content-Encoding"), rec.Body.Len())
		}
	})

	t.Run("revalidates with the ETag", func(t *testing.T) {
		for _, path := range []string{"/main.dart.js", "/", "/files/photos", "/icon.png"} {
			rec := get(path, map[string]string{"Accept-Encoding": "gzip, br"})
			etag := rec.Header().Get("ETag")
			if rec.Code != http.StatusOK || etag == "" {
				t.Fatalf("%s: status %d, ETag %q", path, rec.Code, etag)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
				t.Errorf("%s: Cache-Control %q, want no-cache", path, got)
			}
			again := get(path, map[string]string{"Accept-Encoding": "gzip, br", "If-None-Match": etag})
			if again.Code != http.StatusNotModified || again.Body.Len() != 0 {
				t.Errorf("%s: revalidation got %d with %d bytes, want 304", path, again.Code, again.Body.Len())
			}
		}
	})

	t.Run("a client route gets index.html", func(t *testing.T) {
		rec := get("/settings/general", map[string]string{"Accept-Encoding": "gzip"})
		r, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(r)
		if rec.Code != http.StatusOK || string(body) != index {
			t.Fatalf("status %d, body %q", rec.Code, body)
		}
	})

	t.Run("images are not recompressed", func(t *testing.T) {
		rec := get("/icon.png", map[string]string{"Accept-Encoding": "gzip, br"})
		if got := rec.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("Content-Encoding %q on a png", got)
		}
	})

	t.Run("a range gets identity bytes", func(t *testing.T) {
		rec := get("/main.dart.js", map[string]string{"Accept-Encoding": "gzip", "Range": "bytes=0-6"})
		if rec.Code != http.StatusPartialContent || rec.Body.String() != "console" {
			t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
		}
	})
}

// TestServeWebAppWithoutBuild keeps a dev server with no built frontend
// answering 404 rather than failing.
func TestServeWebAppWithoutBuild(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.NoRoute(serveWebApp(fstest.MapFS{"stub.txt": {}}))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}
