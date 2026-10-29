package native

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0xzahed/ventox/internal/copier"
	"github.com/0xzahed/ventox/internal/ui"
)

// WriteWindowsUSB creates a Rufus-style dual-partition UEFI/GPT bootable drive
// Partition 1 (FAT32, 1024 MiB): UEFI Boot files + WinPE boot.wim + autounattend.xml
// Partition 2 (NTFS/ExFAT, Remaining): Full installation media + large install.wim (>4GB)
func WriteWindowsUSB(diskPath, isoPath string, analysis *ISOAnalysis) error {
	bootMount, installMount, isoMount, cleanup, err := SetupDualPartitions(diskPath, isoPath)
	if err != nil {
		return err
	}
	defer cleanup()

	// 1. Populate Partition 1 (BOOT, FAT32)
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

	// 2. Populate Partition 2 (INSTALL, NTFS/ExFAT)
	ui.PrintInfo("Copying Windows installation files to Partition 2 (INSTALL)...")

	largeWIMRel := analysis.WIMPath
	skipMap := map[string]bool{
		strings.ToLower(largeWIMRel): true,
	}

	if err := copyRecursive(isoMount, installMount, skipMap); err != nil {
		return fmt.Errorf("error copying installation files: %w", err)
	}

	// Stream large install.wim with real-time live progress bar
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
