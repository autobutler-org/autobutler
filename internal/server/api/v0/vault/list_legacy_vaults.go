package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/storageutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"

	"github.com/gin-gonic/gin"
)

// listLegacyVaults godoc
// @Summary List the household vaults an earlier Quark left on this device
// @Description Lists the shared household vault from before per-account vaults (#3616): the copy kept beside the database when an upgrade dropped its tables, and any vault.db an older Quark left on a drive. None of it is in anyone's new vault, because it is encrypted under the old shared master password. Nothing is changed or deleted; the list is empty when there is nothing to report.
// @Tags vault
// @Produce json
// @Success 200 {object} vaultutil.ListLegacyVaultsResult
// @Failure 401 {object} serverutil.Response
// @Failure 500 {object} serverutil.Response
// @Security BearerAuth
// @Router /vault/legacy [get]
func listLegacyVaults(c *gin.Context) *serverutil.Response {
	deps, _, failed := callerContext(c)
	if failed != nil {
		return failed
	}
	params := vaultutil.ListLegacyVaultsParams{DataDir: storageutil.GetDataDir()}
	if storage := deps.StorageService(); storage != nil {
		params.Storage = storage
	}
	result, err := vaultutil.ListLegacyVaults(params)
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return serverutil.Ok().WithData(result)
}

var listLegacyVaultsRoute = serverutil.ApiRoute("GET", "/vault/legacy", listLegacyVaults)
