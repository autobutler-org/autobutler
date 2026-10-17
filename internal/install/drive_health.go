// cspell:words smartctl nvme mmcblk libexec

package install

import "github.com/autobutler-org/quark/pkg/util/storageutil"

// driveHealthHelperContent is the root helper behind storageutil.HelperPath.
// sudoers allows it any argument, so the script accepts only a whole-disk
// path and then runs a fixed smartctl read. A USB bridge that hides SMART is
// tried again with SATA passthrough. smartctl is not installed here: when it
// is missing the script exits 127 and the device list shows Unknown.
const driveHealthHelperContent = `#!/bin/sh
# Written by quark install. Changes are overwritten on the next install.
# Reads SMART for one whole disk and prints smartctl's JSON. The device path
# is checked here because the sudoers entry does not constrain arguments.
set -eu
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH

if [ "$#" -ne 1 ]; then
	echo "usage: $0 /dev/<disk>" >&2
	exit 2
fi

disk=$1
case $disk in
/dev/sd[a-z]|/dev/sd[a-z][a-z]|/dev/vd[a-z]|/dev/xvd[a-z]|/dev/nvme[0-9]n[0-9]|/dev/nvme[0-9][0-9]n[0-9]|/dev/mmcblk[0-9]|/dev/mmcblk[0-9][0-9])
	;;
*)
	echo "invalid device" >&2
	exit 2
	;;
esac

if ! command -v smartctl >/dev/null 2>&1; then
	echo "health tool is not installed" >&2
	exit 127
fi

# A non-zero smartctl status can still carry JSON. Prefer that JSON. Try SATA
# passthrough when the first read has no SMART data, which is how many USB
# bridges behave.
out=$(smartctl -j "$disk" 2>/dev/null || true)
case $out in
*'"smart_status"'*|*'nvme_smart_health_information_log'*)
	printf '%s\n' "$out"
	exit 0
	;;
esac

out=$(smartctl -j -d sat "$disk" 2>/dev/null || true)
case $out in
*'"smart_status"'*|*'nvme_smart_health_information_log'*)
	printf '%s\n' "$out"
	exit 0
	;;
esac

echo "drive health could not be read" >&2
exit 1
`

// The path constant lives in storageutil so the service and this installer
// name the same file. Referenced so a rename fails the build.
var _ = storageutil.HelperPath
