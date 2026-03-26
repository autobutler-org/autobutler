package versionutil

import (
	"testing"
)

func TestNewVersion(t *testing.T) {
	// Save and restore package-level Semver
	origSemver := Semver
	defer func() { Semver = origSemver }()

	Semver = "v1.2.3"
	v := NewVersion("abc1234", "go1.21.0", "2024-01-15")

	if v.Semver != "v1.2.3" {
		t.Errorf("Semver = %q, want %q", v.Semver, "v1.2.3")
	}
	if v.GitCommit != "abc1234" {
		t.Errorf("GitCommit = %q, want %q", v.GitCommit, "abc1234")
	}
	if v.GoVersion != "go1.21.0" {
		t.Errorf("GoVersion = %q, want %q", v.GoVersion, "go1.21.0")
	}
	if v.BuildDate != "2024-01-15" {
		t.Errorf("BuildDate = %q, want %q", v.BuildDate, "2024-01-15")
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name      string
		semver    string
		gitCommit string
		buildDate string
		want      string
	}{
		{
			name:      "NoSemver and NoCommit",
			semver:    NoSemver,
			gitCommit: NoCommit,
			buildDate: "",
			want:      NoCommit,
		},
		{
			name:      "NoSemver with real commit",
			semver:    NoSemver,
			gitCommit: "abc1234",
			buildDate: "",
			want:      "abc1234",
		},
		{
			name:      "NoSemver with commit and build date",
			semver:    NoSemver,
			gitCommit: "abc1234",
			buildDate: "2024-01-15",
			want:      "abc1234 from 2024-01-15",
		},
		{
			name:      "real semver with NoCommit",
			semver:    "v1.2.3",
			gitCommit: NoCommit,
			buildDate: "",
			want:      "v1.2.3",
		},
		{
			name:      "real semver with real commit",
			semver:    "v1.2.3",
			gitCommit: "abc1234",
			buildDate: "",
			want:      "v1.2.3@abc1234",
		},
		{
			name:      "real semver with commit and build date",
			semver:    "v1.2.3",
			gitCommit: "abc1234",
			buildDate: "2024-01-15",
			want:      "v1.2.3@abc1234 from 2024-01-15",
		},
		{
			name:      "real semver with NoCommit and build date",
			semver:    "v1.2.3",
			gitCommit: NoCommit,
			buildDate: "2024-01-15",
			want:      "v1.2.3 from 2024-01-15",
		},
	}

	origSemver := Semver
	defer func() { Semver = origSemver }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Semver = tt.semver
			v := NewVersion(tt.gitCommit, "go1.21.0", tt.buildDate)
			got := v.VersionString()
			if got != tt.want {
				t.Errorf("VersionString() = %q, want %q", got, tt.want)
			}
		})
	}
}
