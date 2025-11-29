package v1

import (
	"autobutler/pkg/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupStorageRoutes configures storage-related API routes
func SetupStorageRoutes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/storage/devices/status", getDeviceStatuses)
	apiGroup.POST("/storage/managed", initializeManagedDevice)
}

func initializeManagedDevice(c *gin.Context) {
	mountPoint := c.PostForm("mountPoint")

	if mountPoint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Mount point is required",
		})
		return
	}

	err := storage.InitializeDeviceDataDir(mountPoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to initialize device",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Device initialized successfully",
		"mountPoint": mountPoint,
	})
}

// DeviceStatus represents the status of a storage device
type DeviceStatus struct {
	storage.Device
	IsEnabled bool   `json:"is_enabled"`
	DataDir   string `json:"data_dir,omitempty"`
	FilesDir  string `json:"files_dir,omitempty"`
}

// getDeviceStatuses returns all detected devices with their enable status
// @Summary Get device statuses
// @Description Returns all storage devices with information about whether they are enabled for Autobutler
// @Tags storage
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/v1/storage/devices/status [get]
func getDeviceStatuses(c *gin.Context) {
	// Detect all storage devices
	detector := storage.NewDetector()
	devices, err := detector.DetectDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to detect storage devices",
			"details": err.Error(),
		})
		return
	}

	// Get managed devices to check which are enabled
	managedDevices, err := storage.GetManagedDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get managed devices",
			"details": err.Error(),
		})
		return
	}

	// Create a map of enabled devices
	enabledMap := make(map[string]storage.ManagedDevice)
	for _, md := range managedDevices {
		enabledMap[md.MountPoint] = md
	}

	// Build status list
	var statuses []DeviceStatus
	for _, device := range devices {
		status := DeviceStatus{
			Device:    device,
			IsEnabled: false,
		}

		if md, exists := enabledMap[device.MountPoint]; exists {
			status.IsEnabled = true
			status.DataDir = md.DataDir
			status.FilesDir = md.FilesDir
		}

		statuses = append(statuses, status)
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": statuses,
		"count":   len(statuses),
	})
}
