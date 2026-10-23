package sqlutil_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/sqlutil"
)

func TestNullInt64_Nil(t *testing.T) {
	got := sqlutil.NullInt64(nil)
	if got.Valid {
		t.Errorf("NullInt64(nil): expected Valid=false, got Valid=true")
	}
	if got.Int64 != 0 {
		t.Errorf("NullInt64(nil): expected Int64=0, got %d", got.Int64)
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

func TestFormatTime_NonUTCNormalizedToUTC(t *testing.T) {
	loc := time.FixedZone("PST", -8*3600)
	ts := time.Date(2024, 6, 1, 0, 0, 0, 0, loc)
	got := sqlutil.FormatTime(ts)
	want := "2024-06-01T08:00:00Z"
	if got != want {
		t.Errorf("FormatTime(%v): got %q, want %q", ts, got, want)
	}
}

func TestNullStringPtr_Invalid(t *testing.T) {
	ns := sql.NullString{String: "", Valid: false}
	got := sqlutil.NullStringPtr(ns)
	if got != nil {
		t.Errorf("NullStringPtr(invalid): expected nil, got %q", *got)
	}
}

func TestNullStringPtr_ValidEmpty(t *testing.T) {
	ns := sql.NullString{String: "", Valid: true}
	got := sqlutil.NullStringPtr(ns)
	if got == nil {
		t.Fatal("NullStringPtr(valid empty): expected non-nil pointer, got nil")
	}
	if *got != "" {
		t.Errorf("NullStringPtr(valid empty): expected empty string, got %q", *got)
	}
}
