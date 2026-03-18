package v1_files

import (
	"archive/zip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// downloadCirrusFile godoc
// @Summary Download a file or folder
// @Description Downloads a single file or zips a folder and streams it back to the client
// @Tags cirrus
// @Produce application/octet-stream
// @Param filePath query string false "File path to download"
// @Param serial query string false "Device serial number to filter by"
// @Success 200 {file} file
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/download [get]
func downloadFile(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	serial := c.Query("serial")

	result, err := storageutil.DownloadFile(storageutil.DownloadFileParams{
		FilePath:     filePath,
		DeviceSerial: serial,
	})

	if err != nil {
		c.Status(http.StatusNotFound)
		return nil
	}

	if result.IsFolder {
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		dirFs := os.DirFS(result.FullPath)
		err := zipWriter.AddFS(dirFs)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filepath.Base(result.FullPath)))
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
	} else {
		file, err := os.Open(result.FullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return nil
		}
		defer file.Close()

		disposition := "inline"
		contentType := "application/octet-stream"
		if result.FileType == storageutil.FileTypePDF {
			disposition = "inline"
			contentType = "application/pdf"
		} else if result.FileType == storageutil.FileTypeImage {
			disposition = "inline"
			contentType = "image/*"
		} else if result.FileType == storageutil.FileTypeVideo {
			disposition = "inline"
			contentType = storageutil.VideoMIMETypeFromExtension(filepath.Ext(result.FullPath))
		}
		c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filepath.Base(result.FullPath)))
		c.Header("Content-Type", contentType)
		c.File(result.FullPath)
	}
	return nil
}

var downloadFileRoute = serverutil.ApiRoute(
	"GET", "/cirrus/download", downloadFile,
)
