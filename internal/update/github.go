package update

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GitHubRelease represents a minimal GitHub release with only the fields we need
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
	// Rest stores all other fields we don't explicitly need
	Rest map[string]any `json:"-"`
}

// GitHubAsset represents a minimal GitHub release asset
type GitHubAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	// Rest stores all other fields we don't explicitly need
	Rest map[string]any `json:"-"`
}

// FetchGitHubReleases fetches all releases from the autobutler.org GitHub repository
func FetchGitHubReleases() ([]GitHubRelease, error) {
	url := "https://api.github.com/repos/autobutler-org/autobutler.org/releases"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var releases []GitHubRelease
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	return releases, nil
}
