package v1_files

import (
	"archive/zip"
	"autobutler/pkg/api"
	"autobutler/pkg/util/fileutil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func downloadFileRoute(group *gin.RouterGroup) {
	serverutil.ApiRoute(group, "GET", "/files/*filePath", func(c *gin.Context) *api.Response {
		filePath := c.Param("filePath")
		downloadFileImpl(c, filePath)
		return api.Ok()
	})
}

func downloadFileImpl(c *gin.Context, filePath string) {
	var fullPath string

	// Get managed devices
	managedDevices, err := storageutil.GetManagedDevices()
	if err != nil || len(managedDevices) == 0 {
		// Fallback to single device
		rootDir := fileutil.GetFilesDir()
		fullPath = filepath.Join(rootDir, filePath)
	} else {
		// Search for file across all managed devices
		var dirsToSearch []fileutil.DirWithDevice
		for _, device := range managedDevices {
			dirsToSearch = append(dirsToSearch, fileutil.DirWithDevice{
				Dir:        device.FilesDir,
				DeviceName: device.Name,
				DevicePath: device.MountPoint,
			})
		}

		fullPath, err = fileutil.FindFileAcrossDevices(dirsToSearch, filePath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
	}

	fileType := fileutil.DetermineFileTypeFromPath(fullPath)

	if fileType == fileutil.FileTypeFolder {
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		dirFs := os.DirFS(fullPath)
		err := zipWriter.AddFS(dirFs)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filepath.Base(fullPath)))
		c.Writer.Header().Set("Content-Type", "application/zip")
	} else {
		file, err := os.Open(fullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer file.Close()

		disposition := "inline"
		contentType := "application/octet-stream"
		if fileType == fileutil.FileTypePDF {
			disposition = "inline"
			contentType = "application/pdf"
		}
		c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filepath.Base(fullPath)))
		c.Header("Content-Type", contentType)
		c.File(fullPath)
	}
}
