package v0_files_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1_files "github.com/autobutler-org/quark/internal/server/api/v0/files"
)

// Tests for the deprecated /cirrus -> /files alias (#1601). Delete alongside
// the shim in legacy_cirrus_alias.go.

const (
	canonicalPrefix = "/files"
	legacyPrefix    = "/cirrus"
)

type routeKey struct {
	method string
	path   string
}

// TestLegacyCirrusAlias_MirrorsEveryCanonicalRoute asserts the alias set is a
// exact 1:1 mirror: every /files route has a /cirrus twin with the same method
// and the same path suffix, and nothing else is registered.
func TestLegacyCirrusAlias_MirrorsEveryCanonicalRoute(t *testing.T) {
	routes := v1_files.NewRouter().Routes()

	canonical := make(map[routeKey]bool)
	legacy := make(map[routeKey]bool)
	for _, route := range routes {
		key := routeKey{method: route.Method, path: route.Path}
		switch {
		case strings.HasPrefix(route.Path, canonicalPrefix):
			canonical[key] = true
		case strings.HasPrefix(route.Path, legacyPrefix):
			legacy[key] = true
		default:
			t.Errorf("route %s %s uses neither the canonical %q nor the legacy %q prefix",
				route.Method, route.Path, canonicalPrefix, legacyPrefix)
		}
	}

	if len(canonical) == 0 {
		t.Fatal("no canonical /files routes registered")
	}
	if len(legacy) != len(canonical) {
		t.Errorf("expected one legacy alias per canonical route: %d canonical, %d legacy",
			len(canonical), len(legacy))
	}

	for key := range canonical {
		want := routeKey{
			method: key.method,
			path:   legacyPrefix + strings.TrimPrefix(key.path, canonicalPrefix),
		}
		if !legacy[want] {
			t.Errorf("canonical route %s %s has no legacy alias %s %s",
				key.method, key.path, want.method, want.path)
		}
	}
}

// TestLegacyCirrusAlias_ServesSameResponse checks that a GET through the legacy
// prefix produces the same status and body as the canonical path, and that only
// the legacy response is marked deprecated.
func TestLegacyCirrusAlias_ServesSameResponse(t *testing.T) {
	engine, filesDir := newTestEngine(t)

	if err := os.WriteFile(filepath.Join(filesDir, "shared.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	canonicalResp := doRequest(engine, http.MethodGet, "/api/v0/files", nil, "")
	legacyResp := doRequest(engine, http.MethodGet, "/api/v0/cirrus", nil, "")

	if legacyResp.Code != canonicalResp.Code {
		t.Fatalf("legacy status %d != canonical status %d: %s",
			legacyResp.Code, canonicalResp.Code, legacyResp.Body.String())
	}
	if legacyResp.Body.String() != canonicalResp.Body.String() {
		t.Errorf("legacy body differs from canonical:\nlegacy:    %s\ncanonical: %s",
			legacyResp.Body.String(), canonicalResp.Body.String())
	}
	if !strings.Contains(legacyResp.Body.String(), "shared.txt") {
		t.Errorf("expected shared.txt in listing, got: %s", legacyResp.Body.String())
	}

	if got := legacyResp.Header().Get("Deprecation"); got != "true" {
		t.Errorf("expected Deprecation: true on legacy response, got %q", got)
	}
	if got := legacyResp.Header().Get("Link"); got != `</api/v0/files>; rel="successor-version"` {
		t.Errorf("unexpected Link header on legacy response: %q", got)
	}

	if got := canonicalResp.Header().Get("Deprecation"); got != "" {
		t.Errorf("canonical response should not be marked deprecated, got %q", got)
	}
}

// TestLegacyCirrusAlias_SuccessorLinkKeepsSubPath checks the Link header names
// the actual canonical URL of the request, not just the route prefix.
func TestLegacyCirrusAlias_SuccessorLinkKeepsSubPath(t *testing.T) {
	engine, filesDir := newTestEngine(t)

	if err := os.WriteFile(filepath.Join(filesDir, "stat-me.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resp := doRequest(engine, http.MethodGet, "/api/v0/cirrus/stat?filePath=stat-me.txt", nil, "")
	if resp.Code != http.StatusOK {
		t.Fatalf("legacy stat returned %d: %s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Link"); got != `</api/v0/files/stat>; rel="successor-version"` {
		t.Errorf("unexpected Link header: %q", got)
	}
}

// TestLegacyCirrusAlias_PostBodySurvives is the reason the alias serves the
// handler directly instead of redirecting: a 3xx would strip this multipart
// body before it ever reached the handler.
func TestLegacyCirrusAlias_PostBodySurvives(t *testing.T) {
	engine, _ := newTestEngine(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	mw.WriteField("folderName", "made-via-alias")
	mw.Close()

	resp := doRequest(engine, http.MethodPost, "/api/v0/cirrus/folder//", &buf, mw.FormDataContentType())
	if resp.Code != http.StatusOK {
		t.Fatalf("create folder via alias returned %d: %s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Deprecation"); got != "true" {
		t.Errorf("expected Deprecation: true, got %q", got)
	}

	// The folder is visible through the canonical listing, so both prefixes hit
	// the same storage.
	if !contains(fileNames(listFiles(t, engine, "")), "made-via-alias") {
		t.Errorf("folder created via alias not found in canonical listing: %v",
			fileNames(listFiles(t, engine, "")))
	}
}

// TestLegacyCirrusAlias_NestedUploadPath covers the /files//upload/*rootDir
// route, whose literal double slash must survive the prefix swap.
func TestLegacyCirrusAlias_NestedUploadPath(t *testing.T) {
	engine, _ := newTestEngine(t)

	resp := uploadFile(t, engine, "/api/v0/cirrus/upload/docs", "readme.txt", "docs content")
	if resp.Code != http.StatusOK {
		t.Fatalf("nested upload via alias returned %d: %s", resp.Code, resp.Body.String())
	}

	if !contains(fileNames(listFiles(t, engine, "")), "docs") {
		t.Errorf("expected 'docs' folder after upload via alias, got: %v",
			fileNames(listFiles(t, engine, "")))
	}
}
