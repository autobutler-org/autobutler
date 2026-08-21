package v0_vault

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type unlockRequest struct {
	MasterPassword string `json:"masterPassword" binding:"required"`
}

var unlockVaultRoute = serverutil.ApiRoute(
	"POST", "/vault/unlock", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		ctx := c.Request.Context()

		config, err := deps.VaultDB().Queries.GetVaultConfig(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return serverutil.BadRequest(fmt.Errorf("vault is not initialized — call POST /vault/setup first"))
		}
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get vault config: %w", err))
		}

		var req unlockRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		params := vaultcrypto.Argon2Params{
			Memory:      uint32(config.Argon2Memory),
			Iterations:  uint32(config.Argon2Iterations),
			Parallelism: uint8(config.Argon2Parallelism),
		}

		key := vaultcrypto.DeriveKey(req.MasterPassword, config.Salt, params)

		if !vaultcrypto.CheckVerificationBlob(key, config.VerificationBlob, config.VerificationNonce) {
			vaultcrypto.ZeroKey(key)
			return &serverutil.Response{
				StatusCode:  401,
				ContentType: serverutil.ContentTypeJSON,
				Error:       fmt.Errorf("incorrect master password"),
			}
		}

		timeout := time.Duration(config.AutoLockSeconds) * time.Second
		deps.VaultSession().Unlock(key, timeout)

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"locked": false,
		})
	},
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
