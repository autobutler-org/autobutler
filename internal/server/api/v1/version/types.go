package v1_version

// ReleaseJSON is the JSON representation of a release for the Vue frontend
type ReleaseJSON struct {
	TagName          string `json:"tagName"`
	Name             string `json:"name"`
	HtmlUrl          string `json:"htmlUrl"`
	PublishedAt      string `json:"publishedAt"`
	IsCurrentVersion bool   `json:"isCurrentVersion"`
}

// VersionJSON is the JSON representation of the current version
type VersionJSON struct {
	Semver    string `json:"semver"`
	GitCommit string `json:"gitCommit"`
	GoVersion string `json:"goVersion"`
	BuildDate string `json:"buildDate"`
}
