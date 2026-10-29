//go:build windows

package native

import "fmt"

// SetupDualPartitions partitions the drive into FAT32 BOOT and NTFS INSTALL on Windows.
func SetupDualPartitions(diskPath, isoPath string) (bootDir, installDir, isoDir string, cleanup func(), err error) {
	// Execute diskpart script on Windows
	return "", "", "", func() {}, fmt.Errorf("dual partition creation via diskpart is pending Windows CLI integration")
}
