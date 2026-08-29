package vaultutil

import (
	"testing"
)

func TestDetectFormat_JSON(t *testing.T) {
	data := []byte(`{"entries": []}`)
	if got := DetectFormat(data); got != "json" {
		t.Errorf("expected json, got %s", got)
	}
	data = []byte(`[{"name": "test"}]`)
	if got := DetectFormat(data); got != "json" {
		t.Errorf("expected json for array, got %s", got)
	}
}

func TestDetectFormat_Bitwarden(t *testing.T) {
	data := []byte("folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n")
	if got := DetectFormat(data); got != "bitwarden" {
		t.Errorf("expected bitwarden, got %s", got)
	}
}

func TestDetectFormat_GenericCSV(t *testing.T) {
	data := []byte("url,username,password\nhttps://example.com,alice,secret\n")
	if got := DetectFormat(data); got != "csv" {
		t.Errorf("expected csv, got %s", got)
	}
}

func TestParseBitwardenCSV(t *testing.T) {
	csv := `folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp
Social,,login,GitHub,some notes,,0,https://github.com/login,alice,gh-pass,JBSWY3DPEHPK3PXP
Banking,,login,Chase,,,,https://chase.com,bob,chase-pw,
`
	entries, errs := ParseBitwardenCSV([]byte(csv))
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	gh := entries[0]
	if gh.Name != "GitHub" {
		t.Errorf("name = %q, want GitHub", gh.Name)
	}
	if gh.URL != "https://github.com/login" {
		t.Errorf("url = %q", gh.URL)
	}
	if gh.Username != "alice" {
		t.Errorf("username = %q", gh.Username)
	}
	if gh.Password != "gh-pass" {
		t.Errorf("password = %q", gh.Password)
	}
	if gh.Notes != "some notes" {
		t.Errorf("notes = %q", gh.Notes)
	}
	if gh.TOTPSecret != "JBSWY3DPEHPK3PXP" {
		t.Errorf("totp = %q", gh.TOTPSecret)
	}
	if gh.Folder != "Social" {
		t.Errorf("folder = %q", gh.Folder)
	}
}

func TestParseGenericCSV(t *testing.T) {
	csv := `url,username,password
https://example.com,alice,secret
https://test.com,bob,pass123
`
	entries, errs := ParseGenericCSV([]byte(csv))
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "example.com" {
		t.Errorf("name should be derived from URL host, got %q", entries[0].Name)
	}
	if entries[0].Username != "alice" {
		t.Errorf("username = %q", entries[0].Username)
	}
}

func TestParseGenericCSV_ChromeFormat(t *testing.T) {
	csv := `name,url,username,password,note
GitHub,https://github.com,alice,gh-pass,my notes
`
	entries, errs := ParseGenericCSV([]byte(csv))
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "GitHub" {
		t.Errorf("name = %q", entries[0].Name)
	}
	if entries[0].Notes != "my notes" {
		t.Errorf("notes = %q", entries[0].Notes)
	}
}

func TestParseGenericCSV_MissingPassword(t *testing.T) {
	csv := `url,username,password
https://example.com,alice,
`
	entries, errs := ParseGenericCSV([]byte(csv))
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d", len(errs))
	}
}

func TestParseQuarkJSON(t *testing.T) {
	j := `{
		"entries": [
			{"name": "GitHub", "url": "https://github.com", "username": "alice", "password": "pw", "folderName": "Dev"},
			{"name": "Gmail", "url": "https://gmail.com", "username": "bob", "password": "gm"}
		],
		"folders": ["Dev"]
	}`
	entries, errs := ParseQuarkJSON([]byte(j))
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Folder != "Dev" {
		t.Errorf("folder = %q, want Dev", entries[0].Folder)
	}
}

func TestParseQuarkJSON_Array(t *testing.T) {
	j := `[{"name": "Test", "url": "https://test.com", "username": "u", "password": "p"}]`
	entries, errs := ParseQuarkJSON([]byte(j))
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestDedupKey(t *testing.T) {
	k1 := dedupKey("GitHub", "github.com")
	k2 := dedupKey("github", "GitHub.com")
	if k1 != k2 {
		t.Error("dedupKey should be case-insensitive")
	}
}

func TestHostFromURL(t *testing.T) {
	tests := []struct{ input, want string }{
		{"https://github.com/login", "github.com"},
		{"http://localhost:8080/path", "localhost"},
		{"", ""},
		{"not a url", ""},
	}
	for _, tt := range tests {
		if got := HostFromURL(tt.input); got != tt.want {
			t.Errorf("HostFromURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
