package githubutil

// GitHubRelease represents a minimal GitHub release with only the fields we need
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
	// Rest stores all other fields we don't explicitly need
	Rest map[string]any `json:"-"`
}
