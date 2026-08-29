package settingsutil

import (
	"path/filepath"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

func settingsPath() string {
	if pathOverride != "" {
		return pathOverride
	}
	dataDir := storageutil.GetDataDir()
	return filepath.Join(dataDir, settingsFileName)
}
