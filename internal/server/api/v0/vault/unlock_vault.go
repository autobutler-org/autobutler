package v0_vault

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var unlockVaultRoute = serverutil.ApiRoute(
	"POST", "/vault/unlock", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		var req unlockRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		result, err := vaultutil.Unlock(c.Request.Context(), vaultutil.UnlockParams{
			Queries:        deps.VaultDB().Queries,
			Session:        deps.VaultSession(),
			MasterPassword: req.MasterPassword,
		})
		switch {
		case errors.Is(err, vaultutil.ErrVaultNotInitialized):
			return serverutil.BadRequest(err)
		case errors.Is(err, vaultutil.ErrIncorrectMasterPassword):
			return &serverutil.Response{
				StatusCode:  401,
				ContentType: serverutil.ContentTypeJSON,
				Error:       err,
			}
		case err != nil:
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"locked": result.Locked,
		})
	},
)
