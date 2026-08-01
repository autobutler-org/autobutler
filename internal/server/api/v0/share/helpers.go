package v0_share

import (
	"database/sql"

	"github.com/autobutler-org/autobutler/internal/db"
)

func dbDeleteShareLinkParams(tokenHash string, userID int64) db.DeleteShareLinkParams {
	return db.DeleteShareLinkParams{
		TokenHash: tokenHash,
		CreatedBy: sql.NullInt64{Int64: userID, Valid: true},
	}
}
