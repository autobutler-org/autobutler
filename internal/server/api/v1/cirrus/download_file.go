package v1_files

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
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

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}
	result, err := deps.StorageService().DownloadFile(storageutil.DownloadFileParams{
		FilePath:     filePath,
		DeviceSerial: serial,
	})
	if err != nil {
		return serverutil.NotFound(err)
	}

	if result.IsFolder {
		zipWriter := zip.NewWriter(c.Writer)
		defer zipWriter.Close()
		dirFs := os.DirFS(result.FullPath)
		if err := zipWriter.AddFS(dirFs); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to zip folder: %w", err))
		}
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filepath.Base(result.FullPath)))
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
		return nil // response written directly to writer
	}

	if _, err := os.Stat(result.FullPath); os.IsNotExist(err) {
		return serverutil.NotFound(fmt.Errorf("file not found: %s", filePath))
	}

	disposition := "inline"
	contentType := "application/octet-stream"
	if result.FileType == storageutil.FileTypePDF {
		contentType = "application/pdf"
	} else if result.FileType == storageutil.FileTypeImage {
		contentType = "image/*"
	} else if result.FileType == storageutil.FileTypeVideo {
		contentType = storageutil.VideoMIMETypeFromExtension(filepath.Ext(result.FullPath))
	}
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filepath.Base(result.FullPath)))
	c.Header("Content-Type", contentType)
	c.File(result.FullPath)
	return nil // response written directly via c.File
}

var downloadFileRoute = serverutil.ApiRoute(
	"GET", "/cirrus/download", downloadFile,
)
