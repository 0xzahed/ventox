//go:build darwin

package native

import (
	"fmt"
	"os/exec"
)

// MountLoopISO attaches and mounts the ISO read-only using hdiutil on macOS.
func MountLoopISO(isoPath, tempMount string) (func(), error) {
	cmd := exec.Command("hdiutil", "attach", "-readonly", "-nobrowse", "-mountpoint", tempMount, isoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("hdiutil attach failed: %s (%w)", string(out), err)
	}

	cleanup := func() {
		_ = exec.Command("hdiutil", "detach", tempMount, "-force").Run()
	}

	return cleanup, nil
}
