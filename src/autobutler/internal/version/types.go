package version

import "fmt"

type Version struct {
	GitCommit string
	Semver    string
	GoVersion string
	BuildDate string
}

const NoCommit = "NOCOMMIT"
const NoSemver = "NOSEMVER"

var Semver string = NoSemver

func NewVersion(gitCommit, goVersion, buildDate string) Version {
	return Version{
		GitCommit: gitCommit,
		Semver:    Semver,
		GoVersion: goVersion,
		BuildDate: buildDate,
	}
}

func (v Version) VersionString() string {
	version := ""
	if v.Semver == NoSemver {
		version = v.GitCommit
	} else {
		version = v.Semver
		if v.GitCommit != NoCommit {
			version += fmt.Sprintf("@%s", v.GitCommit)
		}
	}
	if v.BuildDate != "" {
		version += fmt.Sprintf(" from %s", v.BuildDate)
	}
	return version
}
