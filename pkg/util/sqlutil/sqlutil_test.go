package sqlutil_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/sqlutil"
)

// ── NullInt64 ────────────────────────────────────────────────────────────────

func TestNullInt64_Nil(t *testing.T) {
	got := sqlutil.NullInt64(nil)
	if got.Valid {
		t.Errorf("NullInt64(nil): expected Valid=false, got Valid=true")
	}
	if got.Int64 != 0 {
		t.Errorf("NullInt64(nil): expected Int64=0, got %d", got.Int64)
	}
}

func TestNullInt64_NonNil(t *testing.T) {
	v := int64(42)
	got := sqlutil.NullInt64(&v)
	if !got.Valid {
		t.Errorf("NullInt64(&42): expected Valid=true, got Valid=false")
	}
	if got.Int64 != 42 {
		t.Errorf("NullInt64(&42): expected Int64=42, got %d", got.Int64)
	}
}

func TestNullInt64_Zero(t *testing.T) {
	v := int64(0)
	got := sqlutil.NullInt64(&v)
	if !got.Valid {
		t.Errorf("NullInt64(&0): expected Valid=true, got Valid=false")
	}
	if got.Int64 != 0 {
		t.Errorf("NullInt64(&0): expected Int64=0, got %d", got.Int64)
	}
}

func TestNullInt64_Negative(t *testing.T) {
	v := int64(-1)
	got := sqlutil.NullInt64(&v)
	if !got.Valid {
		t.Errorf("NullInt64(&-1): expected Valid=true, got Valid=false")
	}
	if got.Int64 != -1 {
		t.Errorf("NullInt64(&-1): expected Int64=-1, got %d", got.Int64)
	}
}

// ── FormatTime ───────────────────────────────────────────────────────────────

func TestFormatTime_UTC(t *testing.T) {
	// A known instant: 2024-01-15 12:34:56 UTC
	ts := time.Date(2024, 1, 15, 12, 34, 56, 0, time.UTC)
	got := sqlutil.FormatTime(ts)
	want := "2024-01-15T12:34:56Z"
	if got != want {
		t.Errorf("FormatTime(%v): got %q, want %q", ts, got, want)
	}
}

func TestFormatTime_NonUTCNormalisedToUTC(t *testing.T) {
	// A time in US/Pacific (UTC-8 in winter). The output must always be UTC.
	loc := time.FixedZone("PST", -8*3600)
	ts := time.Date(2024, 6, 1, 0, 0, 0, 0, loc) // midnight PST = 08:00 UTC
	got := sqlutil.FormatTime(ts)
	want := "2024-06-01T08:00:00Z"
	if got != want {
		t.Errorf("FormatTime(%v): got %q, want %q", ts, got, want)
	}
}

func TestFormatTime_Epoch(t *testing.T) {
	ts := time.Unix(0, 0).UTC()
	got := sqlutil.FormatTime(ts)
	want := "1970-01-01T00:00:00Z"
	if got != want {
		t.Errorf("FormatTime(epoch): got %q, want %q", got, want)
	}
}

func TestFormatTime_RFC3339Format(t *testing.T) {
	ts := time.Now().UTC()
	got := sqlutil.FormatTime(ts)
	// Must be parseable as RFC3339.
	if _, err := time.Parse(time.RFC3339, got); err != nil {
		t.Errorf("FormatTime result %q is not valid RFC3339: %v", got, err)
	}
}

// ── NullStringPtr ─────────────────────────────────────────────────────────────

func TestNullStringPtr_Invalid(t *testing.T) {
	ns := sql.NullString{String: "", Valid: false}
	got := sqlutil.NullStringPtr(ns)
	if got != nil {
		t.Errorf("NullStringPtr(invalid): expected nil, got %q", *got)
	}
}

func TestNullStringPtr_ValidNonEmpty(t *testing.T) {
	ns := sql.NullString{String: "hello", Valid: true}
	got := sqlutil.NullStringPtr(ns)
	if got == nil {
		t.Fatal("NullStringPtr(valid): expected non-nil pointer, got nil")
	}
	if *got != "hello" {
		t.Errorf("NullStringPtr(valid): expected %q, got %q", "hello", *got)
	}
}

func TestNullStringPtr_ValidEmpty(t *testing.T) {
	// A valid empty string is still a non-nil pointer to "".
	ns := sql.NullString{String: "", Valid: true}
	got := sqlutil.NullStringPtr(ns)
	if got == nil {
		t.Fatal("NullStringPtr(valid empty): expected non-nil pointer, got nil")
	}
	if *got != "" {
		t.Errorf("NullStringPtr(valid empty): expected empty string, got %q", *got)
	}
}

func TestNullStringPtr_ValidWithSpecialChars(t *testing.T) {
	ns := sql.NullString{String: "café & <tags>", Valid: true}
	got := sqlutil.NullStringPtr(ns)
	if got == nil {
		t.Fatal("NullStringPtr: expected non-nil pointer, got nil")
	}
	if *got != "café & <tags>" {
		t.Errorf("NullStringPtr: expected %q, got %q", "café & <tags>", *got)
	}
}
