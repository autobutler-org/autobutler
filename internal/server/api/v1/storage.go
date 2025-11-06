package v1

import (
	"autobutler/pkg/storage"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupStorageRoutes configures storage-related API routes
func SetupStorageRoutes(apiGroup *gin.RouterGroup) {
	apiGroup.GET("/storage/devices", getStorageDevices)
	apiGroup.GET("/storage/summary", getStorageSummary)
}

// getStorageDevices returns all detected storage devices
// @Summary Get storage devices
// @Description Returns a list of all detected storage devices with usage information (READ-ONLY)
// @Tags storage
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/storage/devices [get]
func getStorageDevices(c *gin.Context) {
	detector := storage.NewDetector()

	// READ-ONLY: Detect devices using system commands
	devices, err := detector.DetectDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to detect storage devices",
			"details": err.Error(),
		})
		return
	}

	// Calculate summary
	summary := detector.CalculateSummary(devices)

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"summary": summary,
	})
}

// getStorageSummary returns storage summary only
// @Summary Get storage summary
// @Description Returns overall storage summary across all devices (READ-ONLY)
// @Tags storage
// @Produce json
// @Success 200 {object} storage.Summary
// @Router /api/v1/storage/summary [get]
func getStorageSummary(c *gin.Context) {
	detector := storage.NewDetector()

	// READ-ONLY: Detect devices
	devices, err := detector.DetectDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to detect storage devices",
			"details": err.Error(),
		})
		return
	}

	// Calculate summary
	summary := detector.CalculateSummary(devices)

	c.JSON(http.StatusOK, summary)
}
