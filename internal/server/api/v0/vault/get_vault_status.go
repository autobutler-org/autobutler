package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var getVaultStatusRoute = serverutil.ApiRoute(
	"GET", "/vault/status", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		result, err := vaultutil.Status(c.Request.Context(), vaultutil.StatusParams{
			VaultQueries: deps.VaultDB().Queries,
			MainQueries:  deps.Database().Queries,
			Session:      deps.VaultSession(),
			Storage:      deps.StorageService(),
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(vaultStatusResponse{
			Initialized:     result.Initialized,
			Locked:          result.Locked,
			AutoLockSeconds: result.AutoLockSeconds,
			StorageDevice:   result.StorageDevice,
			DeviceConnected: result.DeviceConnected,
			LockReason:      result.LockReason,
		})
	},
)
