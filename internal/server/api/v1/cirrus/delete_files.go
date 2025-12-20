package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")
		fmt.Printf("Deleting multiple files: %s\n", filePaths)

		managedDevices, err := cirrusutil.GetManagedDevices()
		if err != nil {
			managedDevices = nil
		}

		if _, err := cirrusutil.DeleteFiles(cirrusutil.DeleteFilesParams{
			RootDir:        rootDir,
			FilePaths:      filePaths,
			ManagedDevices: managedDevices,
		}); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithError(err)
		}
		return serverutil.Ok()
	},
)
