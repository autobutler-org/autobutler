package vaultutil

import (
	"database/sql"
	"time"
)

func nullableInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func fromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func formatTimestamp(t time.Time) string {
	return t.UTC().Format(timestampFormat)
}
