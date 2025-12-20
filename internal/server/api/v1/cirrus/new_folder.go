package v1_files

import (
	"autobutler/pkg/util/cirrusutil"
	"autobutler/pkg/util/serverutil"

	"github.com/gin-gonic/gin"
)

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/folder/cirrus/*folderDir", func(c *gin.Context) *serverutil.Response {
		folderDir := c.Param("folderDir")
		folderName := c.PostForm("folderName")

		if _, err := cirrusutil.CreateFolder(cirrusutil.CreateFolderParams{
			FolderDir:  folderDir,
			FolderName: folderName,
		}); err != nil {
			return serverutil.NewResponse().WithStatusCode(500).WithError(err)
		}
		return serverutil.Ok()
	},
)
