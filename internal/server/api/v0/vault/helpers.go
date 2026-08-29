package v0_vault

import (
	"errors"
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

func getDeps(c *gin.Context) (deputil.Dependencies, *serverutil.Response) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return nil, serverutil.InternalServerError(nil)
	}
	return deps, nil
}

func requireVaultAvailable(c *gin.Context) (deputil.Dependencies, *serverutil.Response) {
	deps, errResp := getDeps(c)
	if errResp != nil {
		return nil, errResp
	}
	if deps.VaultDB() == nil {
		return nil, &serverutil.Response{
			StatusCode:  503,
			ContentType: serverutil.ContentTypeJSON,
			Error:       fmt.Errorf("vault storage device is disconnected"),
		}
	}
	return deps, nil
}

func requireUnlockedVault(c *gin.Context) (deputil.Dependencies, []byte, *serverutil.Response) {
	deps, errResp := requireVaultAvailable(c)
	if errResp != nil {
		return nil, nil, errResp
	}

	key, ok := deps.VaultSession().Key()
	if !ok {
		reason := deps.VaultSession().LockReason()
		msg := "vault is locked"
		if reason != "" {
			msg = fmt.Sprintf("vault is locked: %s", reason)
		}
		return nil, nil, &serverutil.Response{
			StatusCode:  423,
			ContentType: serverutil.ContentTypeJSON,
			Error:       errors.New(msg),
		}
	}

	deps.VaultSession().Touch()
	return deps, key, nil
}
