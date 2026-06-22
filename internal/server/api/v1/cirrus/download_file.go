package v1_files

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/photoutil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	_ "github.com/gen2brain/heic"
	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// downloadCirrusFile godoc
// @Summary Download a file or folder
// @Description Downloads a single file or zips a folder and streams it back to the client
// @Tags cirrus
// @Produce application/octet-stream
// @Param filePath query string false "File path to download"
// @Param serial query string false "Device serial number to filter by"
// @Param format query string false "Output format conversion (e.g. 'jpeg' to convert HEIC to JPEG)"
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

	f, err := os.Open(result.FullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return serverutil.NotFound(fmt.Errorf("file not found: %s", filePath))
		}
		return serverutil.InternalServerError(fmt.Errorf("failed to open file: %w", err))
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(result.FullPath))
	wantsJPEG := c.Query("format") == "jpeg"

	if wantsJPEG && result.FileType == storageutil.FileTypeImage {
		baseName := strings.TrimSuffix(filepath.Base(result.FullPath), ext) + ".jpg"

		if photoutil.IsRawFile(result.FullPath) {
			jpegBytes, err := photoutil.RawToJPEGBytes(result.FullPath, 92)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("failed to convert RAW to JPEG: %w", err))
			}
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", baseName))
			c.Data(http.StatusOK, "image/jpeg", jpegBytes)
			return nil
		}

		img, _, err := image.Decode(f)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to decode image: %w", err))
		}

		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("failed to encode JPEG: %w", err))
		}

		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", baseName))
		c.Data(http.StatusOK, "image/jpeg", buf.Bytes())
		return nil
	}

	f.Close() // close before c.File re-opens it
	disposition := "inline"
	contentType := "application/octet-stream"
	if result.FileType == storageutil.FileTypePDF {
		contentType = "application/pdf"
	} else if result.FileType == storageutil.FileTypeImage {
		contentType = storageutil.ImageMIMETypeFromExtension(filepath.Ext(result.FullPath))
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
