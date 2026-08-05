package middleware

import "testing"

// TestQueryTokenAllowed verifies that only the designated paths allow
// ?token= query-parameter auth. All other paths must be rejected to prevent
// session tokens from leaking into access logs, proxy logs, and browser history.
func TestQueryTokenAllowed(t *testing.T) {
	allowed := []string{
		"/api/v0/events",         // WebSocket — browsers cannot set headers on upgrade
		"/api/v0/cirrus",         // file download / streaming (src= attribute usage)
		"/api/v0/cirrus/photo",   // photo sub-path
		"/api/v0/photos",         // photo serving
		"/api/v0/photos/thumb",   // thumbnail sub-path
		"/videos/",               // video deep-link player
		"/audio/",                // audio deep-link player
	}
	forbidden := []string{
		"/api/v0/auth/login",
		"/api/v0/health",
		"/api/v0/storage/devices/status",
		"/api/v1/vfs",
		"/api/v0/albums",
		"/api/v0/books",
		"/api/v0/favorites",
		"/api/v0/migration/run",
		"/dav/",
	}

	for _, p := range allowed {
		if !queryTokenAllowed(p) {
			t.Errorf("path %q should be allowed for ?token= auth but queryTokenAllowed returned false", p)
		}
	}
	for _, p := range forbidden {
		if queryTokenAllowed(p) {
			t.Errorf("path %q should NOT be allowed for ?token= auth but queryTokenAllowed returned true", p)
		}
	}
}
