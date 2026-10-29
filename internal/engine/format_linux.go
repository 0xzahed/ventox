//go:build linux

package engine

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/ui"
)

// FormatDisk formats or resets the target device according to the chosen mode on Linux
func FormatDisk(devicePath string, mode FormatMode, label string) error {
	if label == "" {
		label = "VENTOX"
	}
	label = strings.ToUpper(strings.TrimSpace(label))
	if len(label) > 11 && (mode == ModeCleanFAT32 || mode == ModeCleanExFAT) {
		label = label[:11]
	}

	ui.PrintInfo("Unmounting device partitions...")
	if err := UnmountDevice(devicePath); err != nil {
		return err
	}

	switch mode {
	case ModeVentoyGPT:
		script, err := LocateVentoy()
		if err != nil {
			return err
		}
		ui.PrintInfo("Writing Ventoy GPT/UEFI bootloader...")
		cmd := exec.Command(script, "-I", "-g", devicePath)
		return cmd.Run()

	case ModeVentoyMBR:
		script, err := LocateVentoy()
		if err != nil {
			return err
		}
		ui.PrintInfo("Writing Ventoy MBR/Legacy bootloader...")
		cmd := exec.Command(script, "-I", devicePath)
		return cmd.Run()

	case ModeCleanExFAT, ModeCleanFAT32, ModeCleanNTFS, ModeCleanEXT4:
		ui.PrintInfo("Wiping old partition tables and filesystem signatures...")
		_ = exec.Command("wipefs", "-a", devicePath).Run()

		ui.PrintInfo("Creating new partition table...")
		partType := "gpt"
		if mode == ModeCleanFAT32 {
			partType = "msdos"
		}
		mklabelCmd := exec.Command("parted", "-s", devicePath, "mklabel", partType)
		if err := mklabelCmd.Run(); err != nil {
			return fmt.Errorf("parted mklabel failed: %w", err)
		}

		mkpartCmd := exec.Command("parted", "-s", devicePath, "mkpart", "primary", "1MiB", "100%")
		if err := mkpartCmd.Run(); err != nil {
			return fmt.Errorf("parted mkpart failed: %w", err)
		}

		time.Sleep(2 * time.Second)
		_ = exec.Command("partprobe", devicePath).Run()
		time.Sleep(1 * time.Second)

		part := GetVentoyDataPartition(devicePath)

		ui.PrintInfo(fmt.Sprintf("Formatting %s with label '%s'...", part, label))

		var mkfsCmd *exec.Cmd
		switch mode {
		case ModeCleanExFAT:
			mkfsCmd = exec.Command("mkfs.exfat", "-n", label, part)
		case ModeCleanFAT32:
			mkfsCmd = exec.Command("mkfs.vfat", "-F", "32", "-n", label, part)
		case ModeCleanNTFS:
			mkfsCmd = exec.Command("mkfs.ntfs", "-f", "-L", label, part)
		case ModeCleanEXT4:
			mkfsCmd = exec.Command("mkfs.ext4", "-F", "-L", label, part)
		}

		if out, err := mkfsCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("format error: %s (%w)", string(out), err)
		}

		_ = exec.Command("sync").Run()
		return nil

	default:
		return fmt.Errorf("unknown format mode: %d", mode)
	}
}
