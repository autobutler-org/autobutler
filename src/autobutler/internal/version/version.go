package version

import (
	"runtime"
	"runtime/debug"
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
