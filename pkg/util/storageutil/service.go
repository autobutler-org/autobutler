package storageutil

import (
	"sync"
	"time"
)

// deviceStatusCache holds a short-lived cached copy of GetDeviceStatuses
// results to avoid redundant disk probes when the endpoint is called in
// rapid succession (see #1022).
type deviceStatusCache struct {
	mu       sync.Mutex
	result   []*DeviceStatus
	cachedAt time.Time
	ttl      time.Duration
}

func (c *deviceStatusCache) get() ([]*DeviceStatus, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.result == nil || time.Since(c.cachedAt) > c.ttl {
		return nil, false
	}
	return c.result, true
}

func (c *deviceStatusCache) set(result []*DeviceStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.result = result
	c.cachedAt = time.Now()
}

// invalidate clears the cache so the next GetDeviceStatuses call re-probes disk.
func (c *deviceStatusCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.result = nil
}

// diskProbeCache holds long-lived disk probe results keyed by device data
// directory. Probes are lightweight but still I/O-bound, so we cache them
// for probeResultTTL to avoid running a probe on every device-status refresh.
type diskProbeCache struct {
	mu      sync.Mutex
	results map[string]diskProbeCacheEntry
}

const probeResultTTL = 1 * time.Hour

type diskProbeCacheEntry struct {
	result   DiskProbeResult
	cachedAt time.Time
}

func (c *diskProbeCache) get(dir string) (DiskProbeResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.results == nil {
		return DiskProbeResult{}, false
	}
	entry, ok := c.results[dir]
	if !ok || time.Since(entry.cachedAt) > probeResultTTL {
		return DiskProbeResult{}, false
	}
	return entry.result, true
}

func (c *diskProbeCache) set(dir string, result DiskProbeResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.results == nil {
		c.results = make(map[string]diskProbeCacheEntry)
	}
	c.results[dir] = diskProbeCacheEntry{result: result, cachedAt: time.Now()}
}

// StorageService wraps a Detector and exposes device-querying methods.
// Construct with NewStorageService(d) and inject via deputil.Dependencies.
type StorageService struct {
	detector   Detector
	cache      deviceStatusCache
	probeCache diskProbeCache
}

// NewStorageService returns a StorageService backed by the given Detector.
func NewStorageService(d Detector) *StorageService {
	return &StorageService{
		detector: d,
		cache:    deviceStatusCache{ttl: 10 * time.Second},
	}
}

// GetManagedDevices returns all devices that have an autobutler data directory.
func (s *StorageService) GetManagedDevices() ([]ManagedDevice, error) {
	devices, err := s.detector.DetectDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	var managed []ManagedDevice
	for _, device := range devices {
		dataDir := GetDataDirForDevice(device.MountPoint)
		cirrusDir, err := GetCirrusDirForDevice(device.MountPoint)
		if err != nil {
			continue
		}
		managed = append(managed, ManagedDevice{
			Device:    device,
			DataDir:   dataDir,
			CirrusDir: cirrusDir,
		})
	}
	return managed, nil
}

// FindManagedDeviceBySerial finds a managed device by USB serial.
// An empty serial returns the first internal device.
func (s *StorageService) FindManagedDeviceBySerial(serial string) (*ManagedDevice, error) {
	managed, err := s.GetManagedDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}
	for _, d := range managed {
		if serial == "" && d.IsInternal {
			return &d, nil
		}
		if d.UsbInfo != nil && d.UsbInfo.GetSerial() == serial {
			return &d, nil
		}
	}
	return nil, nil
}

// GetDeviceStatuses returns all detected devices with their enable status.
// Results are cached for up to 10 seconds to avoid repeated disk probes when
// the endpoint is hit in rapid succession (#1022).
func (s *StorageService) GetDeviceStatuses() ([]*DeviceStatus, error) {
	if cached, ok := s.cache.get(); ok {
		return cached, nil
	}
	statuses, err := s.getDeviceStatusesFresh()
	if err != nil {
		return nil, err
	}
	s.cache.set(statuses)
	return statuses, nil
}

func (s *StorageService) getDeviceStatusesFresh() ([]*DeviceStatus, error) {
	devices, err := s.detector.DetectDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires device detection failure
	}

	managed, err := s.GetManagedDevices()
	if err != nil {
		return nil, err // coverage: ignore - requires filesystem errors reading managed devices
	}

	enabledMap := make(map[string]ManagedDevice)
	for _, md := range managed {
		enabledMap[md.MountPoint] = md
	}

	var statuses []*DeviceStatus
	for _, device := range devices {
		isEnabled := device.IsInternal
		dataDir := ""
		cirrusDir := ""
		if md, exists := enabledMap[device.MountPoint]; exists {
			isEnabled = true
			dataDir = md.DataDir
			cirrusDir = md.CirrusDir
		}

		var probeResult *DiskProbeResult
		if dataDir != "" {
			if cached, ok := s.probeCache.get(dataDir); ok {
				// Use cached probe result — probe ran recently.
				probeResult = &cached
			} else {
				// Kick off a background probe; this call returns the unknown result.
				// The next GetDeviceStatuses call (after the probe completes) will
				// return the measured values from cache.
				go func(dir string) {
					r := ProbeDisk(dir)
					s.probeCache.set(dir, r)
					s.cache.invalidate() // expire status cache so next request shows new data
				}(dataDir)
			}
		}

		statuses = append(statuses, &DeviceStatus{
			Device:    device,
			IsEnabled: isEnabled,
			DataDir:   dataDir,
			CirrusDir: cirrusDir,
			DiskProbe: probeResult,
		})
	}
	return statuses, nil
}

// InvalidateDeviceCache clears the device status cache so the next call
// to GetDeviceStatuses re-probes all devices from disk. Call this after
// any mount/unmount operation to prevent stale UI state.
func (s *StorageService) InvalidateDeviceCache() {
	s.cache.invalidate()
}

// FindUsbDeviceBySerial finds a USB device by serial number.
func (s *StorageService) FindUsbDeviceBySerial(serial string) (UsbDevice, error) {
	return FindUsbDeviceBySerial(serial)
}
