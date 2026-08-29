package v0_vault

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var changeVaultPasswordRoute = serverutil.ApiRoute(
	"PUT", "/vault/change-password", func(c *gin.Context) *serverutil.Response {
		deps, oldKey, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(oldKey)

		var req changePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		result, err := vaultutil.ChangePassword(c.Request.Context(), vaultutil.ChangePasswordParams{
			VaultDB:         deps.VaultDB(),
			Session:         deps.VaultSession(),
			OldKey:          oldKey,
			CurrentPassword: req.CurrentPassword,
			NewPassword:     req.NewPassword,
		})
		switch {
		case errors.Is(err, vaultutil.ErrNewPasswordTooShort):
			return serverutil.BadRequest(err)
		case errors.Is(err, vaultutil.ErrIncorrectCurrentPassword):
			return &serverutil.Response{
				StatusCode:  401,
				ContentType: serverutil.ContentTypeJSON,
				Error:       err,
			}
		case err != nil:
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"changed":     result.Changed,
			"reEncrypted": result.ReEncrypted,
		})
	},
)
