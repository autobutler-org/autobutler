// Package versionutil reports and compares the build version of the binary.
package versionutil

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

type Version struct {
	GitCommit string
	Semver    string
	GoVersion string
	BuildDate string
}

const NoCommit = "NOCOMMIT"
const NoSemver = "NOSEMVER"

var Semver = NoSemver

// GitCommit is stamped by the Makefile with -ldflags, and is the reason a
// build can name itself at all.
//
// The build info below carries the same thing, but only sometimes: the
// toolchain records vcs.revision when it can see a git checkout, so a binary
// built from a source tarball, inside a container without .git, or with
// -buildvcs=false reports NOCOMMIT and has no way to say which build it is.
// A value the linker put there survives all of that.
var GitCommit = NoCommit

func NewVersion(gitCommit, goVersion, buildDate string) *Version {
	return &Version{
		GitCommit: gitCommit,
		Semver:    Semver,
		GoVersion: goVersion,
		BuildDate: buildDate,
	}
}

func GetVersion() *Version {
	version := NewVersion(GitCommit, runtime.Version(), "")
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return version // coverage: ignore
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision": // coverage: ignore
			// Only as a fallback: an unstamped `go build` still names itself,
			// while a stamped build keeps the commit the Makefile chose.
			if version.GitCommit == NoCommit {
				version.GitCommit = setting.Value
			}
		case "vcs.time": // coverage: ignore
			version.BuildDate = setting.Value
		}
	}
	return version
}

// CompareVersions compares two Version structs based on their Semver fields.
// It returns -1 if v1 < v2, 0 if v1 == v2, and 1 if v1 > v2.
// If either version has NoSemver, it returns 2.
func CompareVersions(v1, v2 Version) int {
	if v1.Semver == NoSemver || v2.Semver == NoSemver || strings.HasPrefix(v1.Semver, "dev-") || strings.HasPrefix(v2.Semver, "dev-") {
		return 2
	}
	v1Parts := strings.Split(strings.TrimPrefix(v1.Semver, "v"), ".")
	v2Parts := strings.Split(strings.TrimPrefix(v2.Semver, "v"), ".")

	for i := range 3 {
		v1Num, err := strconv.Atoi(v1Parts[i])
		if err != nil {
			return 0
		}
		v2Num, err := strconv.Atoi(v2Parts[i])
		if err != nil {
			return 0
		}
		if v1Num != v2Num {
			if v1Num < v2Num {
				return -1
			}
			return 1
		}
	}
	return 0
}
