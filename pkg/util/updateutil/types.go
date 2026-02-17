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
	Kind      UpdateSourceKind `json:"kind"`
	Container string           `json:"container"`
	Path      string           `json:"path"`
}

// NewUpdateSource creates a new UpdateSource with the specified kind, container, and path
func NewUpdateSource(kind UpdateSourceKind, container string, path string) *UpdateSource {
	return &UpdateSource{
		Kind:      kind,
		Container: container,
		Path:      path,
	}
}

// NewUpdateSourceFromString creates a new UpdateSource from a string in the format "kind:container:path"
func NewUpdateSourceFromString(source string) (*UpdateSource, error) {
	parts := strings.Split(source, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid update source format: %s", source)
	}
	return &UpdateSource{
		Kind:      UpdateSourceKind(parts[0]),
		Container: parts[1],
		Path:      parts[2],
	}, nil
}

func (s *UpdateSource) BaseUrl() string {
	switch s.Kind {
	case UpdateSourceKindGithub:
		return fmt.Sprintf("https://github.com/%s/%s/releases/download", s.Container, s.Path)
	case UpdateSourceKindAzure:
		return fmt.Sprintf("https://%s.blob.core.windows.net/%s", s.Container, s.Path)
	default:
		return ""
	}
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
