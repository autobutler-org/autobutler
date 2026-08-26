package updateutil

import (
	"fmt"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/githubutil"
)

// DefaultUpdateSources is the ordered list of places to look for releases.
// Every entry must actually publish assets named by ConstructArchiveName
// ("quark_<Os>_<arch>.tar.gz") — a source that cannot serve those only burns a
// round trip and buries the real failure inside a concatenated error string.
//
// autobutler-org/quark.org was removed here: no such repository exists. It was
// half-applied find-and-replace fallout from the AutoButler -> Quark rename
// (#1610). The nearby real repos are not substitutes — autobutler.org is the
// pre-rename repo, frozen at v0.12.0 and publishing "autobutler_*" assets that
// never match ConstructArchiveName, and quark.autobutler.org is the website and
// publishes no releases at all.
var DefaultUpdateSources = []*UpdateSource{
	NewUpdateSource(
		UpdateSourceKindAzure,
		"quarkrelease",
		"releases/quark",
	),
	NewUpdateSource(
		UpdateSourceKindGithub,
		"autobutler-org",
		"quark",
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

// String renders a source the way an operator needs to read it in an error.
// The default struct formatting produced "&{github autobutler-org quark.org }",
// which is where a 404 on a nonexistent repository went to hide (#1610).
func (s *UpdateSource) String() string {
	if s == nil {
		return "<nil update source>"
	}
	switch s.Kind {
	case UpdateSourceKindGithub:
		return fmt.Sprintf("github repository %s/%s", s.Account, s.Path)
	case UpdateSourceKindAzure:
		return fmt.Sprintf("azure storage account %s, path %s", s.Account, s.Path)
	default:
		return fmt.Sprintf("unknown source kind %q (%s/%s)", s.Kind, s.Account, s.Path)
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
