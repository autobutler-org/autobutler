// cspell:words smartctl nvme mmcblk

package storageutil

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestClassifySmartctl_Healthy(t *testing.T) {
	health, ok := classifySmartctlJSON([]byte(`{
		"smart_status": {"passed": true},
		"temperature": {"current": 32},
		"power_on_time": {"hours": 100},
		"rotation_rate": 0,
		"interface_speed": {"current": {"string": "6.0 Gb/s"}},
		"ata_smart_attributes": {"table": [
			{"id": 5, "raw": {"value": 0}},
			{"id": 197, "raw": {"value": 0}},
			{"id": 198, "raw": {"value": 0}},
			{"id": 9, "raw": {"value": 100}}
		]}
	}`))
	if !ok {
		t.Fatal("expected a SMART report")
	}
	if health.Verdict != HealthHealthy || health.Reason != ReasonHealthy {
		t.Fatalf("health = %+v, want healthy", health)
	}
	if health.SmartStatus != SmartPassed || health.Media != MediaSolidState || health.LinkSpeed != "6.0 Gb/s" {
		t.Fatalf("detail = %+v", health)
	}
	if health.PowerOnHours == nil || *health.PowerOnHours != 100 || health.TemperatureCelsius == nil || *health.TemperatureCelsius != 32 {
		t.Fatalf("readings = %+v", health)
	}
}

func TestClassifySmartctl_Failing(t *testing.T) {
	health, ok := classifySmartctlJSON([]byte(`{
		"smart_status": {"passed": false},
		"temperature": {"current": 41},
		"power_on_time": {"hours": 1200},
		"rotation_rate": 7200,
		"ata_smart_attributes": {"table": [
			{"id": 5, "raw": {"value": 8}},
			{"id": 197, "raw": {"value": 3}}
		]}
	}`))
	if !ok {
		t.Fatal("expected a SMART report")
	}
	if health.Verdict != HealthReplaceSoon || health.Reason != ReasonReplaceFailed {
		t.Fatalf("health = %+v, want replace soon because SMART failed", health)
	}
	if health.SmartStatus != SmartFailed || health.Media != MediaSpinning {
		t.Fatalf("detail = %+v", health)
	}
	if health.ReallocatedSectors == nil || *health.ReallocatedSectors != 8 || health.PendingSectors == nil || *health.PendingSectors != 3 {
		t.Fatalf("sectors = %+v", health)
	}
}

func TestClassifySmartctl_PendingSectorsReplaceSoon(t *testing.T) {
	health, ok := classifySmartctlJSON([]byte(`{
		"smart_status": {"passed": true},
		"ata_smart_attributes": {"table": [
			{"id": 5, "raw": {"value": 0}},
			{"id": 197, "raw": {"value": 2}}
		]}
	}`))
	if !ok || health.Verdict != HealthReplaceSoon || health.Reason != ReasonReplaceSectors {
		t.Fatalf("health = %+v, ok=%v, want replace soon for pending sectors", health, ok)
	}
}

func TestClassifySmartctl_HeavilyUsedHours(t *testing.T) {
	health, ok := classifySmartctlJSON([]byte(`{
		"smart_status": {"passed": true},
		"power_on_time": {"hours": 60000},
		"rotation_rate": 7200,
		"ata_smart_attributes": {"table": [
			{"id": 5, "raw": {"value": 0}},
			{"id": 197, "raw": {"value": 0}}
		]}
	}`))
	if !ok {
		t.Fatal("expected a SMART report")
	}
	if health.Verdict != HealthWatch || health.Reason != ReasonWatchHours {
		t.Fatalf("health = %+v, want watch for power-on hours", health)
	}
	if !strings.Contains(health.Reason, "heavily used") {
		t.Fatalf("reason = %q, want heavily used", health.Reason)
	}
	if regexp.MustCompile(`\b(19|20)\d{2}\b`).MatchString(health.Reason) {
		t.Fatalf("reason %q invents a manufacture year", health.Reason)
	}
}

