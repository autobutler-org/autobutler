package middleware

import "testing"

// TestTokenQueryParamAllowed verifies that ?token= is only accepted on
// streaming endpoints, not on arbitrary API paths.
func TestTokenQueryParamAllowed(t *testing.T) {
	tests := []struct {
		path    string
		allowed bool
	}{
		// Allowed: WebSocket / event-stream endpoints
		{"/api/v0/events", true},
		{"/api/v1/events", true},
		{"/api/v0/events/", true},

		// Not allowed: everything else
		{"/api/v0/auth/login", false},
		{"/api/v0/cirrus/download", false},
		{"/api/v0/cirrus/upload", false},
		{"/api/v1/auth/login", false},
		{"/api/v1/users/me", false},
		{"/dav/files", false},
		{"/", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := tokenQueryParamAllowed(tt.path)
			if got != tt.allowed {
				t.Errorf("tokenQueryParamAllowed(%q) = %v; want %v", tt.path, got, tt.allowed)
			}
		})
	}
}
