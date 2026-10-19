package musicutil

import "github.com/autobutler-org/quark/pkg/util/jobutil"

func status(params StatusParams) (StatusResult, error) {
	// A one-track page still counts every track the caller can read.
	tracks, err := listTracks(ListTracksParams{Ctx: params.Ctx, DB: params.DB, Access: params.Access, Limit: 1})
	if err != nil {
		return StatusResult{}, err
	}
	result := StatusResult{TrackCount: tracks.Total}
	if params.Jobs == nil {
		return result, nil
	}
	listed, err := params.Jobs.List(params.Ctx, jobutil.ListParams{Kinds: []string{RescanKind}})
	if err != nil {
		return StatusResult{}, err
	}
	// Newest first: the first active job is the scan in progress, and the
	// first completed one the last scan.
	for _, job := range listed.Jobs {
		switch {
		case result.ScanJobID == nil && (job.Status == jobutil.StatusPending || job.Status == jobutil.StatusRunning):
			result.ScanJobID = &job.ID
		case result.LastScanAt == nil && job.Status == jobutil.StatusCompleted:
			result.LastScanAt = job.FinishedAt
		}
	}
	return result, nil
}