func TestClassifySmartctl_ReallocatedIsWatch(t *testing.T) {
	health, ok := classifySmartctlJSON([]byte(`{
		"smart_status": {"passed": true},
		"power_on_time": {"hours": 10},
		"ata_smart_attributes": {"table": [{"id": 5, "raw": {"value": 4}}]}
	}`))
	if !ok || health.Verdict != HealthWatch || health.Reason != ReasonWatchSectors {
		t.Fatalf("health = %+v, ok=%v, want watch", health, ok)
	}
}

func TestClassifySmartctl_UnreadableIsNotHealthy(t *testing.T) {
	for _, output := range []string{"", "not json", `{"smartctl":{"exit_status":2}}`, `{"smart_status":`} {
		health, ok := classifySmartctlJSON([]byte(output))
		if ok {
			t.Fatalf("output %q classified as %+v", output, health)
		}
		if health.Verdict == HealthHealthy {
			t.Fatalf("output %q reported healthy", output)
		}
	}
	got := readSmartctlAttempts(context.Background(), "/dev/sda", func(context.Context, string, ...string) ([]byte, error) {
		return []byte("smartctl: open failed"), errors.New("exit status 2")
	})
	if got.Verdict != HealthUnknown || got.Reason != ReasonUnreadable || got.Verdict == HealthHealthy {
		t.Fatalf("failed read = %+v, want unknown", got)
	}
}

func TestReadSmartctl_MissingTool(t *testing.T) {
	ran := false
	system := HealthSystem{
		LookPath:        func(string) (string, error) { return "", exec.ErrNotFound },
		HelperInstalled: func() bool { return false },
		Run: func(context.Context, string, ...string) ([]byte, error) {
			ran = true
			return nil, errors.New("should not run")
		},
	}
	got := readSmartctl(context.Background(), "/dev/sdb1", system)
	if got.Verdict != HealthUnknown || got.Reason != ReasonToolMissing {
		t.Fatalf("missing tool = %+v", got)
	}
	if !strings.Contains(got.Reason, "health tool is not installed") {
		t.Fatalf("reason = %q", got.Reason)
	}
	if ran {
		t.Fatal("smartctl ran even though it is not installed")
	}
}

func TestReadSmartctl_SATAPassthrough(t *testing.T) {
	var calls [][]string
	system := HealthSystem{
		LookPath:        func(string) (string, error) { return "/usr/sbin/smartctl", nil },
		HelperInstalled: func() bool { return false },
		Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			for _, arg := range args {
				if arg == "sat" {
					return []byte(`{"smart_status":{"passed":true},"rotation_rate":7200}`), nil
				}
			}
			return []byte(`{"smartctl":{"exit_status":2}}`), errors.New("exit status 2")
		},
	}
	got := readSmartctl(context.Background(), "/dev/sdc1", system)
	if got.Verdict != HealthHealthy {
		t.Fatalf("passthrough = %+v, want the second read's healthy report", got)
	}
	if len(calls) != 2 || !strings.Contains(strings.Join(calls[1], " "), "-d sat /dev/sdc") {
		t.Fatalf("calls = %q, want a plain read then SATA passthrough of /dev/sdc", calls)
	}
}

func TestReadSmartctl_Helper(t *testing.T) {
	var calls [][]string
	system := HealthSystem{
		HelperInstalled: func() bool { return true },
		LookPath:        func(string) (string, error) { return "", exec.ErrNotFound },
		Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			return []byte(`{"smart_status":{"passed":false}}`), nil
		},
	}
	got := readSmartctl(context.Background(), "/dev/nvme0n1p2", system)
	if got.Verdict != HealthReplaceSoon {
		t.Fatalf("helper read = %+v", got)
	}
	if len(calls) != 1 || calls[0][0] != "sudo" || calls[0][len(calls[0])-1] != "/dev/nvme0n1" {
		t.Fatalf("calls = %q, want one sudo of the helper for the whole disk", calls)
	}
}

