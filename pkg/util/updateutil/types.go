package updateutil

import (
	"autobutler/pkg/util/githubutil"
	"fmt"
	"strings"
)

var DefaultUpdateSources = []*UpdateSource{
	NewUpdateSource(
		UpdateSourceKindAzure,
		"autobutlerrelease",
		"releases/autobutler",
	),
	// NewUpdateSource(
	// 	UpdateSourceKindGithub,
	// 	"autobutler-org",
	// 	"autobutler.org",
	// ),
	// NewUpdateSource(
	// 	UpdateSourceKindGithub,
	// 	"autobutler-org",
	// 	"autobutler",
	// ),
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
}

// NewUpdateSource creates a new UpdateSource with the specified kind, account, and path
func NewUpdateSource(kind UpdateSourceKind, account string, path string) *UpdateSource {
	return &UpdateSource{
		Kind:    kind,
		Account: account,
		Path:    path,
	}
}

// NewUpdateSourceFromString creates a new UpdateSource from a string in the format "kind:account:path"
func NewUpdateSourceFromString(source string) (*UpdateSource, error) {
	parts := strings.Split(source, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid update source format: %s", source)
	}
	return &UpdateSource{
		Kind:    UpdateSourceKind(parts[0]),
		Account: parts[1],
		Path:    parts[2],
	}, nil
}

func (s *UpdateSource) BaseUrl() string {
	switch s.Kind {
	case UpdateSourceKindGithub:
		return fmt.Sprintf("https://github.com/%s/%s/releases/download", s.Account, s.Path)
	case UpdateSourceKindAzure:
		return fmt.Sprintf("https://%s.blob.core.windows.net/", s.Account)
	default:
		return ""
	}
}

func (s *UpdateSource) UpdateUrl() string {
	switch s.Kind {
	case UpdateSourceKindGithub:
		return fmt.Sprintf("%s/releases/download", s.BaseUrl())
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
			prefix := strings.Join(parts[1:], "/")
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
	url := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.BrowserDownloadURL, ".tar.gz") {
			url = asset.BrowserDownloadURL
			break
		}
	}
	return url
}
