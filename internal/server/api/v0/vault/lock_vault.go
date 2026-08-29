package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

var lockVaultRoute = serverutil.ApiRoute(
	"POST", "/vault/lock", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		deps.VaultSession().Lock()

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"locked": true,
		})
	},
)
