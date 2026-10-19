package v0_vault

import (
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
)

type router struct{}

func (r *router) Routes() []*serverutil.Route {
	return []*serverutil.Route{
		getVaultKeyRoute,
		createVaultKeyRoute,
		deleteVaultRoute,
		listVaultItemsRoute,
		createVaultItemRoute,
		getVaultItemRoute,
		updateVaultItemRoute,
		deleteVaultItemRoute,
	}
}

// createVaultKeyBody is what PUT /vault/key reads: the wrapped vault key and
// password, the caller's auth key, which re-confirms that the key is wrapped
// under the account's login password.
type createVaultKeyBody struct {
	vaultutil.Key
	// Password is the standard base64 auth key of the caller's login
	// password, as DELETE /auth/account and PUT /auth/recovery-key take it.
	Password string `json:"password"`
}
