// cspell:words smartctl nvme mmcblk libexec

package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/storageutil"
)

func TestDriveHealthHelperOutsideSelfUpdatableDir(t *testing.T) {
	if !filepath.IsAbs(storageutil.HelperPath) {
		t.Fatalf("helper path %s is not absolute", storageutil.HelperPath)
	}
	if strings.HasPrefix(storageutil.HelperPath, serviceBinDir+"/") || strings.HasPrefix(storageutil.HelperPath, serviceDataDir+"/") {
		t.Errorf("helper %s sits somewhere the service can write", storageutil.HelperPath)
	}
	if !strings.Contains(driveHealthHelperContent, "smartctl") || !strings.Contains(driveHealthHelperContent, "health tool is not installed") {
		t.Fatal("helper does not run smartctl or report a missing tool")
	}
}

func TestDriveHealthHelperRejectsBadDevices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drive-health")
	if err := os.WriteFile(path, []byte(driveHealthHelperContent), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := [][]string{
		{},
		{"/dev/sda", "extra"},
		{"/tmp/evil"},
		{"/dev/sda1"},
		{"sda"},
		{"/dev/../etc/passwd"},
		{"/dev/disk3s5"},
	}
	for _, args := range cases {
		cmd := exec.Command(path, args...)
		err := cmd.Run()
		var exitErr *exec.ExitError
		if !errorsAsExit(err, &exitErr) || exitErr.ExitCode() != 2 {
			t.Errorf("drive-health %q exit = %v, want 2", args, err)
		}
	}
}

func errorsAsExit(err error, target **exec.ExitError) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	*target = exitErr
	return true
}
