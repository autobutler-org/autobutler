package updateutil

import (
	"fmt"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/githubutil"
)

var DefaultUpdateSources = []*UpdateSource{
	NewUpdateSource(
		UpdateSourceKindAzure,
		"autobutlerrelease",
		"releases/autobutler",
	),
	NewUpdateSource(
		UpdateSourceKindGithub,
		"autobutler-org",
		"autobutler.org",
	),
	NewUpdateSource(
		UpdateSourceKindGithub,
		"autobutler-org",
		"autobutler",
	),
}

type UpdateSourceKind string

const (
	UpdateSourceKindAzure  UpdateSourceKind = "azure"
	UpdateSourceKindGithub UpdateSourceKind = "github"
)

type UpdateSource struct {
	Kind    UpdateSourceKind `json:"kind"`
	Account string           `json:"account"`
	Path    string           `json:"path"`
	// BaseURLOverride replaces the computed base URL when set. Used in tests to
	// point requests at a mock HTTP server instead of github.com or Azure.
	BaseURLOverride string `json:"-"`
}

// NewUpdateSource creates a new UpdateSource with the specified kind, account, and path
func NewUpdateSource(kind UpdateSourceKind, account string, path string) *UpdateSource {
	return &UpdateSource{
		Kind:    kind,
		Account: account,
		Path:    path,
	}
}

func (s *UpdateSource) BaseUrl() string {
	if s.BaseURLOverride != "" {
		return s.BaseURLOverride
	}
	switch s.Kind {
	case UpdateSourceKindGithub:
		return fmt.Sprintf("https://github.com/%s/%s/releases/download", s.Account, s.Path)
	case UpdateSourceKindAzure:
		return fmt.Sprintf("https://%s.blob.core.windows.net", s.Account)
	default:
		return ""
	}
}

func (s *UpdateSource) UpdateUrl() string {
	switch s.Kind {
	case UpdateSourceKindGithub:
		// BaseUrl() already includes /releases/download for GitHub sources.
		return s.BaseUrl()
	case UpdateSourceKindAzure:
		return fmt.Sprintf("%s/%s", s.BaseUrl(), s.Path)
	default:
		return ""
	}
}

func (s *UpdateSource) Container() string {
	if s.Kind == UpdateSourceKindAzure {
		return strings.Split(s.Path, "/")[0]
	}
	return ""
}

func (s *UpdateSource) BlobPrefix() *string {
	if s.Kind == UpdateSourceKindAzure {
		parts := strings.Split(s.Path, "/")
		if len(parts) > 1 {
			prefix := strings.Join(parts[1:], "/") + "/"
			return &prefix
		}
	}
	return nil
}

type UpdateVersion struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

func getAssetURLFromRelease(release *githubutil.Release) string {
	archiveName := ConstructArchiveName()
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.BrowserDownloadURL, archiveName) {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}
