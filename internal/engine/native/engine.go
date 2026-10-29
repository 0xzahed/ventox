package native

import (
	"fmt"

	"github.com/0xzahed/ventox/internal/ui"
)

// DeployNativeBoot flashes the selected ISO onto target USB using the native engine
// (Dual-partition UEFI for Windows with bypass, or raw block write for Linux)
// This operates completely independently of Ventoy!
func DeployNativeBoot(diskPath, isoPath string) error {
	ui.PrintInfo("Analyzing ISO image architecture (native inspector)...")
	analysis, err := AnalyzeISO(isoPath)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("ISO inspect warning: %v. Proceeding with block write.", err))
	}

	if analysis != nil && analysis.IsWindows {
		ui.PrintSuccess(fmt.Sprintf("Detected: Windows OS Installer (Install WIM: %s)", analysis.WIMPath))
		ui.PrintInfo("Applying Rufus-Style Dual-Partition Architecture (FAT32 BOOT + NTFS INSTALL)...")
		return WriteWindowsUSB(diskPath, isoPath, analysis)
	}

	ui.PrintSuccess("Detected: Linux Hybrid OS Image")
	ui.PrintInfo("Applying Native Direct Block Streaming (DD Mode)...")
	return WriteLinuxUSB(diskPath, isoPath)
}
