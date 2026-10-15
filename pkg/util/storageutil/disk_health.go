// cspell:words smartctl nvme mmcblk libexec

package storageutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// HealthHealthy means SMART was read and nothing in it calls for action.
	HealthHealthy = "healthy"
	// HealthWatch means the drive is readable and shows wear or a small fault.
	// Assigning it a role does not require acknowledgment.
	HealthWatch = "watch"
	// HealthReplaceSoon means the drive is failing or close to it. Assigning
	// it default storage or a backup role requires acknowledgment.
	HealthReplaceSoon = "replace-soon"
	// HealthUnknown means SMART could not be read. It is never a healthy drive.
	HealthUnknown = "unknown"

	// SmartPassed and SmartFailed are the overall SMART status, when one was read.
	SmartPassed = "passed"
	SmartFailed = "failed"

	// MediaSpinning is a rotating disk. MediaSolidState is an SSD or NVMe drive.
	MediaSpinning   = "spinning"
	MediaSolidState = "solid-state"

	// ReasonHealthy is the one sentence for a drive SMART says is fine.
	ReasonHealthy = "This drive reports that it is healthy."
	// ReasonWatchSectors is the one sentence when sectors have been reallocated
	// and the drive has not failed.
	ReasonWatchSectors = "This drive has reallocated sectors. Keep an eye on it."
	// ReasonWatchHours is the one sentence for high power-on hours. It says
	// heavily used and does not invent a manufacture year.
	ReasonWatchHours = "This drive is heavily used. Keep an eye on it."
	// ReasonWatchWear is the one sentence when a solid-state drive has used
	// much of its rated life and has not failed.
	ReasonWatchWear = "This drive has used much of its solid-state life. Keep an eye on it."
	// ReasonReplaceFailed is the one sentence when SMART says the drive is failing.
	ReasonReplaceFailed = "This drive is failing. Replace it soon."
	// ReasonReplaceSectors is the one sentence when sectors are waiting to be
	// reallocated or could not be corrected.
	ReasonReplaceSectors = "This drive has sectors waiting to be reallocated. Replace it soon."
	// ReasonReplaceWear is the one sentence when a solid-state drive has used
	// up its rated life.
	ReasonReplaceWear = "This drive has used up its solid-state life. Replace it soon."
	// ReasonToolMissing is the one sentence when smartctl, or diskutil on
	// macOS, is not installed. The device list still loads.
	ReasonToolMissing = "Drive health could not be read because the health tool is not installed."
	// ReasonUnreadable is the one sentence when the tool ran but SMART did not
	// come back, including a USB bridge that rejects SATA passthrough.
	ReasonUnreadable = "Drive health could not be read."

	// HelperPath is the root-owned helper `quark install` writes. It runs a
	// fixed smartctl read and checks the device path itself. The service
	// account may run it through sudo and cannot rewrite it.
	HelperPath = "/usr/local/libexec/quark/drive-health"

	// heavilyUsedHours is power-on time that marks a drive heavily used.
	// SMART reports hours powered on, not a manufacture date.
	heavilyUsedHours = 40000
	// ssdWatchPercent and ssdReplacePercent are NVMe percentage-used thresholds.
	ssdWatchPercent   = 50
	ssdReplacePercent = 90

	// healthCacheTTL is how long a verdict is reused. Health changes slowly,
	// and smartctl is a root command, so the device list does not read every time.
	healthCacheTTL = 15 * time.Minute
	// healthReadTimeout bounds one drive's smartctl or diskutil read.
	healthReadTimeout = 30 * time.Second
)

// wholeDiskName is the whole-disk names the root helper accepts. Partitions
// are stripped before the match. The helper's case statement accepts the same set.
var wholeDiskName = regexp.MustCompile(`^(sd[a-z]{1,2}|vd[a-z]|xvd[a-z]|nvme[0-9]{1,2}n[0-9]|mmcblk[0-9]{1,2})$`)

// errHealthToolMissing means the health tool is not on PATH, or the helper
// exited 127 because smartctl is not installed.
var errHealthToolMissing = errors.New("health tool is not installed")

