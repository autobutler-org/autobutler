package v1_files

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")
		serial := c.Query("serial")

		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
		}
		channel := deps.Worker().GetDeleteFilesChannel()
		channel <- storageutil.DeleteFilesParams{
			RootDir:      rootDir,
			FilePaths:    filePaths,
			DeviceSerial: serial,
		}
		return serverutil.Accepted()
	},
)
