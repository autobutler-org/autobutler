//go:build darwin
// +build darwin

package usbutil

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func (p *partition) MountCommand(mountTargetPath string) *exec.Cmd {
	return exec.Command("mount", p.path, mountTargetPath)
}

func (p *partition) MountPath() (string, error) {
	// Check mount output for this device on macOS
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run mount: %w", err)
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, p.path) {
			// Extract mount point from mount output
			// Format: device on mountpoint (...)
			parts := strings.SplitN(line, " on ", 2)
			if len(parts) == 2 {
				mountPoint := strings.Split(parts[1], " ")[0]
				return mountPoint, nil
			}
		}
	}
	return "", fmt.Errorf("partition %s is not mountable", p.path)
}

func (p *partition) Path() string {
	return p.path
}

func (p *partition) SizeBytes() (int, error) {
	// On macOS, use diskutil to get partition size
	cmd := exec.Command("diskutil", "info", "-plist", p.path)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get partition info: %w", err)
	}

	// Parse plist output to get TotalSize
	// Simple parsing: look for <key>TotalSize</key><integer>VALUE</integer>
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if strings.Contains(line, "<key>TotalSize</key>") && i+1 < len(lines) {
			nextLine := lines[i+1]
			if strings.Contains(nextLine, "<integer>") {
				start := strings.Index(nextLine, ">") + 1
				end := strings.Index(nextLine, "</integer>")
				if start > 0 && end > start {
					sizeStr := nextLine[start:end]
					size, err := strconv.Atoi(sizeStr)
					if err != nil {
						return 0, fmt.Errorf("failed to parse partition size: %w", err)
					}
					return size, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("could not determine partition size for %s", p.path)
}
