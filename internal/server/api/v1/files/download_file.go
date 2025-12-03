package v1_files

import (
	"archive/zip"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var downloadFileRoute = serverutil.ApiRoute(
	"GET", "/files/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")

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

		result := fileutil.DownloadFile(fileutil.DownloadFileParams{
			FilePath:       filePath,
			ManagedDevices: managedDevices,
		})

		if result.Error != nil {
			c.Status(http.StatusNotFound)
			return serverutil.Ok()
		}

		if result.IsFolder {
			zipWriter := zip.NewWriter(c.Writer)
			defer zipWriter.Close()
			dirFs := os.DirFS(result.FullPath)
			err := zipWriter.AddFS(dirFs)
			if err != nil {
				c.Status(http.StatusInternalServerError)
				return serverutil.Ok()
			}
			c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filepath.Base(result.FullPath)))
			c.Writer.Header().Set("Content-Type", "application/zip")
		} else {
			file, err := os.Open(result.FullPath)
			if err != nil {
				c.Status(http.StatusNotFound)
				return serverutil.Ok()
			}
			defer file.Close()

			disposition := "inline"
			contentType := "application/octet-stream"
			if result.FileType == fileutil.FileTypePDF {
				disposition = "inline"
				contentType = "application/pdf"
			}
			c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filepath.Base(result.FullPath)))
			c.Header("Content-Type", contentType)
			c.File(result.FullPath)
		}
		return serverutil.Ok()
	},
)
