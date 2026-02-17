package v1_version

import "autobutler/pkg/util/updateutil"

// UpdateParams defines the parameters for updating to a specific version
type UpdateParams struct {
	Version string                   `json:"version"`
	Source  *updateutil.UpdateSource `json:"source,omitempty"`
	Force   bool                     `json:"force,omitempty"`
}

// VersionJSON is the JSON representation of the current version
type VersionJSON struct {
	Semver    string `json:"semver"`
	GitCommit string `json:"gitCommit"`
	GoVersion string `json:"goVersion"`
	BuildDate string `json:"buildDate"`
}
