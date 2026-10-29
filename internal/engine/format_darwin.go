//go:build darwin

package engine

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/0xzahed/ventox/internal/ui"
)

// FormatDisk formats or resets the target device according to the chosen mode on macOS
func FormatDisk(devicePath string, mode FormatMode, label string) error {
	if label == "" {
		label = "VENTOX"
	}
	label = strings.ToUpper(strings.TrimSpace(label))
	if len(label) > 11 && (mode == ModeCleanFAT32 || mode == ModeCleanExFAT) {
		label = label[:11]
	}

	ui.PrintInfo("Unmounting device partitions on macOS...")
	if err := UnmountDevice(devicePath); err != nil {
		return err
	}

	var cmd *exec.Cmd

	switch mode {
	case ModeVentoyGPT:
		ui.PrintInfo("Creating Dual-Partition UEFI Layout on macOS (BOOT FAT32 + INSTALL ExFAT)...")
		cmd = exec.Command("diskutil", "partitionDisk", devicePath, "GPT", "MS-DOS", "BOOT", "1024M", "ExFAT", "INSTALL", "0b")

	case ModeVentoyMBR:
		ui.PrintInfo("Creating MBR Partition Layout on macOS...")
		cmd = exec.Command("diskutil", "partitionDisk", devicePath, "MBR", "MS-DOS", label, "0b")

	case ModeCleanExFAT:
		ui.PrintInfo(fmt.Sprintf("Erasing %s as ExFAT (GPT)...", devicePath))
		cmd = exec.Command("diskutil", "eraseDisk", "ExFAT", label, "GPT", devicePath)

	case ModeCleanFAT32:
		ui.PrintInfo(fmt.Sprintf("Erasing %s as FAT32 / MS-DOS (MBR)...", devicePath))
		cmd = exec.Command("diskutil", "eraseDisk", "MS-DOS", label, "MBR", devicePath)

	case ModeCleanNTFS:
		ui.PrintInfo(fmt.Sprintf("Erasing %s as ExFAT for Windows compatibility...", devicePath))
		cmd = exec.Command("diskutil", "eraseDisk", "ExFAT", label, "GPT", devicePath)

	case ModeCleanEXT4:
		ui.PrintInfo(fmt.Sprintf("Erasing %s as APFS (Apple Native)...", devicePath))
		cmd = exec.Command("diskutil", "eraseDisk", "APFS", label, "GPT", devicePath)

	default:
		return fmt.Errorf("unknown format mode: %d", mode)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("macOS format failed: %s (%w)", string(out), err)
	}

	_ = exec.Command("sync").Run()
	return nil
}
