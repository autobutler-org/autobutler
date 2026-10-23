package githubutil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/githubutil"
)

func mockReleasesServer(t *testing.T, releases []githubutil.Release) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Expected Accept header application/vnd.github+json, got %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("Expected X-GitHub-Api-Version 2022-11-28, got %s", r.Header.Get("X-GitHub-Api-Version"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(releases)
	}))
}

func mockLatestReleaseServer(t *testing.T, release githubutil.Release) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(release)
	}))
}

func mockErrorServer(statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))
}

func TestFetchGitHubReleases_Success(t *testing.T) {
	mockReleases := []githubutil.Release{
		{
			TagName: "v1.0.0",
			Assets: []githubutil.Asset{
				{BrowserDownloadURL: "https://example.com/v1.0.0/binary.tar.gz"},
			},
		},
		{
			TagName: "v1.1.0",
			Assets: []githubutil.Asset{
				{BrowserDownloadURL: "https://example.com/v1.1.0/binary.tar.gz"},
			},
		},
	}

	server := mockReleasesServer(t, mockReleases)
	defer server.Close()
	reset := githubutil.SetBaseURLForTesting(server.URL)
	defer reset()

	releases, err := githubutil.FetchReleases("test-org", "test-repo")
	if err != nil {
		t.Fatalf("FetchReleases failed: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("Expected 2 releases, got %d", len(releases))
	}
	if releases[0].TagName != "v1.0.0" {
		t.Errorf("Expected first release tag v1.0.0, got %s", releases[0].TagName)
	}
	if releases[1].TagName != "v1.1.0" {
		t.Errorf("Expected second release tag v1.1.0, got %s", releases[1].TagName)
	}
}

func TestFetchGitHubReleases_Empty(t *testing.T) {
	server := mockReleasesServer(t, []githubutil.Release{})
	defer server.Close()
	reset := githubutil.SetBaseURLForTesting(server.URL)
	defer reset()

	releases, err := githubutil.FetchReleases("test-org", "test-repo")
	if err != nil {
		t.Fatalf("FetchReleases failed: %v", err)
	}
	if len(releases) != 0 {
		t.Errorf("Expected 0 releases, got %d", len(releases))
	}
}

func TestFetchGitHubReleases_HTTPError(t *testing.T) {
	server := mockErrorServer(http.StatusNotFound)
	defer server.Close()
	reset := githubutil.SetBaseURLForTesting(server.URL)
	defer reset()

	_, err := githubutil.FetchReleases("test-org", "test-repo")
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestFetchLatestRelease_Success(t *testing.T) {
	mockRelease := githubutil.Release{
		TagName: "v2.0.0",
		Assets: []githubutil.Asset{
			{BrowserDownloadURL: "https://example.com/v2.0.0/binary.tar.gz"},
		},
	}

	server := mockLatestReleaseServer(t, mockRelease)
	defer server.Close()
	reset := githubutil.SetBaseURLForTesting(server.URL)
	defer reset()

	release, err := githubutil.FetchLatestRelease("test-org", "test-repo")
	if err != nil {
		t.Fatalf("FetchLatestRelease failed: %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Errorf("Expected tag v2.0.0, got %s", release.TagName)
	}
	if len(release.Assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(release.Assets))
	}
}

func TestFetchLatestRelease_HTTPError(t *testing.T) {
	server := mockErrorServer(http.StatusNotFound)
	defer server.Close()
	reset := githubutil.SetBaseURLForTesting(server.URL)
	defer reset()

	_, err := githubutil.FetchLatestRelease("test-org", "test-repo")
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}
