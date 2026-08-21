package versionutil_test

import (
	"runtime"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/versionutil"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       versionutil.Version
		v2       versionutil.Version
		expected int
	}{
		{
			name:     "equal versions",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 0,
		},
		{
			name:     "v1 less than v2 - major version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "v2.2.3"},
			expected: -1,
		},
		{
			name:     "v1 greater than v2 - major version",
			v1:       versionutil.Version{Semver: "v2.2.3"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 1,
		},
		{
			name:     "v1 less than v2 - minor version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "v1.5.3"},
			expected: -1,
		},
		{
			name:     "v1 greater than v2 - minor version",
			v1:       versionutil.Version{Semver: "v1.5.3"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 1,
		},
		{
			name:     "v1 less than v2 - patch version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "v1.2.7"},
			expected: -1,
		},
		{
			name:     "v1 greater than v2 - patch version",
			v1:       versionutil.Version{Semver: "v1.2.7"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 1,
		},
		{
			name:     "versions without v prefix",
			v1:       versionutil.Version{Semver: "1.2.3"},
			v2:       versionutil.Version{Semver: "1.2.3"},
			expected: 0,
		},
		{
			name:     "v1 has NoSemver",
			v1:       versionutil.Version{Semver: versionutil.NoSemver},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 2,
		},
		{
			name:     "v2 has NoSemver",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: versionutil.NoSemver},
			expected: 2,
		},
		{
			name:     "both have NoSemver",
			v1:       versionutil.Version{Semver: versionutil.NoSemver},
			v2:       versionutil.Version{Semver: versionutil.NoSemver},
			expected: 2,
		},
		{
			name:     "v1.0.0 vs v1.0.1",
			v1:       versionutil.Version{Semver: "v1.0.0"},
			v2:       versionutil.Version{Semver: "v1.0.1"},
			expected: -1,
		},
		{
			name:     "v0.0.1 vs v0.0.0",
			v1:       versionutil.Version{Semver: "v0.0.1"},
			v2:       versionutil.Version{Semver: "v0.0.0"},
			expected: 1,
		},
		{
			name:     "dev version vs semver",
			v1:       versionutil.Version{Semver: "dev-abc1234"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 2,
		},
		{
			name:     "semver vs dev version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "dev-xyz5678"},
			expected: 2,
		},
		{
			name:     "two dev versions",
			v1:       versionutil.Version{Semver: "dev-abc1234"},
			v2:       versionutil.Version{Semver: "dev-xyz5678"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionutil.CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%v, %v) = %d; want %d", tt.v1.Semver, tt.v2.Semver, result, tt.expected)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	// GetVersion reads from debug.ReadBuildInfo() which includes build-time metadata
	version := versionutil.GetVersion()

	// Version should never be nil
	if version == nil {
		t.Fatal("GetVersion() returned nil")
	}

	// GoVersion should always be set to runtime.Version()
	if version.GoVersion != runtime.Version() {
		t.Errorf("GetVersion().GoVersion = %q; want %q", version.GoVersion, runtime.Version())
	}

	// Semver should be set from the package variable
	if version.Semver == "" {
		t.Error("GetVersion().Semver is empty")
	}

	// GitCommit should be set (either from build info or NoCommit)
	if version.GitCommit == "" {
		t.Error("GetVersion().GitCommit is empty")
	}

	// BuildDate may or may not be set depending on build context
	// but it should never cause a panic
	_ = version.BuildDate
}

func TestGetVersion_Fields(t *testing.T) {
	version := versionutil.GetVersion()

	// Test that the version can generate a version string without panic
	versionStr := version.VersionString()
	if versionStr == "" {
		t.Error("VersionString() returned empty string")
	}

	// Test that all fields are accessible
	t.Logf("GitCommit: %s", version.GitCommit)
	t.Logf("Semver: %s", version.Semver)
	t.Logf("GoVersion: %s", version.GoVersion)
	t.Logf("BuildDate: %s", version.BuildDate)
	t.Logf("VersionString: %s", versionStr)
}

func TestCompareVersions_InvalidVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       versionutil.Version
		v2       versionutil.Version
		expected int
	}{
		{
			name:     "v1 has invalid major version",
			v1:       versionutil.Version{Semver: "vX.2.3"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 0,
		},
		{
			name:     "v2 has invalid major version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "vY.2.3"},
			expected: 0,
		},
		{
			name:     "v1 has invalid minor version",
			v1:       versionutil.Version{Semver: "v1.X.3"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 0,
		},
		{
			name:     "v2 has invalid minor version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "v1.Y.3"},
			expected: 0,
		},
		{
			name:     "v1 has invalid patch version",
			v1:       versionutil.Version{Semver: "v1.2.X"},
			v2:       versionutil.Version{Semver: "v1.2.3"},
			expected: 0,
		},
		{
			name:     "v2 has invalid patch version",
			v1:       versionutil.Version{Semver: "v1.2.3"},
			v2:       versionutil.Version{Semver: "v1.2.Y"},
			expected: 0,
		},
		{
			name:     "both have invalid versions",
			v1:       versionutil.Version{Semver: "vX.Y.Z"},
			v2:       versionutil.Version{Semver: "vA.B.C"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionutil.CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%v, %v) = %d; want %d", tt.v1.Semver, tt.v2.Semver, result, tt.expected)
			}
		})
	}
}
