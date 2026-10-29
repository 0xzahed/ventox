//go:build linux

package native

import (
	"fmt"
	"os/exec"
)

// MountLoopISO attaches and mounts the ISO read-only using loop device on Linux.
func MountLoopISO(isoPath, tempMount string) (func(), error) {
	cmd := exec.Command("mount", "-o", "loop,ro", isoPath, tempMount)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("loop mount failed: %s (%w)", string(out), err)
	}

	cleanup := func() {
		_ = exec.Command("umount", "-l", tempMount).Run()
	}

	return cleanup, nil
}
