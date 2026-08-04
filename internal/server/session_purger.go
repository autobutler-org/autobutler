package server

import (
	"context"
	"log"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/deputil"
)

// sessionPurgeInterval is how often expired sessions are deleted from the DB.
// Sessions expire after 30 days; purging every 6 hours keeps the table small
// without hammering SQLite unnecessarily.
const sessionPurgeInterval = 6 * time.Hour

// startSessionPurger runs DeleteExpiredSessions once at startup, then
// periodically every sessionPurgeInterval. It is a no-op when no database
// is configured. Runs as a background goroutine for the server lifetime.
func startSessionPurger(deps deputil.Dependencies) {
	dbConn := deps.Database()
	if dbConn == nil || dbConn.Queries == nil {
		return
	}
	q := dbConn.Queries

	purge := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := q.DeleteExpiredSessions(ctx); err != nil {
			log.Printf("[session-purger] delete expired sessions: %v", err)
		}
	}

	// Run once immediately so stale sessions left over from previous runs are
	// cleaned up before the first request is served.
	purge()

	ticker := time.NewTicker(sessionPurgeInterval)
	defer ticker.Stop()
	for range ticker.C {
		purge()
	}
}
