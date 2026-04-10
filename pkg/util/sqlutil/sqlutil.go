package sqlutil

import (
	"database/sql"
	"time"
)

// NullInt64 converts a *int64 to a sql.NullInt64 for use in sqlc-generated queries.
func NullInt64(id *int64) sql.NullInt64 {
	if id == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *id, Valid: true}
}

// FormatTime formats a time.Time as an RFC3339 UTC string for JSON responses.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