func TestReadSmartctl_HelperMissingTool(t *testing.T) {
	system := HealthSystem{
		HelperInstalled: func() bool { return true },
		Run: func(context.Context, string, ...string) ([]byte, error) {
			return nil, errHealthToolMissing
		},
	}
	got := readSmartctl(context.Background(), "/dev/sda", system)
	if got.Verdict != HealthUnknown || got.Reason != ReasonToolMissing {
		t.Fatalf("helper without smartctl = %+v", got)
	}
}

func TestClassifyDiskutilInfo(t *testing.T) {
	verified := classifyDiskutilInfo("   SMART Status:              Verified\n")
	if verified.Verdict != HealthHealthy || verified.SmartStatus != SmartPassed {
		t.Fatalf("verified = %+v", verified)
	}
	failing := classifyDiskutilInfo("   SMART Status:              Failing\n")
	if failing.Verdict != HealthReplaceSoon || failing.Reason != ReasonReplaceFailed {
		t.Fatalf("failing = %+v", failing)
	}
	for _, info := range []string{"   SMART Status:              Not Supported\n", "Device Location: External\n"} {
		got := classifyDiskutilInfo(info)
		if got.Verdict != HealthUnknown || got.Verdict == HealthHealthy {
			t.Fatalf("info %q = %+v, want unknown", info, got)
		}
	}
}

func TestWholeDisk(t *testing.T) {
	cases := map[string]string{
		"/dev/sda1":      "/dev/sda",
		"/dev/sda":       "/dev/sda",
		"/dev/nvme0n1p2": "/dev/nvme0n1",
		"/dev/mmcblk0p1": "/dev/mmcblk0",
		"/dev/vda2":      "/dev/vda",
		"/dev/disk3s5":   "",
		"/tmp/sda":       "",
		"":               "",
	}
	for path, want := range cases {
		if got := wholeDisk(path); got != want {
			t.Errorf("wholeDisk(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestHealthCacheRefreshesSlowly(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	cache := &healthCache{ttl: healthCacheTTL, now: func() time.Time { return now }}
	reads := 0
	read := func(context.Context, string) DiskHealth {
		reads++
		return DiskHealth{Verdict: HealthHealthy, Reason: ReasonHealthy}
	}
	first := cache.get(context.Background(), "/dev/sda", read)
	second := cache.get(context.Background(), "/dev/sda", read)
	if reads != 1 || first.Verdict != HealthHealthy || second.Verdict != HealthHealthy {
		t.Fatalf("reads = %d, first %+v, second %+v, want one cached read", reads, first, second)
	}
	now = now.Add(healthCacheTTL + time.Second)
	cache.get(context.Background(), "/dev/sda", read)
	if reads != 2 {
		t.Fatalf("reads = %d, want a refresh after the TTL", reads)
	}
}

func TestHealthRank_UnknownIsNotHealthy(t *testing.T) {
	if HealthRank(HealthUnknown) <= HealthRank(HealthHealthy) {
		t.Fatal("unknown ranked as healthy or better")
	}
	if HealthRank("bogus") != HealthRank(HealthUnknown) {
		t.Fatal("unrecognized verdict should rank with unknown")
	}
}

func TestNVMeWearAndKelvin(t *testing.T) {
	worn, ok := classifySmartctlJSON([]byte(`{
		"nvme_smart_health_information_log": {
			"critical_warning": 0,
			"temperature": 308,
			"percentage_used": 91,
			"power_on_hours": 1000,
			"media_errors": 0
		}
	}`))
	if !ok || worn.Verdict != HealthReplaceSoon || worn.Reason != ReasonReplaceWear {
		t.Fatalf("worn = %+v, ok=%v", worn, ok)
	}
	if worn.TemperatureCelsius == nil || *worn.TemperatureCelsius != 35 {
		t.Fatalf("temperature = %v, want 35 C from Kelvin", worn.TemperatureCelsius)
	}
	if worn.Media != MediaSolidState {
		t.Fatalf("media = %q", worn.Media)
	}
}
