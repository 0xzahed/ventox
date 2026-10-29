//go:build darwin

package native

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

// SetupDualPartitions partitions the drive into FAT32 BOOT and ExFAT INSTALL on macOS.
func SetupDualPartitions(diskPath, isoPath string) (bootDir, installDir, isoDir string, cleanup func(), err error) {
	ui.PrintInfo("Unmounting target drive partitions on macOS...")
	if err := engine.UnmountDevice(diskPath); err != nil {
		return "", "", "", nil, err
	}

	ui.PrintInfo("Creating GPT dual-partition layout via diskutil (BOOT FAT32 + INSTALL ExFAT)...")
	// diskutil partitionDisk <disk> GPT MS-DOS BOOT 1024M ExFAT INSTALL 0b
	cmd := exec.Command("diskutil", "partitionDisk", diskPath, "GPT", "MS-DOS", "BOOT", "1024M", "ExFAT", "INSTALL", "0b")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", "", "", nil, fmt.Errorf("diskutil partitionDisk failed: %s (%w)", string(out), err)
	}

	time.Sleep(2 * time.Second)

	bootMount := "/Volumes/BOOT"
	installMount := "/Volumes/INSTALL"

	isoMount, err := os.MkdirTemp("", "ventox_iso_*")
	if err != nil {
		return "", "", "", nil, err
	}

	cleanupFn := func() {
		ui.PrintInfo("Unmounting and detaching drives on macOS...")
		_ = exec.Command("hdiutil", "detach", isoMount, "-force").Run()
		_ = exec.Command("diskutil", "unmountDisk", "force", diskPath).Run()
		_ = os.Remove(isoMount)
	}

	ui.PrintInfo("Mounting ISO image via hdiutil...")
	attachCmd := exec.Command("hdiutil", "attach", "-readonly", "-nobrowse", "-mountpoint", isoMount, isoPath)
	if out, err := attachCmd.CombinedOutput(); err != nil {
		cleanupFn()
		return "", "", "", nil, fmt.Errorf("hdiutil attach failed: %s (%w)", string(out), err)
	}

	// Verify macOS volumes exist
	if _, err := os.Stat(bootMount); err != nil {
		cleanupFn()
		return "", "", "", nil, fmt.Errorf("BOOT volume missing at %s: %w", bootMount, err)
	}
	if _, err := os.Stat(installMount); err != nil {
		cleanupFn()
		return "", "", "", nil, fmt.Errorf("INSTALL volume missing at %s: %w", installMount, err)
	}

	return bootMount, installMount, isoMount, cleanupFn, nil
}
