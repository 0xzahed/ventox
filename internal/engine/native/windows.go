package native

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/copier"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

// WriteWindowsUSB creates a Rufus-style dual-partition UEFI/GPT bootable drive
// Partition 1 (FAT32, 1024 MiB): UEFI Boot files + WinPE boot.wim + autounattend.xml
// Partition 2 (NTFS, Remaining): Full installation media + large install.wim (>4GB)
func WriteWindowsUSB(diskPath, isoPath string, analysis *ISOAnalysis) error {
	ui.PrintInfo("Unmounting any active partitions on target drive...")
	if err := engine.UnmountDevice(diskPath); err != nil {
		return err
	}

	// 1. Wipe old signatures
	ui.PrintInfo("Wiping old partition tables (wipefs)...")
	_ = exec.Command("wipefs", "-a", diskPath).Run()

	// 2. Create GPT layout
	ui.PrintInfo("Creating GPT dual-partition layout (Rufus/Microsoft standard)...")
	if err := exec.Command("parted", "-s", diskPath, "mklabel", "gpt").Run(); err != nil {
		return fmt.Errorf("parted mklabel gpt failed: %w", err)
	}

	// Partition 1: 1024 MiB FAT32 for EFI bootloader and WinPE
	if err := exec.Command("parted", "-s", diskPath, "mkpart", "BOOT", "fat32", "1MiB", "1025MiB").Run(); err != nil {
		return fmt.Errorf("failed to create BOOT partition: %w", err)
	}
	_ = exec.Command("parted", "-s", diskPath, "set", "1", "esp", "on").Run()
	_ = exec.Command("parted", "-s", diskPath, "set", "1", "boot", "on").Run()

	// Partition 2: NTFS for full Windows installation (handles files > 4GB)
	if err := exec.Command("parted", "-s", diskPath, "mkpart", "INSTALL", "ntfs", "1025MiB", "100%").Run(); err != nil {
		return fmt.Errorf("failed to create INSTALL partition: %w", err)
	}

	time.Sleep(2 * time.Second)
	_ = exec.Command("partprobe", diskPath).Run()
	time.Sleep(1 * time.Second)

	part1 := engine.GetVentoyDataPartition(diskPath)
	part2 := strings.TrimSuffix(part1, "1") + "2"

	// 3. Format partitions
	ui.PrintInfo(fmt.Sprintf("Formatting Partition 1 (%s) as FAT32 [Label: BOOT]...", part1))
	if out, err := exec.Command("mkfs.vfat", "-F", "32", "-n", "BOOT", part1).CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs.vfat error: %s (%w)", string(out), err)
	}

	ui.PrintInfo(fmt.Sprintf("Formatting Partition 2 (%s) as NTFS [Label: INSTALL]...", part2))
	if out, err := exec.Command("mkfs.ntfs", "-f", "-L", "INSTALL", part2).CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs.ntfs error: %s (%w)", string(out), err)
	}

	// 4. Temporary Mounts
	isoMount, err := os.MkdirTemp("", "ventox_iso_*")
	if err != nil {
		return err
	}
	bootMount, err := os.MkdirTemp("", "ventox_boot_*")
	if err != nil {
		return err
	}
	installMount, err := os.MkdirTemp("", "ventox_install_*")
	if err != nil {
		return err
	}

	defer func() {
		ui.PrintInfo("Cleaning up temporary mount points...")
		_ = exec.Command("umount", "-l", bootMount).Run()
		_ = exec.Command("umount", "-l", installMount).Run()
		_ = exec.Command("umount", "-l", isoMount).Run()
		_ = os.Remove(bootMount)
		_ = os.Remove(installMount)
		_ = os.Remove(isoMount)
	}()

	ui.PrintInfo("Mounting ISO image via loopback...")
	if out, err := exec.Command("mount", "-o", "loop,ro", isoPath, isoMount).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount ISO: %s (%w)", string(out), err)
	}

	ui.PrintInfo("Mounting target USB partitions...")
	if out, err := exec.Command("mount", part1, bootMount).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount BOOT partition: %s (%w)", string(out), err)
	}
	if out, err := exec.Command("mount", part2, installMount).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to mount INSTALL partition: %s (%w)", string(out), err)
	}

	// 5. Populate Partition 1 (BOOT, FAT32)
	ui.PrintInfo("Writing UEFI bootloader and WinPE files to Partition 1 (BOOT)...")
	essentialBootItems := []string{"efi", "EFI", "boot", "BOOT", "bootmgr", "bootmgr.efi"}
	for _, item := range essentialBootItems {
		src := filepath.Join(isoMount, item)
		if _, err := os.Stat(src); err == nil {
			dest := filepath.Join(bootMount, item)
			_ = copyRecursive(src, dest, nil)
		}
	}

	// Copy sources/boot.wim into Partition 1
	srcBootWim := filepath.Join(isoMount, "sources", "boot.wim")
	if _, err := os.Stat(srcBootWim); err == nil {
		_ = os.MkdirAll(filepath.Join(bootMount, "sources"), 0755)
		_ = copyFile(srcBootWim, filepath.Join(bootMount, "sources", "boot.wim"))
	}

	// Inject Windows 11 Bypass (TPM, Secure Boot, RAM, Online Account)
	ui.PrintInfo("Injecting Windows 11 Hardware Bypass (TPM/SecureBoot/OOBE BypassNRO)...")
	bypassXML := GenerateAutoUnattendXML()
	_ = os.WriteFile(filepath.Join(bootMount, "autounattend.xml"), []byte(bypassXML), 0644)
	_ = os.WriteFile(filepath.Join(installMount, "autounattend.xml"), []byte(bypassXML), 0644)

	// 6. Populate Partition 2 (INSTALL, NTFS)
	ui.PrintInfo("Copying Windows installation files to Partition 2 (INSTALL)...")

	// Copy everything except large install.wim first
	largeWIMRel := analysis.WIMPath
	skipMap := map[string]bool{
		strings.ToLower(largeWIMRel): true,
	}

	if err := copyRecursive(isoMount, installMount, skipMap); err != nil {
		return fmt.Errorf("error copying installation files: %w", err)
	}

	// Now stream large install.wim with real-time live progress bar
	srcWIM := filepath.Join(isoMount, largeWIMRel)
	destWIM := filepath.Join(installMount, largeWIMRel)
	_ = os.MkdirAll(filepath.Dir(destWIM), 0755)

	ui.PrintInfo(fmt.Sprintf("Streaming main OS image (%s)...", filepath.Base(srcWIM)))
	if err := copier.CopyWithProgress(srcWIM, destWIM); err != nil {
		return fmt.Errorf("failed to stream install.wim: %w", err)
	}

	ui.PrintInfo("Flushing disk cache (sync)...")
	_ = exec.Command("sync").Run()

	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyRecursive(srcDir, destDir string, skipRelPaths map[string]bool) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil || rel == "." {
			return nil
		}

		if skipRelPaths != nil && skipRelPaths[strings.ToLower(rel)] {
			return nil
		}

		destPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Don't copy files over 1GB here (handled separately with progress)
		if info.Size() > 1024*1024*1024 {
			return nil
		}

		return copyFile(path, destPath)
	})
}
