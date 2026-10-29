//go:build darwin

package engine

import (
	"fmt"
	"os/exec"
	"time"
)

// UnmountDevice unmounts all partitions and active mounts associated with the device on macOS.
func UnmountDevice(devicePath string) error {
	cmd := exec.Command("diskutil", "unmountDisk", "force", devicePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("diskutil unmountDisk failed: %s (%w)", string(out), err)
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}
