package smb

import (
	"runtime"
	"strings"
	"testing"
)

// TestShareBlock_ContainsShareName verifies that the generated Samba share
// block includes the share name section header.
func TestShareBlock_ContainsShareName(t *testing.T) {
	block := shareBlock("/data/files")
	if !strings.Contains(block, "["+shareName+"]") {
		t.Errorf("shareBlock missing [%s] header; got:\n%s", shareName, block)
	}
}

// TestShareBlock_ContainsPath verifies the injected path appears in the config block.
func TestShareBlock_ContainsPath(t *testing.T) {
	path := "/var/lib/quark/files"
	block := shareBlock(path)
	if !strings.Contains(block, "path = "+path) {
		t.Errorf("shareBlock missing path %q; got:\n%s", path, block)
	}
}

// TestShareBlock_Writable verifies the share is configured writable.
func TestShareBlock_Writable(t *testing.T) {
	block := shareBlock("/tmp/test")
	if !strings.Contains(block, "writeable = yes") {
		t.Errorf("shareBlock should set writeable=yes; got:\n%s", block)
	}
}

// TestShareBlock_NotPublic verifies the share is configured as private (auth required).
func TestShareBlock_NotPublic(t *testing.T) {
	block := shareBlock("/tmp/test")
	if !strings.Contains(block, "public = no") {
		t.Errorf("shareBlock should set public=no; got:\n%s", block)
	}
}

// TestIsLinux_MatchesRuntime verifies IsLinux() matches runtime.GOOS.
func TestIsLinux_MatchesRuntime(t *testing.T) {
	want := runtime.GOOS == "linux"
	if got := IsLinux(); got != want {
		t.Errorf("IsLinux() = %v; want %v (runtime.GOOS=%q)", got, want, runtime.GOOS)
	}
}

// TestIsConfigured_NotConfiguredWhenFileMissing verifies IsConfigured returns
// false when the smb.conf file does not exist.
func TestIsConfigured_NotConfiguredWhenFileMissing(t *testing.T) {
	// Only safe to run where the real smb.conf doesn't exist.
	// The real smbConfigPath (/etc/samba/smb.conf) is only present when Samba
	// is installed. In CI it should not exist, so this is a no-op pass there.
	// On a development machine with Samba installed, skip to avoid false negatives.
	if IsInstalled() {
		t.Skip("Samba installed — smb.conf may exist; skipping missing-file test")
	}
	if IsConfigured() {
		t.Error("IsConfigured() = true but smb.conf should not exist without Samba installed")
	}
}
