package updateutil

import (
	"archive/tar"
	"autobutler/pkg/util/githubutil"
	"autobutler/pkg/util/versionutil"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
)

func IsDevelopmentVersion(version string) bool {
	if matched, err := regexp.MatchString("^.*-", version); err != nil || matched {
		return true
	}
	if matched, err := regexp.MatchString("-.*$", version); err != nil || matched {
		return true
	}
	return false
}

// ListPossibleUpdatesResult contains the result of listing possible updates
type ListPossibleUpdatesResult struct {
	Releases []githubutil.GitHubRelease
}

// ListPossibleUpdates retrieves all available releases that are newer than the current version
func ListPossibleUpdates(org string, repo string, allVersions bool) (*ListPossibleUpdatesResult, error) {
	releases, err := githubutil.FetchGitHubReleases(org, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	if allVersions {
		return &ListPossibleUpdatesResult{
			Releases: releases,
		}, nil
	}

	currentVersion := versionutil.GetVersion()
	if currentVersion.Semver == "" {
		return &ListPossibleUpdatesResult{
			Releases: releases,
		}, nil
	}

	var possibleUpdates []githubutil.GitHubRelease
	for _, release := range releases {
		if IsDevelopmentVersion(release.TagName) {
			continue
		}
		comparison := versionutil.CompareVersions(
			versionutil.Version{
				Semver: release.TagName,
			},
			*currentVersion,
		)
		if comparison > 0 {
			possibleUpdates = append(possibleUpdates, release)
		}
	}

	return &ListPossibleUpdatesResult{
		Releases: possibleUpdates,
	}, nil
}

func GetLatestVersion(org string, repo string) (*githubutil.GitHubRelease, error) {
	release, err := githubutil.FetchLatestGitHubRelease(org, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	return release, nil
}

type UpdateParams struct {
	Version       string `json:"version"`
	BaseUpdateURL string `json:"baseUpdateURL,omitempty"`
	Force         bool   `json:"force,omitempty"`
}

// Update downloads and installs a new version of the application
func Update(params UpdateParams) error {
	version := params.Version
	if version == "" {
		return fmt.Errorf("version cannot be empty")
	}

	_, err := backupSelf()
	if err != nil {
		return fmt.Errorf("failed to copy current binary: %w", err)
	}

	baseUrl := params.BaseUpdateURL
	if baseUrl == "" {
		baseUrl = "https://github.com/autobutler-org/autobutler.org/releases/download"
	}

	goos := fmt.Sprintf("%s%s", strings.ToUpper(string(runtime.GOOS[0])), string(runtime.GOOS[1:]))
	url := fmt.Sprintf("%s/%s/autobutler_%s_%s.tar.gz", baseUrl, version, goos, runtime.GOARCH)
	fmt.Println("Downloading update from", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download update from %s: %s", url, resp.Status)
	}

	if err := replaceSelf(resp.Body); err != nil {
		return fmt.Errorf("failed to replace self with update from %s: %w", url, err)
	}

	fmt.Println("Update successful.")
	return nil
}

const binaryName = "autobutler"

var backupName = fmt.Sprintf("%s_backup", binaryName)
var extractedName = fmt.Sprintf("%s_extracted", binaryName)

func backupSelf() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	tmpFile, err := os.CreateTemp("", backupName)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	src, err := os.Open(execPath)
	if err != nil {
		return "", fmt.Errorf("failed to open current binary: %w", err)
	}
	defer src.Close()

	if _, err := src.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to seek in current binary: %w", err)
	}
	if _, err := tmpFile.ReadFrom(src); err != nil {
		return "", fmt.Errorf("failed to copy binary to temp: %w", err)
	}
	return tmpFile.Name(), nil
}

func replaceSelf(body io.Reader) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	tmpFile, err := os.CreateTemp("", "autobutler_update_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file for update: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.ReadFrom(body); err != nil {
		return fmt.Errorf("failed to write update to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	// Rewind the temp file to the beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek in temp file: %w", err)
	}

	gzReader, err := gzip.NewReader(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var binFile *os.File
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}
		if header.Typeflag == tar.TypeReg && (header.Name == binaryName || header.Name == fmt.Sprintf("./%s", binaryName)) {
			binFile, err = os.CreateTemp("", extractedName)
			if err != nil {
				return fmt.Errorf("failed to create temp file for extracted binary: %w", err)
			}
			if _, err := io.Copy(binFile, tarReader); err != nil {
				return fmt.Errorf("failed to extract binary from tar: %w", err)
			}
			if err := binFile.Sync(); err != nil {
				return fmt.Errorf("failed to sync extracted binary: %w", err)
			}
			break
		}
	}
	if binFile == nil {
		return fmt.Errorf("binary not found in archive")
	}
	defer os.Remove(binFile.Name())
	defer binFile.Close()

	// Make the extracted binary executable
	if err := os.Chmod(binFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	// Close the file before operations
	binFile.Close()

	// Create a temporary file in the same directory as the target executable
	// This ensures we're on the same filesystem for atomic rename
	execDir := execPath[:strings.LastIndex(execPath, "/")]
	tmpNew, err := os.CreateTemp(execDir, ".autobutler_new_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file in target directory: %w", err)
	}
	tmpNewPath := tmpNew.Name()
	defer os.Remove(tmpNewPath)

	// Copy the new binary to the target filesystem
	src, err := os.Open(binFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open extracted binary: %w", err)
	}
	defer src.Close()

	if _, err := io.Copy(tmpNew, src); err != nil {
		tmpNew.Close()
		return fmt.Errorf("failed to copy new binary: %w", err)
	}

	if err := tmpNew.Sync(); err != nil {
		tmpNew.Close()
		return fmt.Errorf("failed to sync new binary: %w", err)
	}
	tmpNew.Close()

	// Set executable permissions on the new file
	if err := os.Chmod(tmpNewPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions on new binary: %w", err)
	}

	// Atomically rename the new binary over the old one
	// This works even while the old binary is running
	if err := os.Rename(tmpNewPath, execPath); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	return nil
}
