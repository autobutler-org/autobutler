package updateutil

import (
	"autobutler/pkg/util/githubutil"
	"strings"
)

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
