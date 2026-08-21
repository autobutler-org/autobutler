package v0_vault

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

var changePasswordRoute = serverutil.ApiRoute(
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

		if len(req.NewPassword) < 8 {
			return serverutil.BadRequest(fmt.Errorf("new password must be at least 8 characters"))
		}

		ctx := c.Request.Context()

		config, err := deps.VaultDB().Queries.GetVaultConfig(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get vault config: %w", err))
		}

		// Verify the current password matches what's unlocked.
		currentParams := vaultcrypto.Argon2Params{
			Memory:      uint32(config.Argon2Memory),
			Iterations:  uint32(config.Argon2Iterations),
			Parallelism: uint8(config.Argon2Parallelism),
		}
		verifyKey := vaultcrypto.DeriveKey(req.CurrentPassword, config.Salt, currentParams)
		if !vaultcrypto.CheckVerificationBlob(verifyKey, config.VerificationBlob, config.VerificationNonce) {
			vaultcrypto.ZeroKey(verifyKey)
			return &serverutil.Response{
				StatusCode:  401,
				ContentType: serverutil.ContentTypeJSON,
				Error:       fmt.Errorf("current password is incorrect"),
			}
		}
		vaultcrypto.ZeroKey(verifyKey)

		// Generate new salt and derive new key.
		newSalt, err := vaultcrypto.GenerateSalt()
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("generate salt: %w", err))
		}

		newKey := vaultcrypto.DeriveKey(req.NewPassword, newSalt, currentParams)
		defer vaultcrypto.ZeroKey(newKey)

		// Re-encrypt all entries inside a transaction so a partial failure
		// doesn't leave some entries on the old key and some on the new.
		tx, err := deps.VaultDB().Db.BeginTx(ctx, nil)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Rollback()

		qtx := deps.VaultDB().Queries.WithTx(tx)

		entries, err := qtx.ListAllVaultEntriesForReEncrypt(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list entries for re-encrypt: %w", err))
		}

		for _, entry := range entries {
			plaintext, err := vaultcrypto.Decrypt(oldKey, entry.Ciphertext, entry.Nonce)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("decrypt entry %d: %w", entry.ID, err))
			}

			var check json.RawMessage
			if err := json.Unmarshal(plaintext, &check); err != nil {
				return serverutil.InternalServerError(fmt.Errorf("corrupt entry %d: %w", entry.ID, err))
			}

			newCiphertext, newNonce, err := vaultcrypto.Encrypt(newKey, plaintext)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("re-encrypt entry %d: %w", entry.ID, err))
			}

			if err := qtx.UpdateVaultEntryCiphertext(ctx, db.UpdateVaultEntryCiphertextParams{
				Ciphertext: newCiphertext,
				Nonce:      newNonce,
				ID:         entry.ID,
			}); err != nil {
				return serverutil.InternalServerError(fmt.Errorf("save re-encrypted entry %d: %w", entry.ID, err))
			}
		}

		newVerBlob, newVerNonce, err := vaultcrypto.MakeVerificationBlob(newKey)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("create verification blob: %w", err))
		}

		if err := qtx.UpdateVaultConfigPassword(ctx, db.UpdateVaultConfigPasswordParams{
			Salt:              newSalt,
			VerificationBlob:  newVerBlob,
			VerificationNonce: newVerNonce,
		}); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("update vault config: %w", err))
		}

		if err := tx.Commit(); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("commit tx: %w", err))
		}

		// Re-unlock with the new key.
		sessionKey := vaultcrypto.DeriveKey(req.NewPassword, newSalt, currentParams)
		deps.VaultSession().Unlock(sessionKey, time.Duration(config.AutoLockSeconds)*time.Second)
		vaultcrypto.ZeroKey(sessionKey)

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"changed":     true,
			"reEncrypted": len(entries),
		})
	},
)
