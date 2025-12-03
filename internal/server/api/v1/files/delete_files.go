package v1_files

import (
	"autobutler/pkg/ui/components/error_message"
	view_files "autobutler/pkg/ui/views/files"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/files", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")
		fmt.Printf("Deleting multiple files: %s\n", filePaths)

		// Get managed devices
		managedDevices, err := storageutil.GetManagedDevices()
		if err != nil || len(managedDevices) == 0 {
			// Fallback to single device
			fileDir := fileutil.GetFilesDir()
			for _, filePath := range filePaths {
				fullPath := filepath.Join(fileDir, rootDir, filePath)
				if err := os.RemoveAll(fullPath); err != nil {
					var buf bytes.Buffer
					error_message.Component(err.Error()).Render(c.Request.Context(), &buf)
					return serverutil.NewResponse().WithStatusCode(500).WithData(buf.String())
				}
			}
		} else {
			// Build list of device directories
			var dirsToSearch []fileutil.DirWithDevice
			for _, device := range managedDevices {
				dirsToSearch = append(dirsToSearch, fileutil.DirWithDevice{
					Dir:        device.FilesDir,
					DeviceName: device.Name,
					DevicePath: device.MountPoint,
				})
			}

			// Delete files from all devices where they exist
			for _, filePath := range filePaths {
				relPath := filepath.Join(rootDir, filePath)
				// Try to find and delete from each device
				for _, dirInfo := range dirsToSearch {
					fullPath := filepath.Join(dirInfo.Dir, relPath)
					if _, err := os.Stat(fullPath); err == nil {
						if err := os.RemoveAll(fullPath); err != nil {
							var buf bytes.Buffer
							error_message.Component(err.Error()).Render(c.Request.Context(), &buf)
							return serverutil.NewResponse().WithStatusCode(500).WithData(buf.String())
						}
					}
				}
			}
		}

		// Always render the full file explorer (button targets #file-explorer)
		component := view_files.GetFileExplorer(c, rootDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			var buf bytes.Buffer
			error_message.Component("Failed to render file explorer: "+err.Error()).Render(c.Request.Context(), &buf)
			return serverutil.NewResponse().WithStatusCode(500).WithData(buf.String())
		}
		return serverutil.Ok()
	},
)
