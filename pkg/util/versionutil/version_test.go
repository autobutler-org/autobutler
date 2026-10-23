package versionutil

import (
	"testing"
)

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
