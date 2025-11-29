package githubutil

// GitHubAsset represents a minimal GitHub release asset
type GitHubAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	// Rest stores all other fields we don't explicitly need
	Rest map[string]any `json:"-"`
}
