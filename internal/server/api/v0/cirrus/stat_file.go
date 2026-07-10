package v0_files

import (
	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"

	"github.com/gin-gonic/gin"
)

type StatFileJSON struct {
	IsDir    bool   `json:"isDir"`
	FileType string `json:"fileType"`
	Name     string `json:"name"`
}

// statFile godoc
// @Summary Stat a file or directory
// @Description Returns filesystem metadata for the given cirrus-relative path: whether it is a directory and its file type. Useful for deep-link resolution when the path extension alone is ambiguous (e.g. a folder named "things.abdoc").
// @Tags cirrus
// @Produce json
// @Param filePath query string true "Cirrus-relative path to stat"
// @Param serial query string false "Device serial number"
// @Success 200 {object} StatFileJSON
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Router /cirrus/stat [get]
func statFile(c *gin.Context) *serverutil.Response {
	filePath := c.Query("filePath")
	serial := c.Query("serial")

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	result, err := deps.StorageService().StatFile(storageutil.StatFileParams{
		FilePath:     filePath,
		DeviceSerial: serial,
	})
	if err != nil {
		return serverutil.NotFound(err)
	}

	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(StatFileJSON{
		IsDir:    result.IsDir,
		FileType: string(result.FileType),
		Name:     result.Name,
	})
}

var statFileRoute = serverutil.ApiRoute(
	"GET", "/cirrus/stat", statFile,
)
