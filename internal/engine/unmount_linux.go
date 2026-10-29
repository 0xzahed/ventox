//go:build linux

package engine

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// UnmountDevice unmounts all partitions and active mounts associated with the device on Linux.
func UnmountDevice(devicePath string) error {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return fmt.Errorf("unable to read /proc/mounts: %w", err)
	}
	defer file.Close()

	var mountpoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			dev := fields[0]
			mp := fields[1]
			if strings.HasPrefix(dev, devicePath) {
				mountpoints = append(mountpoints, mp)
			}
		}
	}

	for _, mp := range mountpoints {
		_ = exec.Command("umount", "-l", mp).Run()
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}