// DiskHealth is one drive's health verdict and the SMART detail behind it.
// Verdict is healthy, watch, replace-soon, or unknown. Reason is one sentence.
// Counts and temperature are nil when that reading was not in the report.
type DiskHealth struct {
	Verdict            string `json:"verdict"`
	Reason             string `json:"reason"`
	SmartStatus        string `json:"smartStatus,omitempty"`
	ReallocatedSectors *int64 `json:"reallocatedSectors,omitempty"`
	PendingSectors     *int64 `json:"pendingSectors,omitempty"`
	PowerOnHours       *int64 `json:"powerOnHours,omitempty"`
	TemperatureCelsius *int   `json:"temperatureCelsius,omitempty"`
	Media              string `json:"media,omitempty"`
	LinkSpeed          string `json:"linkSpeed,omitempty"`
	// WorseThanDefault is set by the device list when this drive's verdict is
	// worse than the drive that currently holds default storage. The order is
	// healthy, then watch, then unknown, then replace soon. It is left false
	// when this drive is the default, or when no default is assigned.
	WorseThanDefault bool `json:"worseThanDefault,omitempty"`
}

// HealthSystem is how a health read talks to the host. A zero value uses the
// real commands. Tests set the funcs they care about.
type HealthSystem struct {
	// LookPath finds a command. Nil uses exec.LookPath.
	LookPath func(file string) (string, error)
	// Run runs name with args and returns its stdout. A non-zero exit still
	// returns whatever was written, because smartctl encodes SMART failure in
	// its status and prints JSON anyway. Nil uses exec.CommandContext.
	Run func(ctx context.Context, name string, args ...string) ([]byte, error)
	// HelperInstalled reports whether quark install has written HelperPath.
	// Nil stats that path on Linux and reports false on macOS.
	HelperInstalled func() bool
}

// UnknownDiskHealth is the verdict when a read was not attempted or produced
// nothing. It is Unknown, never Healthy.
func UnknownDiskHealth() DiskHealth {
	return DiskHealth{Verdict: HealthUnknown, Reason: ReasonUnreadable}
}

// HealthRank orders verdicts from best to worst: healthy, watch, unknown,
// replace soon. An unrecognized verdict ranks with unknown, so it is not healthy.
func HealthRank(verdict string) int {
	switch verdict {
	case HealthHealthy:
		return 0
	case HealthWatch:
		return 1
	case HealthReplaceSoon:
		return 3
	default:
		return 2
	}
}

// ReadDriveHealth reads one drive. It always returns a verdict: a missing tool
// or a failed read is Unknown and is not an error, so a device list still loads.
func ReadDriveHealth(ctx context.Context, devicePath string, system HealthSystem) DiskHealth {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, healthReadTimeout)
	defer cancel()
	if strings.TrimSpace(devicePath) == "" {
		return UnknownDiskHealth()
	}
	return readPlatform(ctx, devicePath, system.orDefault())
}

// CachedDriveHealth is ReadDriveHealth through a process-wide cache that
// refreshes every healthCacheTTL. The returned pointer is a copy; callers may
// set WorseThanDefault on it.
func CachedDriveHealth(ctx context.Context, devicePath string) *DiskHealth {
	health := defaultHealthCache.get(ctx, devicePath, func(ctx context.Context, path string) DiskHealth {
		return ReadDriveHealth(ctx, path, HealthSystem{})
	})
	return &health
}

func (s HealthSystem) orDefault() HealthSystem {
	if s.LookPath == nil {
		s.LookPath = exec.LookPath
	}
	if s.Run == nil {
		s.Run = defaultHealthRun
	}
	if s.HelperInstalled == nil {
		s.HelperInstalled = defaultHelperInstalled
	}
	return s
}

func defaultHealthRun(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 127 {
		return out, errHealthToolMissing
	}
	return out, err
}

// readSmartctl reads Linux SMART. It tries the root helper when quark install
// has written it, and otherwise smartctl itself. A USB bridge that hides SMART
// is tried again with SATA passthrough. Both failing is Unknown.
func readSmartctl(ctx context.Context, devicePath string, system HealthSystem) DiskHealth {
	disk := wholeDisk(devicePath)
	if disk == "" {
		return UnknownDiskHealth()
	}
	system = system.orDefault()
	if system.HelperInstalled() {
		out, err := system.Run(ctx, "sudo", "-n", HelperPath, disk)
		if errors.Is(err, errHealthToolMissing) {
			return toolMissingHealth()
		}
		if health, ok := classifySmartctlJSON(out); ok {
			return health
		}
		return UnknownDiskHealth()
	}
	if _, err := system.LookPath("smartctl"); err != nil {
		return toolMissingHealth()
	}
	return readSmartctlAttempts(ctx, disk, system.Run)
}

