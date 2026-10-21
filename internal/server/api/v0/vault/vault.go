// Package v0_vault serves /api/v0/vault, each account's end-to-end encrypted vault (#3456): the account's wrapped
// vault key and its encrypted items. The client encrypts everything, and the Quark stores and returns ciphertext it
// never opens. Every route acts on the caller's own account and nothing is admin-only: an admin reads another
// account's vault as not found, like anyone else. GET /vault/legacy is the one route about the device rather than
// an account: it lists the shared household vault an earlier Quark left behind (#3616), which every account used.
package v0_vault

import "github.com/autobutler-org/quark/pkg/util/serverutil"

// NewRouter returns every vault route. Mount it in the signed-in group.
func NewRouter() serverutil.Router {
	return &router{}
}
