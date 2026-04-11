package remoteutil

import (
	"fmt"
	"log"
)

// EnsureStarted starts the tsnet node, re-provisioning if local state has been
// wiped. provisionFn is called only when no persisted state exists; it should
// return a fresh Headscale pre-auth key. startProxy is called with the local
// port after tsnet starts successfully.
func EnsureStarted(localPort int, provisionFn func() (string, error)) error {
	if IsRunning() {
		return nil
	}

	authKey := ""
	if !HasPersistedState() {
		log.Printf("[remote] no persisted tsnet state, re-provisioning...")
		key, err := provisionFn()
		if err != nil {
			return fmt.Errorf("re-provision: %w", err)
		}
		authKey = key
	}

	if err := Start(authKey); err != nil {
		return fmt.Errorf("start tsnet: %w", err)
	}

	if err := StartProxy(localPort); err != nil {
		Stop()
		return fmt.Errorf("start proxy: %w", err)
	}

	return nil
}
