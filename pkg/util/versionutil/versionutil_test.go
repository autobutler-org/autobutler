package versionutil_test

import (
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

func TestCompareVersions_InvalidVersions(t *testing.T) {
	result := versionutil.CompareVersions(
		versionutil.Version{Semver: "vX.Y.Z"},
		versionutil.Version{Semver: "vA.B.C"},
	)
	if result != 0 {
		t.Errorf("CompareVersions(vX.Y.Z, vA.B.C) = %d; want 0", result)
	}
}

// A binary built from a source tarball, in a container without .git, or with
// -buildvcs=false gets no vcs.revision from the toolchain, and used to report
// NOCOMMIT with no way to say which build it was. The Makefile stamps the
// commit with -ldflags so the value survives that.
func TestGetVersion_PrefersStampedCommit(t *testing.T) {
	const stamped = "a1b2c3d"

	original := versionutil.GitCommit
	t.Cleanup(func() { versionutil.GitCommit = original })

	versionutil.GitCommit = stamped
	if got := versionutil.GetVersion().GitCommit; got != stamped {
		t.Errorf("GetVersion().GitCommit = %q; want the stamped %q", got, stamped)
	}

	// Unstamped, the build info still names the build when it can. This test
	// binary is built from a checkout, so a real revision is there to find.
	versionutil.GitCommit = versionutil.NoCommit
	if got := versionutil.GetVersion().GitCommit; got == "" {
		t.Error("GetVersion().GitCommit is empty with no stamp; want the build info's revision")
	}
}
