package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

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
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(data)
}

var cirrusRoute = serverutil.ApiRoute(
	"GET", "/json/cirrus", func(c *gin.Context) *serverutil.Response {
		return cirrusRouteCommon(c, "")
	},
)

var cirrusNestedRoute = serverutil.ApiRoute(
	"GET", "/json/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		return cirrusRouteCommon(c, filePath)
	},
)
