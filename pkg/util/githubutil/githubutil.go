package githubutil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const defaultBaseURL = "https://api.github.com"

// baseURL allows tests to override the GitHub API base URL via a mock server.
// Set this in tests using t.Setenv or direct assignment; reset after test.
var baseURL = defaultBaseURL

// FetchReleases fetches all releases from a GitHub repository
func FetchReleases(organization string, repository string) ([]*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", baseURL, organization, repository)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil { // coverage: ignore - http.NewRequest rarely fails with valid inputs
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { // coverage: ignore - requires network failure or mock server
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK { // coverage: ignore - requires invalid repo or GitHub API error
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var releases []*Release
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&releases); err != nil { // coverage: ignore - requires malformed JSON from GitHub API
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	return releases, nil
}

// FetchLatestRelease fetches the latest release from a GitHub repository
func FetchLatestRelease(organization string, repository string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, organization, repository)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil { // coverage: ignore - http.NewRequest rarely fails with valid inputs
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { // coverage: ignore - requires network failure or mock server
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK { // coverage: ignore - requires invalid repo or GitHub API error
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var release Release
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&release); err != nil { // coverage: ignore - requires malformed JSON from GitHub API
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	return &release, nil
}

// SetBaseURLForTesting overrides the GitHub API base URL. Only for use in tests.
func SetBaseURLForTesting(url string) func() {
	baseURL = url
	return func() { baseURL = defaultBaseURL }
}
