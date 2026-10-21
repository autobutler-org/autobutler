package photoutil

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/autobutler-org/quark/pkg/util/eventbus"
)

// Reasons a photo could not be indexed, reported in [IndexFailure.Reason].
const (
	// IndexFailureUndecodable is a photo no decoder here can read, so it has
	// a content hash but no perceptual one.
	IndexFailureUndecodable = "undecodable"
	// IndexFailureMissingConverter is a camera RAW photo on a device with no
	// RAW converter (dcraw or exiftool) installed.
	IndexFailureMissingConverter = "missing_converter"
	// IndexFailureTooLarge is an image whose header claims more than
	// [MaxDecodePixels] pixels.
	IndexFailureTooLarge = "too_large"
	// IndexFailureUnreadable is a photo that could not be opened, read, or
	// stored at all.
	IndexFailureUnreadable = "unreadable"
)

// MaxIndexFailures is the most failed photos [IndexStatusResult.Failures]
// lists. [IndexStatusResult.Failed] still counts every one.
const MaxIndexFailures = 500

// indexProgressInterval is the least time between two progress events.
const indexProgressInterval = 2 * time.Second

// IndexControl is the photo library index pass's in-memory state: whether it
// is running or paused, what it has counted, and what failed (#3556). The
// server keeps one on its dependency graph; [BackfillHashes] reports into it
// and waits on it between photos, and the status, pause and resume routes
// read and change it. Nothing here is stored: a restart forgets a pause, and
// the next pass skips what is already indexed.
type IndexControl struct {
	mu sync.Mutex
	// resumed is closed when a pause ends; nil while not paused.
	resumed      chan struct{}
	running      bool
	status       IndexStatusResult
	lastProgress time.Time
}

// NewIndexControl returns an index control that is idle and not paused.
func NewIndexControl() *IndexControl {
	return &IndexControl{}
}

// IndexFailure is one photo the index pass could not fully read.
type IndexFailure struct {
	Serial  string `json:"serial"`
	RelPath string `json:"relPath"`
	// Reason is one of the IndexFailure* constants.
	Reason string `json:"reason"`
}

// IndexStatusParams names the index control to report.
type IndexStatusParams struct {
	// Control is the server's index control. Nil reports an idle, empty index.
	Control *IndexControl
}

// IndexStatusResult is the photo library index's state on this device.
type IndexStatusResult struct {
	// Total is the photos in the library when the pass last listed it.
	Total int `json:"total"`
	// Hashed is the photos with both a perceptual and a content hash, which
	// duplicate detection needs.
	Hashed int `json:"hashed"`
	// DatesRead is the photos whose capture date has been read.
	DatesRead int `json:"datesRead"`
	// Failed is the photos the current or last pass could not fully read.
	Failed int `json:"failed"`
	// Failures lists the first [MaxIndexFailures] of them.
	Failures []IndexFailure `json:"failures"`
	// LastRunAt is when a pass last finished, nil if none has since start.
	LastRunAt *time.Time `json:"lastRunAt"`
	// Running is whether a pass is under way, paused or not.
	Running bool `json:"running"`
	// Paused is whether the pass is held, or will hold when it starts.
	Paused bool `json:"paused"`
}

// PauseIndexParams names the index control to pause.
type PauseIndexParams struct {
	Control *IndexControl
	// EventBus, when set, hears photo_index_changed.
	EventBus *eventbus.Bus
}

// ResumeIndexParams names the index control to resume.
type ResumeIndexParams struct {
	Control *IndexControl
	// EventBus, when set, hears photo_index_changed.
	EventBus *eventbus.Bus
}

// ErrNoIndexControl reports a pause or resume with no index control to act on.
var ErrNoIndexControl = errors.New("photo index control is not available")

// IndexStatus reports the photo library index: what it holds, what failed,
// and whether its pass is running or paused.
func IndexStatus(params IndexStatusParams) (IndexStatusResult, error) {
	if params.Control == nil {
		return IndexStatusResult{Failures: []IndexFailure{}}, nil
	}
	return params.Control.snapshot(), nil
}

// PauseIndex holds the index pass before its next photo, or before its first
// if it has not started. The photo being read finishes first. Pausing a
// paused pass changes nothing.
func PauseIndex(params PauseIndexParams) (IndexStatusResult, error) {
	c := params.Control
	if c == nil {
		return IndexStatusResult{}, ErrNoIndexControl
	}
	c.mu.Lock()
	if c.resumed == nil {
		c.resumed = make(chan struct{})
	}
	c.mu.Unlock()
	publishIndexChanged(params.EventBus)
	return c.snapshot(), nil
}

// ResumeIndex lets a paused index pass carry on from the photo it stopped
// before. Resuming a pass that is not paused changes nothing.
func ResumeIndex(params ResumeIndexParams) (IndexStatusResult, error) {
	c := params.Control
	if c == nil {
		return IndexStatusResult{}, ErrNoIndexControl
	}
	c.mu.Lock()
	if c.resumed != nil {
		close(c.resumed)
		c.resumed = nil
	}
	c.mu.Unlock()
	publishIndexChanged(params.EventBus)
	return c.snapshot(), nil
}

// publishIndexChanged tells open clients to fetch the index status again.
func publishIndexChanged(bus *eventbus.Bus) {
	if bus != nil {
		bus.Publish(eventbus.Event{Kind: eventbus.EventPhotoIndexChanged})
	}
}

func (c *IndexControl) snapshot() IndexStatusResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.status
	s.Failures = append([]IndexFailure{}, c.status.Failures...)
	s.Running = c.running
	s.Paused = c.resumed != nil
	return s
}

// begin starts a pass over a library of total photos, hashed and dated of
// which are already indexed, forgetting the last pass's failures.
func (c *IndexControl) begin(total, hashed, dated int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = true
	c.lastProgress = time.Now()
	c.status.Total, c.status.Hashed, c.status.DatesRead = total, hashed, dated
	c.status.Failed, c.status.Failures = 0, nil
}

// indexed counts a photo newly hashed and newly dated, and reports whether
// enough time has passed since the last progress event for another.
func (c *IndexControl) indexed(hashed, dated bool) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if hashed {
		c.status.Hashed++
	}
	if dated {
		c.status.DatesRead++
	}
	return c.progressDue()
}

// failed counts a photo the pass could not fully read, and reports whether a
// progress event is due.
func (c *IndexControl) failed(f IndexFailure) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.Failed++
	if len(c.status.Failures) < MaxIndexFailures {
		c.status.Failures = append(c.status.Failures, f)
	}
	return c.progressDue()
}

// progressDue reports whether a progress event is due, and if so starts the
// next interval. c.mu is held.
func (c *IndexControl) progressDue() bool {
	if time.Since(c.lastProgress) < indexProgressInterval {
		return false
	}
	c.lastProgress = time.Now()
	return true
}

// finish ends the pass. A pass that ran to the end records its finish time;
// one stopped early by its context does not.
func (c *IndexControl) finish(completed bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	if completed {
		now := time.Now()
		c.status.LastRunAt = &now
	}
}

// waitWhilePaused returns once the pass is not paused, or with ctx's error
// when ctx ends first. It holds no IO slot while it waits.
func (c *IndexControl) waitWhilePaused(ctx context.Context) error {
	if c == nil {
		return ctx.Err()
	}
	for {
		c.mu.Lock()
		resumed := c.resumed
		c.mu.Unlock()
		if resumed == nil {
			return ctx.Err()
		}
		select {
		case <-resumed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
