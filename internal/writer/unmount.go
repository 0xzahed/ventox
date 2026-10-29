package writer

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

// UnmountDevice safely unmounts all active mounts for the target device across Linux, macOS, and Windows.
func UnmountDevice(ctx context.Context, dev domain.BlockDevice) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.CommandContext(ctx, "diskutil", "unmountDisk", "force", dev.Path)
		if out, err := cmd.CombinedOutput(); err != nil {
			// If already not mounted, diskutil may exit non-zero
			if !strings.Contains(string(out), "not currently mounted") {
				return fmt.Errorf("failed to unmount disk %s: %s (%w)", dev.Path, strings.TrimSpace(string(out)), err)
			}
		}
		return nil

	case "linux":
		// Collect all mounts from disk and partitions
		var mounts []string
		mounts = append(mounts, dev.Mountpoints...)
		for _, part := range dev.Partitions {
			mounts = append(mounts, part.Mountpoints...)
		}

		for _, mp := range mounts {
			if mp == "" {
				continue
			}
			cmd := exec.CommandContext(ctx, "umount", "-f", mp)
			if out, err := cmd.CombinedOutput(); err != nil {
				if !strings.Contains(string(out), "not mounted") {
					return fmt.Errorf("failed to unmount %s: %s (%w)", mp, strings.TrimSpace(string(out)), err)
				}
			}
		}
		return nil

	case "windows":
		// On Windows, volume locking and dismounting occurs during write handle open
		return nil

	default:
		return nil
	}
}
