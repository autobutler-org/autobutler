package remoteutil

import (
	"fmt"
	"log"
)

// EnsureStarted starts the tsnet node, re-provisioning if local state has been
// wiped. provisionFn is called only when no persisted state exists; it should
// return a fresh Headscale pre-auth key. The proxy is started against
// localPort after tsnet starts successfully; localTLS must match how the
// server is serving that port.
func EnsureStarted(localPort int, localTLS bool, provisionFn func() (string, error)) error {
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

	if err := StartProxy(localPort, localTLS); err != nil {
		Stop()
		return fmt.Errorf("start proxy: %w", err)
	}

	return nil
}
