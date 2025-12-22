package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var deleteFilesRoute = serverutil.ApiRoute(
	"DELETE", "/cirrus", func(c *gin.Context) *serverutil.Response {
		rootDir := c.Query("rootDir")
		filePaths := c.QueryArray("filePaths")

		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
		}
		channel := deps.Worker().GetDeleteFilesChannel()
		channel <- cirrusutil.DeleteFilesParams{
			RootDir:   rootDir,
			FilePaths: filePaths,
		}
		return serverutil.Accepted()
	},
)