func readSmartctlAttempts(ctx context.Context, disk string, run func(context.Context, string, ...string) ([]byte, error)) DiskHealth {
	for _, deviceType := range []string{"", "sat"} {
		args := []string{"-j"}
		if deviceType != "" {
			args = append(args, "-d", deviceType)
		}
		args = append(args, disk)
		out, err := run(ctx, "smartctl", args...)
		if errors.Is(err, errHealthToolMissing) {
			return toolMissingHealth()
		}
		if health, ok := classifySmartctlJSON(out); ok {
			return health
		}
	}
	return UnknownDiskHealth()
}

func toolMissingHealth() DiskHealth {
	return DiskHealth{Verdict: HealthUnknown, Reason: ReasonToolMissing}
}

// wholeDisk is the whole-disk path a SMART read is allowed to name.
// /dev/sda1 and /dev/nvme0n1p2 become /dev/sda and /dev/nvme0n1. A path that
// is not a disk name returns empty, and the caller reports Unknown.
func wholeDisk(devicePath string) string {
	if !strings.HasPrefix(devicePath, "/dev/") || strings.Contains(devicePath, "..") {
		return ""
	}
	base := trimDiskPartition(path.Base(devicePath))
	if !wholeDiskName.MatchString(base) {
		return ""
	}
	return "/dev/" + base
}

