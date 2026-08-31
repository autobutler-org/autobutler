package vaultutil_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/vaultutil"
)

// An import file the parsers can actually handle passes straight through.
func TestReadImportAcceptsAnOrdinaryFile(t *testing.T) {
	content := "name,url,username,password\nExample,https://example.com,me,secret\n"

	data, err := vaultutil.ReadImport(strings.NewReader(content))
	if err != nil {
		t.Fatalf("ReadImport: %v", err)
	}
	if string(data) != content {
		t.Errorf("content: got %q, want %q", data, content)
	}
}

// The endpoint used to io.ReadAll an uploaded file with no cap at all, so any
// client could make the server allocate a body of its choosing (#1723). An
// oversized import must be refused, not truncated into a partial import.
func TestReadImportRefusesAnOversizedFile(t *testing.T) {
	oversized := strings.NewReader(strings.Repeat("a", int(vaultutil.MaxImportBytes)+1))

	if _, err := vaultutil.ReadImport(oversized); !errors.Is(err, vaultutil.ErrImportTooLarge) {
		t.Fatalf("want ErrImportTooLarge, got %v", err)
	}
}

// Exactly at the limit is still an import, not an error — an off-by-one here
// would reject a legitimate file.
func TestReadImportAcceptsExactlyTheLimit(t *testing.T) {
	atLimit := strings.NewReader(strings.Repeat("a", int(vaultutil.MaxImportBytes)))

	data, err := vaultutil.ReadImport(atLimit)
	if err != nil {
		t.Fatalf("a file exactly at the limit must be accepted, got %v", err)
	}
	if int64(len(data)) != vaultutil.MaxImportBytes {
		t.Errorf("got %d bytes, want %d", len(data), vaultutil.MaxImportBytes)
	}
}
