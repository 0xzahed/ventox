//go:build windows

package native

import (
	"fmt"
	"os/exec"
)

// MountLoopISO mounts the ISO using PowerShell Mount-DiskImage on Windows.
func MountLoopISO(isoPath, tempMount string) (func(), error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Mount-DiskImage -ImagePath '%s'", isoPath))
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("windows Mount-DiskImage failed: %w", err)
	}

	cleanup := func() {
		_ = exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Dismount-DiskImage -ImagePath '%s'", isoPath)).Run()
	}

	return cleanup, nil
}
