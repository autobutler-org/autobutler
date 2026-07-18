package server

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/deputil"
)

// startSessionCleaner runs a background goroutine that deletes expired sessions
// from the database every 24 hours. A random jitter of up to 5 minutes is added
// at startup so that restarted instances don't all hit the DB at the same second.
//
// There is no security gap without this — GetSession already filters on
// expires_at — but without periodic cleanup the sessions table grows forever.
//
// The goroutine exits when ctx is cancelled (e.g. on graceful server shutdown).
func startSessionCleaner(ctx context.Context, deps deputil.Dependencies) {
	jitter := time.Duration(rand.Intn(5*60)) * time.Second //nolint:gosec // jitter, not crypto
	select {
	case <-ctx.Done():
		return
	case <-time.After(jitter):
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	cleanup := func() {
		db := deps.Database()
		if db == nil {
			return
		}
		if err := db.Queries.DeleteExpiredSessions(ctx); err != nil {
			slog.Warn("[session-cleanup] failed to purge expired sessions", "err", err)
		} else {
			slog.Debug("[session-cleanup] expired sessions purged")
		}
	}

	cleanup() // Run once immediately after jitter.
	for {
		select {
		case <-ctx.Done():
			slog.Debug("[session-cleanup] shutting down")
			return
		case <-ticker.C:
			cleanup()
		}
	}
}
