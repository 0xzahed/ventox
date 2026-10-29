//go:build darwin

package native

import (
	"os/exec"
	"strings"
)

// GetRawDevicePath returns the unbuffered raw character device path on macOS (/dev/rdiskX).
func GetRawDevicePath(diskPath string) string {
	if strings.HasPrefix(diskPath, "/dev/disk") {
		return strings.Replace(diskPath, "/dev/disk", "/dev/rdisk", 1)
	}
	return diskPath
}

// PostRawWriteCleanup informs macOS kernel of changes or ejects disk.
func PostRawWriteCleanup(diskPath string) {
	_ = exec.Command("diskutil", "unmountDisk", "force", diskPath).Run()
}
