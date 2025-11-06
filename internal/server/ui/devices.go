package ui

import (
	"autobutler/internal/server/ui/components/device_card"
	"autobutler/internal/server/ui/types"
	"autobutler/internal/server/ui/views"
	"autobutler/pkg/storage"

	"github.com/gin-gonic/gin"
)

func SetupDevicesRoutes(router *gin.Engine) {
	setupDevicesView(router)
	setupDevicesComponents(router)
}

func setupDevicesView(router *gin.Engine) {
	uiRoute(router, "/devices", func(c *gin.Context) {
		// Detect storage devices using READ-ONLY operations
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()

		// Convert storage.Device to device_card.Device
		var cardDevices []device_card.Device

		if err == nil {
			for _, d := range devices {
				cardDevice := device_card.Device{
					Name:        d.Name,
					Type:        d.Type,
					DevicePath:  d.DevicePath,
					CapacityGB:  storage.BytesToGB(d.TotalBytes),
					UsedGB:      storage.BytesToGB(d.UsedBytes),
					PercentUsed: d.PercentUsed,
					Mount:       d.MountPoint,
					Health:      d.Health,
					Status:      d.Status,
				}

				// Convert categories
				if sysBytes, ok := d.Categories["system"]; ok {
					cardDevice.SystemGB = storage.BytesToGB(sysBytes)
				}
				if docBytes, ok := d.Categories["documents"]; ok {
					cardDevice.DocumentsGB = storage.BytesToGB(docBytes)
				}
				if mediaBytes, ok := d.Categories["media"]; ok {
					cardDevice.MediaGB = storage.BytesToGB(mediaBytes)
				}
				if backupBytes, ok := d.Categories["backups"]; ok {
					cardDevice.BackupsGB = storage.BytesToGB(backupBytes)
				}
				if freeBytes, ok := d.Categories["free"]; ok {
					cardDevice.FreeGB = storage.BytesToGB(freeBytes)
				}

				cardDevices = append(cardDevices, cardDevice)
			}
		}

		// Calculate summary
		summary := detector.CalculateSummary(devices)

		if err := views.Devices(types.NewPageState(), cardDevices, summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}

func setupDevicesComponents(router *gin.Engine) {
	// Component endpoint for HTMX to refresh just the devices list
	uiRoute(router, "/components/devices/list", func(c *gin.Context) {
		// Re-detect storage devices (READ-ONLY)
		detector := storage.NewDetector()
		devices, err := detector.DetectDevices()

		// Convert storage.Device to device_card.Device
		var cardDevices []device_card.Device

		if err == nil {
			for _, d := range devices {
				cardDevice := device_card.Device{
					Name:        d.Name,
					Type:        d.Type,
					DevicePath:  d.DevicePath,
					CapacityGB:  storage.BytesToGB(d.TotalBytes),
					UsedGB:      storage.BytesToGB(d.UsedBytes),
					PercentUsed: d.PercentUsed,
					Mount:       d.MountPoint,
					Health:      d.Health,
					Status:      d.Status,
				}

				// Convert categories
				if sysBytes, ok := d.Categories["system"]; ok {
					cardDevice.SystemGB = storage.BytesToGB(sysBytes)
				}
				if docBytes, ok := d.Categories["documents"]; ok {
					cardDevice.DocumentsGB = storage.BytesToGB(docBytes)
				}
				if mediaBytes, ok := d.Categories["media"]; ok {
					cardDevice.MediaGB = storage.BytesToGB(mediaBytes)
				}
				if backupBytes, ok := d.Categories["backups"]; ok {
					cardDevice.BackupsGB = storage.BytesToGB(backupBytes)
				}
				if freeBytes, ok := d.Categories["free"]; ok {
					cardDevice.FreeGB = storage.BytesToGB(freeBytes)
				}

				cardDevices = append(cardDevices, cardDevice)
			}
		}

		// Calculate summary
		summary := detector.CalculateSummary(devices)

		// Render just the devices content component
		if err := views.DevicesContent(cardDevices, summary).Render(c.Request.Context(), c.Writer); err != nil {
			c.Status(400)
			return
		}
		c.Status(200)
	})
}
