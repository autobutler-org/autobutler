package version

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

func GetVersion() Version {
	info, ok := debug.ReadBuildInfo()
	version := NewVersion(NoCommit, runtime.Version(), "")
	if !ok || info == nil {
		return version
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			version.GitCommit = setting.Value
		case "vcs.time":
			version.BuildDate = setting.Value
		}
	}
	return version
}

// CompareVersions compares two Version structs based on their Semver fields.
// It returns -1 if v1 < v2, 0 if v1 == v2, and 1 if v1 > v2.
// If either version has NoSemver, it returns 2.
func CompareVersions(v1, v2 Version) int {
	if v1.Semver == NoSemver || v2.Semver == NoSemver {
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