func trimDiskPartition(base string) string {
	if strings.HasPrefix(base, "nvme") || strings.HasPrefix(base, "mmcblk") {
		if i := strings.LastIndex(base, "p"); i > 0 && allDigits(base[i+1:]) {
			return base[:i]
		}
		return base
	}
	i := 0
	for i < len(base) && base[i] >= 'a' && base[i] <= 'z' {
		i++
	}
	if i == 0 || (i < len(base) && !allDigits(base[i:])) {
		return ""
	}
	return base[:i]
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

type smartctlReport struct {
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature *struct {
		Current int `json:"current"`
	} `json:"temperature"`
	PowerOnTime *struct {
		Hours int64 `json:"hours"`
	} `json:"power_on_time"`
	RotationRate *int `json:"rotation_rate"`
	Interface    struct {
		Current struct {
			String string `json:"string"`
		} `json:"current"`
	} `json:"interface_speed"`
	ATA struct {
		Table []struct {
			ID  int `json:"id"`
			Raw struct {
				Value float64 `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	NVMe *struct {
		CriticalWarning int   `json:"critical_warning"`
		Temperature     int   `json:"temperature"`
		PercentageUsed  int   `json:"percentage_used"`
		PowerOnHours    int64 `json:"power_on_hours"`
		MediaErrors     int64 `json:"media_errors"`
	} `json:"nvme_smart_health_information_log"`
}

// classifySmartctlJSON parses smartctl --json. The bool is false when the
// output is not a SMART report, and the caller must not treat that as healthy.
func classifySmartctlJSON(output []byte) (DiskHealth, bool) {
	output = bytes.TrimSpace(output)
	if len(output) == 0 || output[0] != '{' {
		return DiskHealth{}, false
	}
	var report smartctlReport
	if err := json.Unmarshal(output, &report); err != nil {
		return DiskHealth{}, false
	}
	if report.SmartStatus == nil && report.NVMe == nil && len(report.ATA.Table) == 0 {
		return DiskHealth{}, false
	}

	health := DiskHealth{}
	var pending, offline int64
	var sawPending, sawOffline bool
	var attrHours *int64
	for _, attr := range report.ATA.Table {
		value := int64(attr.Raw.Value)
		switch attr.ID {
		case 5:
			health.ReallocatedSectors = int64Ptr(value)
		case 9:
			attrHours = int64Ptr(value)
		case 197:
			pending = value
			sawPending = true
			health.PendingSectors = int64Ptr(value)
		case 198:
			offline = value
			sawOffline = true
		}
	}
	switch {
	case report.PowerOnTime != nil:
		health.PowerOnHours = int64Ptr(report.PowerOnTime.Hours)
	case attrHours != nil:
		health.PowerOnHours = attrHours
	case report.NVMe != nil:
		health.PowerOnHours = int64Ptr(report.NVMe.PowerOnHours)
	}
	if report.Temperature != nil && report.Temperature.Current > 0 {
		health.TemperatureCelsius = intPtr(report.Temperature.Current)
	} else if report.NVMe != nil && report.NVMe.Temperature > 0 {
		health.TemperatureCelsius = intPtr(celsiusFromNVMe(report.NVMe.Temperature))
	}
	if report.Interface.Current.String != "" {
		health.LinkSpeed = report.Interface.Current.String
	}
	switch {
	case report.NVMe != nil:
		health.Media = MediaSolidState
	case report.RotationRate != nil && *report.RotationRate > 0:
		health.Media = MediaSpinning
	case report.RotationRate != nil:
		health.Media = MediaSolidState
	}
	if report.SmartStatus != nil {
		if report.SmartStatus.Passed {
			health.SmartStatus = SmartPassed
		} else {
			health.SmartStatus = SmartFailed
		}
	}

	var hours int64
	if health.PowerOnHours != nil {
		hours = *health.PowerOnHours
	}
	var reallocated int64
	if health.ReallocatedSectors != nil {
		reallocated = *health.ReallocatedSectors
	}
	percent := 0
	if report.NVMe != nil {
		percent = report.NVMe.PercentageUsed
	}
	smartFailed := report.SmartStatus != nil && !report.SmartStatus.Passed
	nvmeFailed := report.NVMe != nil && (report.NVMe.CriticalWarning != 0 || report.NVMe.MediaErrors > 0)
	pendingBad := (sawPending && pending > 0) || (sawOffline && offline > 0)

	switch {
	case smartFailed || nvmeFailed:
		health.Verdict = HealthReplaceSoon
		health.Reason = ReasonReplaceFailed
	case pendingBad:
		health.Verdict = HealthReplaceSoon
		health.Reason = ReasonReplaceSectors
	case percent >= ssdReplacePercent:
		health.Verdict = HealthReplaceSoon
		health.Reason = ReasonReplaceWear
	case reallocated > 0:
		health.Verdict = HealthWatch
		health.Reason = ReasonWatchSectors
	case percent >= ssdWatchPercent:
		health.Verdict = HealthWatch
		health.Reason = ReasonWatchWear
	case hours >= heavilyUsedHours:
		health.Verdict = HealthWatch
		health.Reason = ReasonWatchHours
	default:
		health.Verdict = HealthHealthy
		health.Reason = ReasonHealthy
	}
	return health, true
}

// classifyDiskutilInfo reads the SMART Status line `diskutil info` prints.
// Verified is healthy, Failing is replace soon, and anything else — including
// Not Supported and a missing line — is Unknown.
func classifyDiskutilInfo(info string) DiskHealth {
	const marker = "SMART Status:"
	i := strings.Index(info, marker)
	if i < 0 {
		return UnknownDiskHealth()
	}
	rest := info[i+len(marker):]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	switch strings.TrimSpace(rest) {
	case "Verified":
		return DiskHealth{Verdict: HealthHealthy, Reason: ReasonHealthy, SmartStatus: SmartPassed}
	case "Failing":
		return DiskHealth{Verdict: HealthReplaceSoon, Reason: ReasonReplaceFailed, SmartStatus: SmartFailed}
	default:
		return UnknownDiskHealth()
	}
}

// celsiusFromNVMe converts an NVMe temperature. Older smartctl JSON reported
// Kelvin (a value above 200); current JSON reports Celsius.
func celsiusFromNVMe(temp int) int {
	if temp > 200 {
		return temp - 273
	}
	return temp
}

func int64Ptr(v int64) *int64 { return &v }

func intPtr(v int) *int { return &v }

type healthCache struct {
	ttl time.Duration
	now func() time.Time
	mu  sync.Mutex
	// entries is keyed by the device path the caller passed, before whole-disk
	// normalization, so two partitions of one disk share nothing by accident
	// of a path we failed to fold.
	entries map[string]healthCacheEntry
}

type healthCacheEntry struct {
	health DiskHealth
	at     time.Time
}

var defaultHealthCache = &healthCache{ttl: healthCacheTTL, now: time.Now}

func (c *healthCache) get(ctx context.Context, devicePath string, read func(context.Context, string) DiskHealth) DiskHealth {
	if strings.TrimSpace(devicePath) == "" {
		return UnknownDiskHealth()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := c.now()
	c.mu.Lock()
	if entry, ok := c.entries[devicePath]; ok && now.Sub(entry.at) < c.ttl {
		health := entry.health
		c.mu.Unlock()
		return health
	}
	c.mu.Unlock()

	health := read(ctx, devicePath)
	if ctx.Err() != nil {
		return health
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]healthCacheEntry{}
	}
	c.entries[devicePath] = healthCacheEntry{health: health, at: c.now()}
	return health
}
