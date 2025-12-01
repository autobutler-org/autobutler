package v1_files

import (
	"github.com/gin-gonic/gin"
)

func SetupRoutes(group *gin.RouterGroup) {
	deleteFilesRoute(group)
	downloadFileRoute(group)
	newFolderRoute(group)
	moveFileRoute(group)
	uploadFilesRoute(group)
}
