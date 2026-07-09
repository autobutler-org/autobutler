package shareutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// useTempStore points the package at a temp shares.json and restores the
// default path when the test finishes.
func useTempStore(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shares.json")
	ResetForTesting(path)
	t.Cleanup(func() { ResetForTesting("") })
	return path
}

func TestCreateAndResolve(t *testing.T) {
	useTempStore(t)

	res, err := Create(CreateShareParams{FilePath: "docs/report.pdf", DeviceSerial: "abc"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if res.Share.Token == "" || res.Share.ID == "" {
		t.Fatal("expected non-empty token and id")
	}

	got, err := Resolve(res.Share.Token, "")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if got.FilePath != "docs/report.pdf" || got.DeviceSerial != "abc" {
		t.Errorf("unexpected share: %+v", got)
	}
	if got.AccessCount != 1 || got.LastAccessAt == nil {
		t.Errorf("expected access bookkeeping, got count=%d last=%v", got.AccessCount, got.LastAccessAt)
	}
}

func TestCreate_RequiresFilePath(t *testing.T) {
	useTempStore(t)
	if _, err := Create(CreateShareParams{}); err == nil {
		t.Fatal("expected error for empty filePath")
	}
}

func TestResolve_UnknownToken(t *testing.T) {
	useTempStore(t)
	if _, err := Resolve("nope", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if _, err := Resolve("", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for empty token, got %v", err)
	}
}

func TestResolve_Expired(t *testing.T) {
	useTempStore(t)
	past := time.Now().Add(-time.Hour)
	res, err := Create(CreateShareParams{FilePath: "a.txt", ExpiresAt: &past})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := Resolve(res.Share.Token, ""); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
	if _, err := Peek(res.Share.Token); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired from Peek, got %v", err)
	}
}

func TestResolve_PasswordFlow(t *testing.T) {
	useTempStore(t)
	res, err := Create(CreateShareParams{FilePath: "a.txt", Password: "hunter22"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := Resolve(res.Share.Token, ""); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
	if _, err := Resolve(res.Share.Token, "wrong"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
	got, err := Resolve(res.Share.Token, "hunter22")
	if err != nil {
		t.Fatalf("Resolve with correct password failed: %v", err)
	}
	if !got.PasswordProtected() {
		t.Error("expected PasswordProtected true")
	}
	// Peek must work without the password but keep the hash out of reach of
	// access bookkeeping (no count increment).
	peeked, err := Peek(res.Share.Token)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if peeked.AccessCount != 1 {
		t.Errorf("Peek should not increment access count, got %d", peeked.AccessCount)
	}
}

func TestListAndDelete(t *testing.T) {
	useTempStore(t)
	first, _ := Create(CreateShareParams{FilePath: "one.txt"})
	second, _ := Create(CreateShareParams{FilePath: "two.txt"})

	shares, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(shares) != 2 {
		t.Fatalf("expected 2 shares, got %d", len(shares))
	}
	if shares[0].ID != second.Share.ID {
		t.Error("expected newest-first ordering")
	}

	if err := Delete(first.Share.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if err := Delete(first.Share.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on double delete, got %v", err)
	}
	shares, _ = List()
	if len(shares) != 1 || shares[0].ID != second.Share.ID {
		t.Errorf("unexpected shares after delete: %+v", shares)
	}
}

func TestPersistenceAcrossReload(t *testing.T) {
	path := useTempStore(t)
	res, err := Create(CreateShareParams{FilePath: "keep.txt"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Simulate process restart: drop cache, same file.
	ResetForTesting(path)
	got, err := Resolve(res.Share.Token, "")
	if err != nil {
		t.Fatalf("Resolve after reload failed: %v", err)
	}
	if got.FilePath != "keep.txt" {
		t.Errorf("unexpected share after reload: %+v", got)
	}

	// File must be 0600 — it holds tokens and password hashes.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat shares file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected shares file mode 0600, got %o", info.Mode().Perm())
	}
}
