package mdnsutil_test

import (
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/mdnsutil"
)

func TestIsAdvertising_FalseInitially(t *testing.T) {
	if mdnsutil.IsAdvertising() {
		t.Error("expected IsAdvertising() == false before Advertise()")
	}
}

func TestStop_NoopWhenNotStarted(t *testing.T) {
	// Should not panic or block.
	mdnsutil.Stop()
}

func TestAdvertise_And_Stop(t *testing.T) {
	if err := mdnsutil.Advertise(mdnsutil.AdvertiseOptions{
		Port:         19999,
		InstanceName: "Test AutoButler",
		TXT:          []string{"version=test"},
	}); err != nil {
		// mDNS registration can legitimately fail in CI environments
		// (no suitable network interface, multicast disabled).
		// Treat as a skip rather than a hard failure.
		t.Skipf("mDNS not available in this environment: %v", err)
	}

	if !mdnsutil.IsAdvertising() {
		t.Error("expected IsAdvertising() == true after Advertise()")
	}

	// Idempotent: a second call should be a no-op.
	if err := mdnsutil.Advertise(mdnsutil.AdvertiseOptions{Port: 19999}); err != nil {
		t.Errorf("second Advertise call should be idempotent but returned error: %v", err)
	}

	mdnsutil.Stop()

	if mdnsutil.IsAdvertising() {
		t.Error("expected IsAdvertising() == false after Stop()")
	}

	// Stop again — should be idempotent.
	mdnsutil.Stop()
}
