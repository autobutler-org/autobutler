package v1_vault

import (
	"fmt"
	"net/url"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

func getDeps(c *gin.Context) (deputil.Dependencies, *serverutil.Response) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return nil, serverutil.InternalServerError(nil)
	}
	return deps, nil
}

func requireUnlockedVault(c *gin.Context) (deputil.Dependencies, []byte, *serverutil.Response) {
	deps, errResp := getDeps(c)
	if errResp != nil {
		return nil, nil, errResp
	}

	key, ok := deps.VaultSession().Key()
	if !ok {
		return nil, nil, &serverutil.Response{
			StatusCode:  423,
			ContentType: serverutil.ContentTypeJSON,
			Error:       fmt.Errorf("vault is locked"),
		}
	}

	deps.VaultSession().Touch()
	return deps, key, nil
}

func extractURLHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// entryPayload is the decrypted JSON stored inside vault_entries.ciphertext.
type entryPayload struct {
	URL          string        `json:"url"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes,omitempty"`
	TOTPSecret   string        `json:"totpSecret,omitempty"`
	CustomFields []customField `json:"customFields,omitempty"`
}

type customField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Hidden bool   `json:"hidden"`
}

// vaultKey returns the encryption key from the vault session, or nil if locked.
func vaultKey(session *vaultcrypto.VaultSession) ([]byte, bool) {
	return session.Key()
}
