package frame_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/server/frame"
	"github.com/gin-gonic/gin"
)

func engine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	frame.Register(e)
	return e
}

func get(e *gin.Engine, target string, header ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestServesThePageAndItsAssets(t *testing.T) {
	cases := []struct {
		target      string
		contentType string
		contains    string
	}{
		{"/frame", "text/html", "<title>Quark Frame</title>"},
		{"/frame/", "text/html", "<title>Quark Frame</title>"},
		{"/frame/frame.js", "javascript", "deriveAuthKey"},
		{"/frame/frame.css", "text/css", "object-fit: contain"},
		{"/frame/manifest.webmanifest", "application/manifest+json", `"start_url": "/frame"`},
	}
	e := engine()
	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			rec := get(e, tc.target)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Header().Get("Content-Type"); !strings.Contains(got, tc.contentType) {
				t.Errorf("Content-Type = %q, want it to contain %q", got, tc.contentType)
			}
			if !strings.Contains(rec.Body.String(), tc.contains) {
				t.Errorf("body does not contain %q", tc.contains)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
		})
	}
}

func TestOnlyKnownAssetsAreServed(t *testing.T) {
	e := engine()
	for _, target := range []string{
		"/frame/missing.js",
		"/frame/frame.txt",
		"/frame/%2e%2e/helpers.go",
		"/frame/assets/frame.js/../../frame.go",
		"/frame/helpers.go",
	} {
		if rec := get(e, target); rec.Code != http.StatusNotFound && rec.Code != http.StatusMovedPermanently {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}

func TestAssetsRevalidateWithAnETag(t *testing.T) {
	e := engine()
	first := get(e, "/frame/frame.js")
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the first response")
	}
	if got := first.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", got)
	}
	second := get(e, "/frame/frame.js", "If-None-Match", etag)
	if second.Code != http.StatusNotModified {
		t.Fatalf("status with a matching ETag = %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Errorf("a 304 carried %d body bytes", second.Body.Len())
	}
}

// The page exists because the Flutter web client is too heavy for a small screen (#3018), so its weight is a
// requirement and not a nicety. The budget is a ceiling on everything the page loads before it signs in.
func TestPageWeightStaysSmall(t *testing.T) {
	const budget = 40 * 1024
	e := engine()
	total := 0
	for _, target := range []string{"/frame", "/frame/frame.js", "/frame/frame.css", "/frame/manifest.webmanifest"} {
		total += get(e, target).Body.Len()
	}
	if total > budget {
		t.Fatalf("the page loads %d bytes, over the %d byte budget", total, budget)
	}
}

var modernSyntax = regexp.MustCompile("=>|`|\\basync\\b|\\bawait\\b|\\blet\\b|\\bconst\\b|\\bclass\\b")

// frame.js promises ES5 plus Promise and fetch, so a browser a few years old can still run it. This catches the
// syntax that breaks that promise; it does not prove the page works in any one browser.
func TestScriptSticksToPlainES5(t *testing.T) {
	src, err := os.ReadFile("assets/frame.js")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		code := line
		if at := strings.Index(code, "//"); at >= 0 {
			code = code[:at]
		}
		// Words inside a quoted message are copy, not syntax.
		code = regexp.MustCompile(`'[^']*'|"[^"]*"`).ReplaceAllString(code, "''")
		if m := modernSyntax.FindString(code); m != "" {
			t.Errorf("frame.js:%d uses %q, which is newer than ES5: %s", i+1, m, trimmed)
		}
	}
}
