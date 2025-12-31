package v1_files

import (
	"archive/zip"
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var downloadFileRoute = serverutil.ApiRoute(
	"GET", "/download/cirrus/*filePath", func(c *gin.Context) *serverutil.Response {
		filePath := c.Param("filePath")
		deviceName := c.Query("device")

		result, err := cirrusutil.DownloadFile(cirrusutil.DownloadFileParams{
			FilePath:   filePath,
			DeviceName: deviceName,
		})

		if err != nil {
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
			if result.FileType == cirrusutil.FileTypePDF {
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
