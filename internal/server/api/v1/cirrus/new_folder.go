package v1_files

import (
	"autobutler/pkg/util/ctxutil"
	"autobutler/pkg/util/deputil"
	"autobutler/pkg/util/serverutil"
	"autobutler/pkg/util/storageutil"
	"fmt"

	"github.com/gin-gonic/gin"
)

var newFolderRoute = serverutil.ApiRoute(
	"POST", "/folder/cirrus/*folderDir", func(c *gin.Context) *serverutil.Response {
		folderDir := c.Param("folderDir")
		folderName := c.PostForm("folderName")
		serial := c.Query("serial")

		deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
		if !ok {
			return serverutil.InternalServerError(fmt.Errorf("dependencies not found in context"))
		}
		channel := deps.Worker().GetCreateFolderChannel()
		channel <- storageutil.CreateFolderParams{
			FolderDir:    folderDir,
			FolderName:   folderName,
			DeviceSerial: serial,
		}
		return serverutil.Accepted()
	},
)
