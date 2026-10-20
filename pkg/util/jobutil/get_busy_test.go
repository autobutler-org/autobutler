package jobutil

import (
	"context"
	"testing"
	"time"
)

// TestGetReadsAJobWhileAWriterHoldsTheDatabasePastOneBusyTimeout is the poll
// GET /jobs/{id} does while a conversion writes progress. The database is
// opened with busy_timeout, so a reader waits five seconds and then
// SQLITE_BUSY. A writer that commits a moment later — a progress update
// whose fsync outlasted that wait — must not fail the poll (#3542).
func TestGetReadsAJobWhileAWriterHoldsTheDatabasePastOneBusyTimeout(t *testing.T) {
	h := newHarness(t)
	res, err := h.queue.Enqueue(context.Background(), EnqueueParams{
		Kind: testKind, Name: "clip", Params: map[string]string{},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var timeoutMS int
	if err := h.database.Db.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeoutMS); err != nil {
		t.Fatal(err)
	}
	if timeoutMS < 1000 {
		t.Fatalf("busy_timeout = %dms, want the production wait", timeoutMS)
	}
	// Past one busy wait, inside a second attempt of the same length.
	hold := time.Duration(timeoutMS)*time.Millisecond + time.Second

	conn, err := h.database.Db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	t.Cleanup(func() {
		<-released
		conn.Close()
	})
	if _, err := conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer close(released)
		time.Sleep(hold)
		if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
			t.Errorf("commit: %v", err)
		}
	}()

	start := time.Now()
	got, err := h.queue.Get(ctx, GetParams{ID: res.Job.ID})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("get after %s, with the writer holding the database for %s: %v", elapsed, hold, err)
	}
	if got.Job.ID != res.Job.ID {
		t.Fatalf("job id = %d, want %d", got.Job.ID, res.Job.ID)
	}
	if elapsed < time.Duration(timeoutMS)*time.Millisecond {
		t.Fatalf("get returned in %s, before one busy timeout (%s) could elapse", elapsed, time.Duration(timeoutMS)*time.Millisecond)
	}
}
