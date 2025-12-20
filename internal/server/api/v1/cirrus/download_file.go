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

func getCirrusFiles(filePath string) ([]*cirrusutil.DeviceFileInfo, error) {
	fullPathDir := filepath.Join(cirrusutil.GetCirrusDir(), filePath)
	deviceName, devicePath := cirrusutil.GetDeviceInfoForPath(fullPathDir)
	return cirrusutil.StatFilesInDir(fullPathDir, deviceName, devicePath)
}

func cirrusRouteCommon(c *gin.Context, filePath string) *serverutil.Response {
	data, err := getCirrusFiles(filePath)
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

var cirrusRoute = serverutil.ApiRoute(
	"GET", "/cirrus", func(c *gin.Context) *serverutil.Response {
		return cirrusRouteCommon(c, "")
	},
)

var cirrusNestedRoute = serverutil.ApiRoute(
	"GET", "/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		return cirrusRouteCommon(c, filePath)
	},
)
