package v1_vault

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type setupRequest struct {
	MasterPassword string `json:"masterPassword" binding:"required"`
}

var setupVaultRoute = serverutil.ApiRoute(
	"POST", "/vault/setup", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		ctx := c.Request.Context()

		// Check if vault is already initialized.
		if _, err := deps.Database().Queries.GetVaultConfig(ctx); err == nil {
			return serverutil.BadRequest(fmt.Errorf("vault is already initialized"))
		} else if !errors.Is(err, sql.ErrNoRows) {
			return serverutil.InternalServerError(fmt.Errorf("check vault config: %w", err))
		}

		var req setupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		if len(req.MasterPassword) < 8 {
			return serverutil.BadRequest(fmt.Errorf("master password must be at least 8 characters"))
		}

		salt, err := vaultcrypto.GenerateSalt()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("generate salt: %w", err))
		}

		params := vaultcrypto.DefaultParams()
		key := vaultcrypto.DeriveKey(req.MasterPassword, salt, params)
		defer vaultcrypto.ZeroKey(key)

		verBlob, verNonce, err := vaultcrypto.MakeVerificationBlob(key)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("create verification blob: %w", err))
		}

		if err := deps.Database().Queries.CreateVaultConfig(ctx, db.CreateVaultConfigParams{
			Salt:              salt,
			Argon2Memory:      int64(params.Memory),
			Argon2Iterations:  int64(params.Iterations),
			Argon2Parallelism: int64(params.Parallelism),
			VerificationBlob:  verBlob,
			VerificationNonce: verNonce,
			AutoLockSeconds:   300,
		}); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("save vault config: %w", err))
		}

		// Auto-unlock after setup.
		unlockKey := vaultcrypto.DeriveKey(req.MasterPassword, salt, params)
		deps.VaultSession().Unlock(unlockKey, 300*time.Second)

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"initialized": true,
			"locked":      false,
		})
	},
)
