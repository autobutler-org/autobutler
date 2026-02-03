package v1_files

import (
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

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

// getCirrusFilesAcrossDevices godoc
// @Summary Lists files on a file path
// @Schemes http https
// @Description merges files across all managed devices for the given filePath. If deviceSerial is empty, list files across all devices. Otherwise, only for the specified device
// @Tags cirrus
// @Produce json
// @Success 200 {array} FileNodeJSON
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Param filePath path string false "File path to list"
// @Param serial query string false "Device serial number to filter by"
// @Router /cirrus [get]
func getCirrusFilesForDevice(filePath string, deviceSerial string) ([]*storageutil.DeviceFileInfo, error) {
	devices, err := storageutil.GetManagedDevices()
	if err != nil {
		return nil, err
	}
	var selectedDevices []storageutil.ManagedDevice
	if deviceSerial == "" {
		selectedDevices = devices
	} else {
		for _, d := range devices {
			if d.UsbInfo != nil && d.UsbInfo.GetSerial() == deviceSerial {
				selectedDevices = append(selectedDevices, d)
				break
			}
		}
		if len(selectedDevices) == 0 {
			return nil, nil // Device not found, return empty
		}
	}
	var allFiles []*storageutil.DeviceFileInfo
	for _, device := range selectedDevices {
		cirrusDir := device.CirrusDir
		fullPathDir := filepath.Join(cirrusDir, filePath)
		deviceSerial := deviceSerial
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
	return allFiles, nil
}

func cirrusRouteCommon(c *gin.Context, filePath string) *serverutil.Response {
	serial := c.Query("serial")
	data, err := getCirrusFilesForDevice(filePath, serial)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(500).WithError(err)
	}

	// Convert to JSON-serializable format
	jsonData := make([]FileNodeJSON, len(data))
	for i, file := range data {
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

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(jsonData)
}

var listFilesRoute = serverutil.ApiRoute(
	"GET", "/cirrus", func(c *gin.Context) *serverutil.Response {
		return cirrusRouteCommon(c, "")
	},
)

var listFilesNestedRoute = serverutil.ApiRoute(
	"GET", "/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		return cirrusRouteCommon(c, filePath)
	},
)
