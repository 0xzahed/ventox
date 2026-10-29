//go:build windows

package safety

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

func checkPlatformSystemDisk(dev domain.BlockDevice) (bool, string, error) {
	sysDrive := os.Getenv("SystemDrive")
	if sysDrive == "" {
		sysDrive = "C:"
	}
	for _, mp := range dev.Mountpoints {
		if strings.EqualFold(strings.TrimRight(mp, "\\/"), strings.TrimRight(sysDrive, "\\/")) {
			return true, fmt.Sprintf("device contains Windows SystemDrive (%s)", sysDrive), nil
		}
	}
	return false, "", nil
}
