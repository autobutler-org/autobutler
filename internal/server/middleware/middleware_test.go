package middleware

import "testing"

func TestPathAllowsQueryToken(t *testing.T) {
	cases := []struct {
		path    string
		allowed bool
	}{
		// Streaming endpoints that legitimately need ?token=
		{"/api/v0/events", true},
		{"/api/v0/events?token=abc", true}, // path with query — HasPrefix matches on path
		{"/api/v0/cirrus/download", true},
		{"/api/v0/cirrus/download?f=photo.jpg", true},
		{"/api/v1/vfs/files/video.mp4", true},
		{"/api/v1/vfs", true},

		// All other API endpoints must NOT accept ?token=
		{"/api/v0/cirrus", false},
		{"/api/v0/cirrus/search", false},
		{"/api/v0/auth/login", false},
		{"/api/v0/auth/setup", false},
		{"/api/v0/settings", false},
		{"/api/v0/health", false},
		{"/api/v1/auth/login", false},
		{"/api/v0/cirrus/list", false},
		{"/api/v0/vault/unlock", false},
		{"", false},
		{"/", false},
	}
	for _, c := range cases {
		got := pathAllowsQueryToken(c.path)
		if got != c.allowed {
			t.Errorf("pathAllowsQueryToken(%q) = %v, want %v", c.path, got, c.allowed)
		}
	}
}
