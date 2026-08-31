package v0_vault

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var importVaultRoute = serverutil.ApiRoute(
	"POST", "/vault/import", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("file required: %w", err))
		}
		defer file.Close()

		data, err := vaultutil.ReadImport(file)
		if errors.Is(err, vaultutil.ErrImportTooLarge) {
			return serverutil.BadRequest(err)
		}
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("read file: %w", err))
		}

		format := c.DefaultPostForm("format", vaultutil.FormatAuto)

		result, err := vaultutil.Import(c.Request.Context(), vaultutil.ImportParams{
			VaultDB: deps.VaultDB(),
			Key:     key,
			Data:    data,
			Format:  format,
		})
		if errors.Is(err, vaultutil.ErrUnsupportedImportFormat) {
			return serverutil.BadRequest(fmt.Errorf("unsupported format: %s", format))
		}
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		return serverutil.Ok().WithData(result)
	},
)
