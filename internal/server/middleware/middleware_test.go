package middleware

import (
	"testing"
)

// --- allowsQueryParamToken ---

func TestAllowsQueryParamToken_AllowedPaths(t *testing.T) {
	allowed := []string{
		"/api/v0/events",
		"/api/v0/cirrus/download",
		"/api/v0/cirrus/download-archive",
		"/api/v0/photos/thumbnail",
	}
	for _, path := range allowed {
		if !allowsQueryParamToken(path) {
			t.Errorf("expected %q to allow ?token=, but it does not", path)
		}
	}
}

func TestAllowsQueryParamToken_DeniedPaths(t *testing.T) {
	denied := []string{
		"/api/v0/auth/login",
		"/api/v0/auth/setup",
		"/api/v0/cirrus",
		"/api/v0/cirrus/upload",
		"/api/v0/photos",
		"/api/v0/photos/favorite",
		"/api/v0/albums",
		"/api/v0/health",
		"/api/v0/settings",
		"/api/v0/devices",
		"/api/v0/shares",
		"",
		"/",
	}
	for _, path := range denied {
		if allowsQueryParamToken(path) {
			t.Errorf("expected %q to deny ?token=, but it is allowed", path)
		}
	}
}

func TestAllowsQueryParamToken_NoPrefixMatch(t *testing.T) {
	// Paths that start with an allowed path but aren't exact matches
	// should not be allowed — map lookup is exact.
	notAllowed := []string{
		"/api/v0/events/extra",
		"/api/v0/cirrus/download/something",
		"/api/v0/photos/thumbnail/123",
	}
	for _, path := range notAllowed {
		if allowsQueryParamToken(path) {
			t.Errorf("expected %q to deny ?token= (not an exact match), but it is allowed", path)
		}
	}
}
