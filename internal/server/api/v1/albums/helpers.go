package v1_albums

import (
	"database/sql"
	"time"
)

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func nullInt64(id *int64) sql.NullInt64 {
	if id == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *id, Valid: true}
}
