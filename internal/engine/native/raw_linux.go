//go:build linux

package native

import "os/exec"

// GetRawDevicePath returns the raw block device path on Linux.
func GetRawDevicePath(diskPath string) string {
	return diskPath
}

// PostRawWriteCleanup re-probes partition table on Linux.
func PostRawWriteCleanup(diskPath string) {
	_ = exec.Command("partprobe", diskPath).Run()
}
