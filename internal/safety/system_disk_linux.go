//go:build linux

package safety

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

func checkPlatformSystemDisk(dev domain.BlockDevice) (bool, string, error) {
	// 1. Check /proc/swaps
	if swapFile, err := os.Open("/proc/swaps"); err == nil {
		scanner := bufio.NewScanner(swapFile)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) > 0 && strings.HasPrefix(fields[0], "/dev/") {
				swapDev := fields[0]
				if strings.HasPrefix(swapDev, dev.Path) {
					swapFile.Close()
					return true, fmt.Sprintf("device contains active swap space: %s", swapDev), nil
				}
			}
		}
		swapFile.Close()
	}

	// 2. Identify root device from /proc/mounts
	if mountFile, err := os.Open("/proc/mounts"); err == nil {
		scanner := bufio.NewScanner(mountFile)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[1] == "/" {
				rootDev := fields[0] // e.g. /dev/nvme0n1p4 or /dev/sda1
				// Resolve symlinks if any
				if resolved, err := filepath.EvalSymlinks(rootDev); err == nil {
					rootDev = resolved
				}
				if strings.HasPrefix(rootDev, dev.Path) {
					mountFile.Close()
					return true, fmt.Sprintf("device is the root filesystem drive (%s)", rootDev), nil
				}
			}
		}
		mountFile.Close()
	}

	return false, "", nil
}
