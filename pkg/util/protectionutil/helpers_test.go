// cspell:ignore mmcblk nvme protectionutil sda vda

package protectionutil

import "testing"

func TestWholeDiskKey(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{path: "/dev/disk3s5", want: "disk3"},
		{path: "/dev/disk3s1", want: "disk3"},
		{path: "/dev/disk3", want: "disk3"},
		{path: "/dev/disk10s2", want: "disk10"},
		{path: "/dev/sda1", want: "sda"},
		{path: "/dev/sda2", want: "sda"},
		{path: "/dev/sda", want: "sda"},
		{path: "/dev/nvme0n1p1", want: "nvme0n1"},
		{path: "/dev/nvme0n1p2", want: "nvme0n1"},
		{path: "/dev/nvme0n1", want: "nvme0n1"},
		{path: "/dev/mmcblk0p1", want: "mmcblk0"},
		{path: "/dev/mmcblk0p2", want: "mmcblk0"},
		{path: "/dev/vda1", want: "vda"},
		{path: "", want: ""},
	}
	for _, tc := range cases {
		if got := wholeDiskKey(tc.path); got != tc.want {
			t.Errorf("wholeDiskKey(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
