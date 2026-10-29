package safety

import (
	"fmt"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

var criticalSystemMounts = []string{
	"/",
	"/boot",
	"/boot/efi",
	"/home",
	"/usr",
	"/var",
	"/etc",
	"C:",
	"C:\\",
	"/System/Volumes/Data",
}

// IsSystemDisk evaluates whether the device hosts critical OS filesystems, swap, or root volumes.
func IsSystemDisk(dev domain.BlockDevice) (bool, string, error) {
	// 1. Direct mountpoints on whole disk
	for _, mp := range dev.Mountpoints {
		for _, crit := range criticalSystemMounts {
			if strings.EqualFold(strings.TrimRight(mp, "/\\"), strings.TrimRight(crit, "/\\")) {
				return true, fmt.Sprintf("device directly mounts critical system path: %s", mp), nil
			}
		}
	}

	// 2. Mountpoints on any child partitions
	for _, part := range dev.Partitions {
		for _, mp := range part.Mountpoints {
			for _, crit := range criticalSystemMounts {
				if strings.EqualFold(strings.TrimRight(mp, "/\\"), strings.TrimRight(crit, "/\\")) {
					return true, fmt.Sprintf("partition %s mounts critical system path: %s", part.Path, mp), nil
				}
			}
		}
	}

	// 3. Platform-specific low-level checks (root disk resolution, swap)
	return checkPlatformSystemDisk(dev)
}
