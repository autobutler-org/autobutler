package v1_files

import (
	"autobutler/pkg/api"
	"autobutler/pkg/ui"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"
	"html"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func deleteFilesRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "DELETE", "/files", func(c *gin.Context) *api.Response {
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
					return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
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
							return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">` + html.EscapeString(err.Error()) + `</span>`)
						}
					}
				}
			}
		}

		// Always render the full file explorer (button targets #file-explorer)
		component := ui.GetFileExplorer(c, rootDir)
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			return api.NewResponse().WithStatusCode(500).WithData(`<span class="text-red-500">Failed to render file explorer: ` + html.EscapeString(err.Error()) + `</span>`)
		}
		return api.Ok()
	})
}
