//go:build linux

package native

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

// SetupDualPartitions partitions the drive into FAT32 BOOT and NTFS INSTALL on Linux.
func SetupDualPartitions(diskPath, isoPath string) (bootDir, installDir, isoDir string, cleanup func(), err error) {
	ui.PrintInfo("Unmounting target drive partitions...")
	if err := engine.UnmountDevice(diskPath); err != nil {
		return "", "", "", nil, err
	}

	ui.PrintInfo("Wiping old partition tables (wipefs)...")
	_ = exec.Command("wipefs", "-a", diskPath).Run()

	ui.PrintInfo("Creating GPT dual-partition layout (Rufus/Microsoft standard)...")
	if err := exec.Command("parted", "-s", diskPath, "mklabel", "gpt").Run(); err != nil {
		return "", "", "", nil, fmt.Errorf("parted mklabel gpt failed: %w", err)
	}

	if err := exec.Command("parted", "-s", diskPath, "mkpart", "BOOT", "fat32", "1MiB", "1025MiB").Run(); err != nil {
		return "", "", "", nil, fmt.Errorf("failed to create BOOT partition: %w", err)
	}
	_ = exec.Command("parted", "-s", diskPath, "set", "1", "esp", "on").Run()
	_ = exec.Command("parted", "-s", diskPath, "set", "1", "boot", "on").Run()

	if err := exec.Command("parted", "-s", diskPath, "mkpart", "INSTALL", "ntfs", "1025MiB", "100%").Run(); err != nil {
		return "", "", "", nil, fmt.Errorf("failed to create INSTALL partition: %w", err)
	}

	time.Sleep(2 * time.Second)
	_ = exec.Command("partprobe", diskPath).Run()
	time.Sleep(1 * time.Second)

	part1 := engine.GetVentoyDataPartition(diskPath)
	part2 := strings.TrimSuffix(part1, "1") + "2"

	ui.PrintInfo(fmt.Sprintf("Formatting Partition 1 (%s) as FAT32 [Label: BOOT]...", part1))
	if out, err := exec.Command("mkfs.vfat", "-F", "32", "-n", "BOOT", part1).CombinedOutput(); err != nil {
		return "", "", "", nil, fmt.Errorf("mkfs.vfat error: %s (%w)", string(out), err)
	}

	ui.PrintInfo(fmt.Sprintf("Formatting Partition 2 (%s) as NTFS [Label: INSTALL]...", part2))
	if out, err := exec.Command("mkfs.ntfs", "-f", "-L", "INSTALL", part2).CombinedOutput(); err != nil {
		return "", "", "", nil, fmt.Errorf("mkfs.ntfs error: %s (%w)", string(out), err)
	}

	isoMount, err := os.MkdirTemp("", "ventox_iso_*")
	if err != nil {
		return "", "", "", nil, err
	}
	bootMount, err := os.MkdirTemp("", "ventox_boot_*")
	if err != nil {
		return "", "", "", nil, err
	}
	installMount, err := os.MkdirTemp("", "ventox_install_*")
	if err != nil {
		return "", "", "", nil, err
	}

	cleanupFn := func() {
		ui.PrintInfo("Cleaning up temporary mount points...")
		_ = exec.Command("umount", "-l", bootMount).Run()
		_ = exec.Command("umount", "-l", installMount).Run()
		_ = exec.Command("umount", "-l", isoMount).Run()
		_ = os.Remove(bootMount)
		_ = os.Remove(installMount)
		_ = os.Remove(isoMount)
	}

	ui.PrintInfo("Mounting ISO image via loopback...")
	if out, err := exec.Command("mount", "-o", "loop,ro", isoPath, isoMount).CombinedOutput(); err != nil {
		cleanupFn()
		return "", "", "", nil, fmt.Errorf("failed to mount ISO: %s (%w)", string(out), err)
	}

	ui.PrintInfo("Mounting target USB partitions...")
	if out, err := exec.Command("mount", part1, bootMount).CombinedOutput(); err != nil {
		cleanupFn()
		return "", "", "", nil, fmt.Errorf("failed to mount BOOT partition: %s (%w)", string(out), err)
	}
	if out, err := exec.Command("mount", part2, installMount).CombinedOutput(); err != nil {
		cleanupFn()
		return "", "", "", nil, fmt.Errorf("failed to mount INSTALL partition: %s (%w)", string(out), err)
	}

	return bootMount, installMount, isoMount, cleanupFn, nil
}
