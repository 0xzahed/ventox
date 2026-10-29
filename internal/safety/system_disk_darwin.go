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
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Part of Whole:") || strings.HasPrefix(l, "Device Node:") {
				parts := strings.Split(l, ":")
				if len(parts) > 1 {
					node := strings.TrimSpace(parts[1])
					if strings.Contains(dev.Path, node) || strings.Contains(node, dev.Name) {
						return true, fmt.Sprintf("device is the primary macOS startup disk (%s)", node), nil
					}
				}
			}
		}
	}

	return false, "", nil
}
