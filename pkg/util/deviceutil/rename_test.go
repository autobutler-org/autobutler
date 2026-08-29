package deviceutil

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRename_RejectsUnusableNames(t *testing.T) {
	for name, given := range map[string]string{
		"empty":             "",
		"whitespace only":   "   \t ",
		"too long":          strings.Repeat("a", MaxDeviceNameLength+1),
		"control character": "my\x00drive",
	} {
		t.Run(name, func(t *testing.T) {
			// Queries stays nil: every rule rejects before the write.
			if _, err := Rename(RenameParams{Ctx: context.Background(), Name: given}); !errors.Is(err, ErrInvalidDeviceName) {
				t.Fatalf("expected ErrInvalidDeviceName, got %v", err)
			}
		})
	}
}

func TestRename_StoresTrimmedName(t *testing.T) {
	queries := newTestDBWithRoles(t)

	result, err := Rename(RenameParams{
		Ctx:     context.Background(),
		Queries: queries,
		Serial:  "USB-001",
		Name:    "  My Drive  ",
	})
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if result.Name != "My Drive" {
		t.Errorf("expected trimmed name, got %q", result.Name)
	}

	stored, err := queries.GetDeviceName(context.Background(), "USB-001")
	if err != nil {
		t.Fatalf("GetDeviceName failed: %v", err)
	}
	if stored != "My Drive" {
		t.Errorf("expected stored name %q, got %q", "My Drive", stored)
	}
}

func TestRename_AcceptsNameAtTheLengthLimit(t *testing.T) {
	// The limit counts runes, so a name of multi-byte characters is measured
	// the way the person typing it sees it.
	name := strings.Repeat("é", MaxDeviceNameLength)
	result, err := Rename(RenameParams{
		Ctx:     context.Background(),
		Queries: newTestDBWithRoles(t),
		Serial:  "USB-001",
		Name:    name,
	})
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if result.Name != name {
		t.Errorf("expected the name unchanged, got %q", result.Name)
	}
}
