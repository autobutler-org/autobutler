package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")
		fmt.Printf("Deleting multiple files: %s\n", filePaths)

		managedDevices, err := fileutil.GetManagedDevices()
		if err != nil {
			managedDevices = nil
		}

		result, err := fileutil.DeleteFiles(fileutil.DeleteFilesParams{
			RootDir:        rootDir,
			FilePaths:      filePaths,
			ManagedDevices: managedDevices,
		})

		if err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(err.Error()))
		}

		// Always render the full file explorer (button targets #file-explorer)
		component := view_files.GetFileExplorer(c, result.RootDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
