package v1_files

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const DefaultDeviceSerial = ""

// FileNodeJSON is a JSON-serializable representation of a file node
type FileNodeJSON struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	IsDir        bool   `json:"isDir"`
	DeviceName   string `json:"deviceName"`
	DevicePath   string `json:"devicePath"`
	FullPath     string `json:"fullPath"`
	DeviceSerial string `json:"deviceSerial"`
}

// listFiles godoc
// @Summary Lists files
// @Schemes http https
// @Description merges files across all managed devices for the given filePath. If deviceSerial is empty, list files across all devices. Otherwise, only for the specified device
// @Tags cirrus
// @Produce json
// @Success 200 {array} FileNodeJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Param rootDir query string false "File dir to list"
// @Param serial query string false "Device serial number to filter by"
// @Router /cirrus [get]
func listFiles(c *gin.Context) *serverutil.Response {
	rootDir := c.Query("rootDir")
	fmt.Printf("List %s\n", rootDir)
	serials := c.QueryArray("serial")

	devices, err := storageutil.GetManagedDevices()
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	var selectedDevices []storageutil.ManagedDevice
	if len(serials) == 0 {
		selectedDevices = devices
	} else {
		// Build a set for quick lookup
		serialSet := make(map[string]bool)
		for _, s := range serials {
			serialSet[s] = true
		}
		for _, d := range devices {
			if d.UsbInfo != nil {
				if serialSet[d.UsbInfo.GetSerial()] {
					selectedDevices = append(selectedDevices, d)
				}
			} else {
				// Internal device (no UsbInfo) represented by empty serial string
				if serialSet[DefaultDeviceSerial] {
					selectedDevices = append(selectedDevices, d)
				}
			}
		}
		if len(selectedDevices) == 0 {
			return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData([]FileNodeJSON{})
		}
	}
	var allFiles []*storageutil.DeviceFileInfo
	for _, device := range selectedDevices {
		cirrusDir := device.CirrusDir
		fullPathDir := filepath.Join(cirrusDir, rootDir)
		deviceSerial := DefaultDeviceSerial
		if device.UsbInfo != nil {
			deviceSerial = device.UsbInfo.GetSerial()
		}
		files, err := storageutil.StatFilesInDir(fullPathDir, device.Name, device.DataDir, deviceSerial)
		if err != nil {
			continue
		}
		allFiles = append(allFiles, files...)
	}
	// Only deduplicate folders (isDir), show all files across all devices
	seenFolders := make(map[string]bool)
	filteredFiles := make([]*storageutil.DeviceFileInfo, 0, len(allFiles))
	for _, file := range allFiles {
		name := file.FileInfo.Name()
		if file.IsDir() {
			if seenFolders[name] {
				continue
			}
			seenFolders[name] = true
		}
		filteredFiles = append(filteredFiles, file)
	}
	allFiles = filteredFiles
	// Convert to JSON-serializable format
	jsonData := make([]FileNodeJSON, len(allFiles))
	for i, file := range allFiles {
		jsonData[i] = FileNodeJSON{
			Name:         file.Name(),
			Size:         file.Size(),
			IsDir:        file.IsDir(),
			DeviceName:   file.DeviceName,
			DevicePath:   file.DevicePath,
			FullPath:     file.FullPath,
			DeviceSerial: file.DeviceSerial,
		}
	}
	return serverutil.Ok().WithData(jsonData)
}

var listFilesRoute = serverutil.ApiRoute(
	"GET", "/cirrus", func(c *gin.Context) *serverutil.Response {
		return listFiles(c)
	},
)
