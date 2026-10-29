//go:build windows

package native

// GetRawDevicePath returns the raw block device path on Windows.
func GetRawDevicePath(diskPath string) string {
	return diskPath
}

// PostRawWriteCleanup handles cache invalidation on Windows.
func PostRawWriteCleanup(diskPath string) {}
