package server

import (
	"context"
	"log"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/diskprofiler"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

// sessionPurgeInterval is how often expired sessions are deleted from the DB.
// Sessions expire after 30 days; purging every 6 hours keeps the table small
// without hammering SQLite unnecessarily.
const sessionPurgeInterval = 6 * time.Hour

// sessionPurgeTimeout returns the per-purge context timeout tuned to the
// storage device speed. Slow SD cards get more time to avoid killing a
// legitimate delete mid-flight; fast SSDs get a tighter bound.
//
// Falls back to 30 s when profiling fails.
func sessionPurgeTimeout(_ deputil.Dependencies) time.Duration {
	dataDir := storageutil.GetDataDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tier, err := diskprofiler.Profile(ctx, dataDir)
	if err != nil {
		log.Printf("[session-purger] diskprofiler: %v — using default timeout", err)
		return 30 * time.Second
	}
	return tier.DeleteTimeout()
}

// startSessionPurger runs DeleteExpiredSessions once at startup, then
// periodically every sessionPurgeInterval. It is a no-op when no database
// is configured. Runs as a background goroutine for the server lifetime.
func startSessionPurger(deps deputil.Dependencies) {
	dbConn := deps.Database()
	if dbConn == nil || dbConn.Queries == nil {
		return
	}
	q := dbConn.Queries
	timeout := sessionPurgeTimeout(deps)
	log.Printf("[session-purger] purge timeout: %s", timeout)

	purge := func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
