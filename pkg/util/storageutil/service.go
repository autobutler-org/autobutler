package storageutil

// StorageService wraps a Detector and exposes device-querying methods.
// Construct with NewStorageService(d) and inject via deputil.Dependencies.
type StorageService struct {
	detector Detector
}

// NewStorageService returns a StorageService backed by the given Detector.
func NewStorageService(d Detector) *StorageService {
	return &StorageService{detector: d}
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
func (s *StorageService) GetDeviceStatuses() ([]*DeviceStatus, error) {
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
		statuses = append(statuses, &DeviceStatus{
			Device:    device,
			IsEnabled: isEnabled,
			DataDir:   dataDir,
			CirrusDir: cirrusDir,
		})
	}
	return statuses, nil
}

// FindUsbDeviceBySerial finds a USB device by serial number.
func (s *StorageService) FindUsbDeviceBySerial(serial string) (UsbDevice, error) {
	return FindUsbDeviceBySerial(serial)
}
