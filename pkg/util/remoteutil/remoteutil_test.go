package remoteutil_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/remoteutil"
)

func TestIsRunning_FalseInitially(t *testing.T) {
	if remoteutil.IsRunning() {
		t.Error("expected IsRunning() == false before Start()")
	}
}

func TestIsTLSActive_FalseInitially(t *testing.T) {
	if remoteutil.IsTLSActive() {
		t.Error("expected IsTLSActive() == false before StartProxy()")
	}
}

func TestRemoteURL_EmptyWhenNotRunning(t *testing.T) {
	if url := remoteutil.RemoteURL(); url != "" {
		t.Errorf("expected empty RemoteURL when not running, got %q", url)
	}
}

func TestHasPersistedState_ReturnsBool(t *testing.T) {
	// Just verify it doesn't panic. The actual return value depends on
	// whether tsnet state has been written to disk on this machine.
	_ = remoteutil.HasPersistedState()
}
