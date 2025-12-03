package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/files", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")
		fmt.Printf("Deleting multiple files: %s\n", filePaths)

		var managedDevices []fileutil.ManagedDevice
		storageDevices, err := storageutil.GetManagedDevices()
		if err == nil {
			for _, d := range storageDevices {
				managedDevices = append(managedDevices, fileutil.ManagedDevice{
					Name:       d.Name,
					MountPoint: d.MountPoint,
					FilesDir:   d.FilesDir,
				})
			}
		}

		result := fileutil.DeleteFiles(fileutil.DeleteFilesParams{
			RootDir:        rootDir,
			FilePaths:      filePaths,
			ManagedDevices: managedDevices,
		})

		if result.Error != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component(result.Error.Error()))
		}

		// Always render the full file explorer (button targets #file-explorer)
		component := view_files.GetFileExplorer(c, result.RootDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithComponent(error_message.Component("Failed to render file explorer: " + err.Error()))
		}
		return serverutil.Ok()
	},
)
