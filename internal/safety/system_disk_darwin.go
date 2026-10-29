//go:build darwin

package safety

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

func checkPlatformSystemDisk(dev domain.BlockDevice) (bool, string, error) {
	// Query root filesystem disk node on macOS
	cmd := exec.Command("diskutil", "info", "/")
	out, err := cmd.Output()
	if err != nil {
		return false, "", nil
	}

	var rootDiskID string
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "Part of Whole:") {
			parts := strings.Split(l, ":")
			if len(parts) > 1 {
				rootDiskID = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if rootDiskID == "" {
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Device Identifier:") {
				parts := strings.Split(l, ":")
				if len(parts) > 1 {
					id := strings.TrimSpace(parts[1])
					// If disk1s1s1, extract parent disk1
					if idx := strings.Index(id, "s"); idx != -1 {
						rootDiskID = id[:idx]
					} else {
						rootDiskID = id
					}
					break
				}
			}
		}
	}

	if rootDiskID != "" {
		if dev.Name != "" && (dev.Name == rootDiskID || strings.HasPrefix(dev.Name, rootDiskID+"s")) {
			return true, fmt.Sprintf("device is the primary macOS startup disk (%s)", rootDiskID), nil
		}
		if dev.Path != "" {
			cleanPath := strings.TrimPrefix(dev.Path, "/dev/")
			cleanPath = strings.TrimPrefix(cleanPath, "r")
			if cleanPath == rootDiskID || strings.HasPrefix(cleanPath, rootDiskID+"s") {
				return true, fmt.Sprintf("device is the primary macOS startup disk (%s)", rootDiskID), nil
			}
		}
	}

	return false, "", nil
}
