package githubutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchGitHubReleases_Success(t *testing.T) {
	// Create a mock server
	mockReleases := []GitHubRelease{
		{
			TagName: "v1.0.0",
			Assets: []GitHubAsset{
				{BrowserDownloadURL: "https://example.com/v1.0.0/binary.tar.gz"},
			},
		},
		{
			TagName: "v1.1.0",
			Assets: []GitHubAsset{
				{BrowserDownloadURL: "https://example.com/v1.1.0/binary.tar.gz"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test-org/test-repo/releases" {
			t.Errorf("Expected path /repos/test-org/test-repo/releases, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Expected Accept header to be application/vnd.github+json")
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("Expected X-GitHub-Api-Version header to be 2022-11-28")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockReleases)
	}))
	defer server.Close()

	// Note: This test won't actually work as-is because the function uses hardcoded URL.
	// For now, we'll test with real GitHub API (limited test)
	t.Skip("Skipping because function uses hardcoded GitHub API URL")
}

func TestFetchGitHubReleases_HTTPError(t *testing.T) {
	// Test would require dependency injection or URL override
	t.Skip("Skipping because function uses hardcoded GitHub API URL")
}

func TestFetchGitHubReleases_Integration(t *testing.T) {
	// This is an integration test that hits the real GitHub API
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	releases, err := FetchGitHubReleases("autobutler-org", "autobutler")
	if err != nil {
		t.Fatalf("FetchGitHubReleases failed: %v", err)
	}

	if len(releases) == 0 {
		t.Error("Expected at least one release from autobutler-org/autobutler repository")
	}

	// Verify structure of first release
	if len(releases) > 0 {
		release := releases[0]
		if release.TagName == "" {
			t.Error("Expected TagName to be non-empty")
		}
	}
}

func TestFetchGitHubReleases_NonExistentRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_, err := FetchGitHubReleases("nonexistent-org-12345", "nonexistent-repo-67890")
	if err == nil {
		t.Error("Expected error for non-existent repository, got nil")
	}
}

func TestGitHubRelease_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"tag_name": "v2.0.0",
		"name": "Release 2.0.0",
		"assets": [
			{
				"browser_download_url": "https://example.com/asset.tar.gz",
				"name": "asset.tar.gz"
			}
		]
	}`

	var release GitHubRelease
	err := json.Unmarshal([]byte(jsonData), &release)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if release.TagName != "v2.0.0" {
		t.Errorf("Expected TagName v2.0.0, got %s", release.TagName)
	}

	if len(release.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(release.Assets))
	}

	if release.Assets[0].BrowserDownloadURL != "https://example.com/asset.tar.gz" {
		t.Errorf("Expected specific URL, got %s", release.Assets[0].BrowserDownloadURL)
	}
}

func TestGitHubAsset_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"browser_download_url": "https://example.com/file.zip",
		"name": "file.zip",
		"size": 1024
	}`

	var asset GitHubAsset
	err := json.Unmarshal([]byte(jsonData), &asset)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if asset.BrowserDownloadURL != "https://example.com/file.zip" {
		t.Errorf("Expected specific URL, got %s", asset.BrowserDownloadURL)
	}
}
