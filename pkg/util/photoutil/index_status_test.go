package photoutil_test

import (
	"testing"

	"github.com/autobutler-org/quark/pkg/util/photoutil"
)

// TestPauseResumeIndex_Idempotent: pausing twice or resuming a pass that is
// not paused changes nothing, so a repeated tap or a retried request is safe.
func TestPauseResumeIndex_Idempotent(t *testing.T) {
	control := photoutil.NewIndexControl()
	for range 2 {
		status, err := photoutil.PauseIndex(photoutil.PauseIndexParams{Control: control})
		if err != nil {
			t.Fatal(err)
		}
		if !status.Paused || status.Running {
			t.Errorf("after pause: %+v, want paused and not running", status)
		}
	}
	for range 2 {
		status, err := photoutil.ResumeIndex(photoutil.ResumeIndexParams{Control: control})
		if err != nil {
			t.Fatal(err)
		}
		if status.Paused {
			t.Errorf("after resume: %+v, want not paused", status)
		}
	}
}

// TestIndexStatus_NoControl: a graph without an index control reports an
// idle, empty index and refuses to pause what it cannot control.
func TestIndexStatus_NoControl(t *testing.T) {
	status, err := photoutil.IndexStatus(photoutil.IndexStatusParams{})
	if err != nil {
		t.Fatal(err)
	}
	if status.Running || status.Paused || status.Total != 0 || status.LastRunAt != nil || status.Failures == nil {
		t.Errorf("status = %+v, want idle, empty, and an empty failure list", status)
	}
	if _, err := photoutil.PauseIndex(photoutil.PauseIndexParams{}); err == nil {
		t.Error("pause without a control succeeded, want an error")
	}
	if _, err := photoutil.ResumeIndex(photoutil.ResumeIndexParams{}); err == nil {
		t.Error("resume without a control succeeded, want an error")
	}
}
