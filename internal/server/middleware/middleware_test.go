package middleware

import "testing"

// TestQueryParamTokenPaths verifies that the ?token= query-param allowlist
// contains exactly the paths that require it (SSE event stream) and does not
// include ordinary API endpoints.
func TestQueryParamTokenPaths(t *testing.T) {
	allowed := []string{
		"/api/v0/events",
	}
	for _, p := range allowed {
		if !queryParamTokenPaths[p] {
			t.Errorf("expected %q to be in queryParamTokenPaths", p)
		}
	}

	// Ordinary endpoints must NOT be in the allowlist.
	rejected := []string{
		"/api/v0/cirrus",
		"/api/v0/auth/login",
		"/api/v0/photos",
		"/api/v0/vault/entries",
		"/api/v1/vfs/files",
	}
	for _, p := range rejected {
		if queryParamTokenPaths[p] {
			t.Errorf("expected %q to NOT be in queryParamTokenPaths", p)
		}
	}
}
