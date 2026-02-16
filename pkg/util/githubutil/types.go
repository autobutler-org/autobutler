package githubutil

// Release represents a minimal GitHub release with only the fields we need
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
	// Rest stores all other fields we don't explicitly need
	Rest map[string]any `json:"-"`
}

// Asset represents a minimal GitHub release asset
type Asset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	// Rest stores all other fields we don't explicitly need
	Rest map[string]any `json:"-"`
}
