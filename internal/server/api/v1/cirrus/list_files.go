package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// FileNodeJSON is a JSON-serializable representation of a file node
type FileNodeJSON struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"isDir"`
	DeviceName string `json:"deviceName"`
	DevicePath string `json:"devicePath"`
	FullPath   string `json:"fullPath"`
}

// getCirrusFilesAcrossDevices merges files across all managed devices for the given filePath
// If deviceName is empty, list files across all devices. Otherwise, only for the specified device.
func getCirrusFilesForDevice(filePath string, deviceName string) ([]*cirrusutil.DeviceFileInfo, error) {
	devices, err := cirrusutil.GetManagedDevices()
	if err != nil {
		return nil, err
	}
	var selectedDevices []cirrusutil.ManagedDevice
	if deviceName == "" {
		selectedDevices = devices
	} else {
		for _, d := range devices {
			if d.Name == deviceName {
				selectedDevices = append(selectedDevices, d)
				break
			}
		}
		if len(selectedDevices) == 0 {
			return nil, nil // Device not found, return empty
		}
	}
	var allFiles []*cirrusutil.DeviceFileInfo
	for _, device := range selectedDevices {
		cirrusDir := device.CirrusDir
		fullPathDir := filepath.Join(cirrusDir, filePath)
		files, err := cirrusutil.StatFilesInDir(fullPathDir, device.Name, device.DataDir)
		if err != nil {
			continue
		}
		allFiles = append(allFiles, files...)
	}
	return allFiles, nil
}

func cirrusRouteCommon(c *gin.Context, filePath string) *serverutil.Response {
	deviceName := c.Query("device")
	data, err := getCirrusFilesForDevice(filePath, deviceName)
	if err != nil {
		return serverutil.NewResponse().WithStatusCode(500).WithError(err)
	}

	// Convert to JSON-serializable format
	jsonData := make([]FileNodeJSON, len(data))
	for i, file := range data {
		jsonData[i] = FileNodeJSON{
			Name:       file.Name(),
			Size:       file.Size(),
			IsDir:      file.IsDir(),
			DeviceName: file.DeviceName,
			DevicePath: file.DevicePath,
			FullPath:   file.FullPath,
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
