package v0_files

import (
	"path/filepath"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

// listArchive godoc
// @Summary List contents of an archive file at a given virtual path
// @Description Opens the archive at filePath and returns the direct children of subPath as FileNodeJSON entries. No data is extracted to disk — only archive headers are read.
// @Tags cirrus
// @Produce json
// @Param filePath query string true "Path to the archive file (relative to device cirrus directory)"
// @Param subPath query string false "Virtual subdirectory inside the archive to list (empty = root)"
// @Param serial query string false "Device serial number"
// @Success 200 {array} FileNodeJSON
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/list-archive [get]
func listArchive(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	if filePath == "" {
		return serverutil.BadRequest(nil)
	}

	subPath := c.Query("subPath")
	serial := c.Query("serial")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	entries, err := deps.StorageService().ListArchiveEntries(storageutil.ListArchiveParams{
		FilePath:     filePath,
		SubPath:      subPath,
		DeviceSerial: serial,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}

	result := make([]FileNodeJSON, len(entries))
	for i, e := range entries {
		// Construct a virtual dirPath: filePath/subPath/name
		// This is the path the client must pass back as filePath to list deeper.
		virtualPath := filePath
		if subPath != "" {
			virtualPath = filepath.ToSlash(filepath.Join(virtualPath, subPath))
		}
		dirPath := filepath.ToSlash(filepath.Join(virtualPath, e.Name))

		fileType := ""
		if !e.IsDir {
			fileType = string(storageutil.DetermineFileTypeFromPath(e.Name))
		}

		// Strip any archive extension segments from the path for display
		// but preserve the full virtual path for navigation.
		nameParts := strings.Split(e.Name, "/")
		displayName := nameParts[len(nameParts)-1]

		result[i] = FileNodeJSON{
			Name:           displayName,
			Size:           e.Size,
			CompressedSize: e.CompressedSize,
			IsDir:          e.IsDir,
			DeviceName:     "",
			DevicePath:     "",
			DirPath:        dirPath,
			FullPath:       dirPath,
			DeviceSerial:   serial,
			FileType:       fileType,
		}
	}

	return serverutil.Ok().WithData(result)
}

var listArchiveRoute = serverutil.ApiRoute(
	"GET", "/cirrus/list-archive", func(c *gin.Context) *serverutil.Response {
		return listArchive(c)
	},
)
