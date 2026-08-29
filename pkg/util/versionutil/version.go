package versionutil

import "fmt"

func (v *Version) VersionString() string {
	version := ""
	if v.Semver == NoSemver {
		version = v.GitCommit
	} else {
		version = v.Semver // coverage: ignore
		if v.GitCommit != NoCommit {
			version += fmt.Sprintf("@%s", v.GitCommit) // coverage: ignore
		}
	}
	if v.BuildDate != "" {
		version += fmt.Sprintf(" from %s", v.BuildDate) // coverage: ignore
	}
	return version
}
