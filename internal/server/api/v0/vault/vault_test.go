package v0_vault

import (
	"database/sql"
	"strings"
	"testing"
)

// --- nullableInt64 / fromNullInt64 ---

func TestNullableInt64_Nil(t *testing.T) {
	got := nullableInt64(nil)
	if got.Valid {
		t.Error("expected invalid NullInt64 for nil pointer")
	}
}

func TestNullableInt64_Value(t *testing.T) {
	v := int64(42)
	got := nullableInt64(&v)
	if !got.Valid || got.Int64 != 42 {
		t.Errorf("expected valid NullInt64{42}, got %+v", got)
	}
}

func TestFromNullInt64_Invalid(t *testing.T) {
	got := fromNullInt64(sql.NullInt64{})
	if got != nil {
		t.Errorf("expected nil for invalid NullInt64, got %v", got)
	}
}

func TestFromNullInt64_Valid(t *testing.T) {
	got := fromNullInt64(sql.NullInt64{Int64: 7, Valid: true})
	if got == nil || *got != 7 {
		t.Errorf("expected 7, got %v", got)
	}
}

func TestNullableInt64_RoundTrip(t *testing.T) {
	v := int64(99)
	out := fromNullInt64(nullableInt64(&v))
	if out == nil || *out != 99 {
		t.Errorf("round-trip failed: got %v", out)
	}
	out2 := fromNullInt64(nullableInt64(nil))
	if out2 != nil {
		t.Errorf("round-trip nil failed: got %v", out2)
	}
}

// --- extractURLHost ---

func TestExtractURLHost_Empty(t *testing.T) {
	if got := extractURLHost(""); got != "" {
		t.Errorf("expected '', got %q", got)
	}
}

func TestExtractURLHost_HTTP(t *testing.T) {
	got := extractURLHost("http://example.com/path?q=1")
	if got != "example.com" {
		t.Errorf("expected 'example.com', got %q", got)
	}
}

func TestExtractURLHost_HTTPS(t *testing.T) {
	got := extractURLHost("https://accounts.google.com/login")
	if got != "accounts.google.com" {
		t.Errorf("expected 'accounts.google.com', got %q", got)
	}
}

func TestExtractURLHost_PortStripped(t *testing.T) {
	got := extractURLHost("https://example.com:8443/path")
	if got != "example.com" {
		t.Errorf("expected 'example.com' (no port), got %q", got)
	}
}

func TestExtractURLHost_InvalidURL_NoGoroutinePanic(t *testing.T) {
	// url.Parse is lenient; just ensure no panic on bizarre input.
	got := extractURLHost("://no-scheme")
	_ = got
}

// --- detectFormat edge cases not covered by import_export_test.go ---

func TestDetectFormat_WhitespaceTrimmed(t *testing.T) {
	data := []byte("   \n   {\"entries\":[]}")
	if got := detectFormat(data); got != "json" {
		t.Errorf("expected 'json' after trimming whitespace, got %q", got)
	}
}

// --- parseBitwardenCSV edge cases ---

func TestParseBitwardenCSV_NoDataRows(t *testing.T) {
	_, errs := parseBitwardenCSV([]byte("header1,header2\n"))
	if len(errs) == 0 {
		t.Error("expected error for no data rows")
	}
}

func TestParseBitwardenCSV_BothNameAndURLEmpty_Skipped(t *testing.T) {
	csv := "folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n" +
		",0,login,,,,,,u,p,"
	entries, errs := parseBitwardenCSV([]byte(csv))
	if len(entries) != 0 {
		t.Errorf("expected row with no name and no URL to be skipped, got %d entries", len(entries))
	}
	if len(errs) == 0 {
		t.Error("expected an error for the skipped row")
	}
}

func TestParseBitwardenCSV_EmptyNameUsesHost(t *testing.T) {
	csv := "folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n" +
		",0,login,,,,,https://example.com,u,p,"
	entries, _ := parseBitwardenCSV([]byte(csv))
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "example.com" {
		t.Errorf("expected 'example.com' as fallback name, got %q", entries[0].Name)
	}
}

// --- parseGenericCSV edge cases ---

func TestParseGenericCSV_NoDataRows(t *testing.T) {
	_, errs := parseGenericCSV([]byte("header1,header2\n"))
	if len(errs) == 0 {
		t.Error("expected error for no data rows")
	}
}

func TestParseGenericCSV_EmptyNameNoURLFallsToPlaceholder(t *testing.T) {
	csv := "name,url,username,password\n,, alice,pass\n"
	entries, _ := parseGenericCSV([]byte(csv))
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// Name should be "Entry 2" when URL is also empty
	if !strings.HasPrefix(entries[0].Name, "Entry") {
		t.Errorf("expected placeholder name starting with 'Entry', got %q", entries[0].Name)
	}
}

// --- secureRandomString ---

func TestSecureRandomString_Length(t *testing.T) {
	for _, l := range []int{1, 8, 20, 64, 128} {
		s, err := secureRandomString("abcdefghijklmnopqrstuvwxyz", l)
		if err != nil {
			t.Fatalf("length %d: %v", l, err)
		}
		if len(s) != l {
			t.Errorf("expected len=%d, got %d", l, len(s))
		}
	}
}

func TestSecureRandomString_CharsetConstrained(t *testing.T) {
	charset := "abc"
	s, err := secureRandomString(charset, 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range s {
		if !strings.ContainsRune(charset, ch) {
			t.Errorf("unexpected character %q in output", ch)
		}
	}
}

func TestSecureRandomString_InvalidLength(t *testing.T) {
	if _, err := secureRandomString("abc", 0); err == nil {
		t.Error("expected error for length=0")
	}
	if _, err := secureRandomString("abc", 129); err == nil {
		t.Error("expected error for length=129")
	}
}

func TestSecureRandomString_Unique(t *testing.T) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s1, _ := secureRandomString(charset, 20)
	s2, _ := secureRandomString(charset, 20)
	if s1 == s2 {
		t.Error("two random strings matched — RNG not functioning")
	}
}
