package musicutil_test

import (
	"context"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/jobutil"
	"github.com/autobutler-org/quark/pkg/util/musicutil"
)

// scanQueue is a queue that knows the rescan kind but runs nothing, so a
// queued scan stays pending until the test settles it.
func scanQueue(database *db.DatabaseSqlc) *jobutil.Queue {
	queue := jobutil.NewQueue(jobutil.NewQueueParams{Database: database})
	queue.Register(jobutil.RegisterParams{Kind: musicutil.RescanKind, Handler: jobutil.Handler{}})
	return queue
}

// completeJob runs a pending job to completion the way an instance would.
func completeJob(t *testing.T, database *db.DatabaseSqlc, id int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := database.Queries.ClaimJob(ctx, db.ClaimJobParams{Owner: "test", ID: id, LaneLimit: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Queries.FinishJob(ctx, db.FinishJobParams{Status: string(jobutil.StatusCompleted), Progress: 1, ID: id, Owner: "test"}); err != nil {
		t.Fatal(err)
	}
}

// TestStatus_CountsReadableTracksAndReportsScans follows a library from no
// scan on record through a finished scan and a second one queued, and checks
// that a member's count covers only what they can read.
func TestStatus_CountsReadableTracksAndReportsScans(t *testing.T) {
	database := library(t)
	ctx := context.Background()
	queue := scanQueue(database)
	status := func(t *testing.T, params musicutil.StatusParams) musicutil.StatusResult {
		t.Helper()
		params.Ctx, params.DB = ctx, database.Db
		got, err := musicutil.Status(params)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	got := status(t, musicutil.StatusParams{Access: adminAccess(t), Jobs: queue})
	if got.TrackCount != 5 || got.LastScanAt != nil || got.ScanJobID != nil {
		t.Fatalf("before any scan = %+v, want 5 tracks and no scan", got)
	}
	if got := status(t, musicutil.StatusParams{Access: memberAccess(t, database, "shared")}); got.TrackCount != 3 {
		t.Errorf("member TrackCount = %d, want 3", got.TrackCount)
	}

	first, err := musicutil.Rescan(musicutil.RescanParams{Ctx: ctx, Jobs: queue})
	if err != nil {
		t.Fatal(err)
	}
	if first.Job.Kind != musicutil.RescanKind || first.Job.Status != jobutil.StatusPending {
		t.Fatalf("Rescan queued %+v, want a pending %s job", first.Job, musicutil.RescanKind)
	}
	got = status(t, musicutil.StatusParams{Access: adminAccess(t), Jobs: queue})
	if got.ScanJobID == nil || *got.ScanJobID != first.Job.ID || got.LastScanAt != nil {
		t.Fatalf("while scanning = %+v, want scan job %d and no last scan", got, first.Job.ID)
	}

	completeJob(t, database, first.Job.ID)
	second, err := musicutil.Rescan(musicutil.RescanParams{Ctx: ctx, Jobs: queue})
	if err != nil {
		t.Fatal(err)
	}
	got = status(t, musicutil.StatusParams{Access: adminAccess(t), Jobs: queue})
	if got.LastScanAt == nil {
		t.Error("LastScanAt is nil after a completed scan")
	}
	if got.ScanJobID == nil || *got.ScanJobID != second.Job.ID {
		t.Errorf("ScanJobID = %v, want %d", got.ScanJobID, second.Job.ID)
	}
}
