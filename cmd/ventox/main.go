package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/0xzahed/ventox/internal/copier"
	"github.com/0xzahed/ventox/internal/detector"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

func main() {
	// Auto re-execute with sudo if not root
	if os.Geteuid() != 0 {
		ui.Banner()
		ui.PrintInfo("Root permissions required to partition USB and write bootloader.")
		ui.PrintInfo("Requesting administrative privileges (sudo)...")
		fmt.Println()

		exe, err := os.Executable()
		if err != nil {
			exe = os.Args[0]
		}

		cmd := exec.Command("sudo", append([]string{exe}, os.Args[1:]...)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	ui.Banner()

	// 1. Detect and Select USB Drive
	ui.PrintStep(1, 4, "Detecting USB Drives")
	disks, err := detector.DetectUSBDisks()
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to query lsblk: %v", err))
	}

	selectedDisk, err := ui.SelectUSB(disks)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Error selecting USB: %v", err))
		os.Exit(1)
	}
	ui.PrintSuccess(fmt.Sprintf("Target USB Selected: %s", selectedDisk.Path))

	// 2. Detect and Select ISO File
	ui.PrintStep(2, 4, "Scanning for ISO Files")
	isos, err := detector.FindISOFallbacks()
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Scanning directories returned warning: %v", err))
	}

	selectedISO, err := ui.SelectISO(isos)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Error selecting ISO: %v", err))
		os.Exit(1)
	}
	ui.PrintSuccess(fmt.Sprintf("ISO File Selected: %s", selectedISO))

	// 3. Confirmation Dialog
	if !ui.ConfirmErase(selectedDisk.Path, selectedISO) {
		ui.PrintWarning("Operation cancelled by user. No changes were made.")
		os.Exit(0)
	}

	// Setup cleanup on interruption (Ctrl+C)
	var activeMount string
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println()
		ui.PrintWarning("Interrupt received! Cleaning up temporary mounts...")
		if activeMount != "" {
			_ = exec.Command("umount", "-l", activeMount).Run()
			_ = os.Remove(activeMount)
		}
		os.Exit(1)
	}()

	// 4. Locate & Install Ventoy
	ui.PrintStep(3, 4, "Installing UEFI/GPT Bootloader (Ventoy)")
	ventoyScript, err := engine.LocateVentoy()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to locate Ventoy: %v", err))
		os.Exit(1)
	}

	ui.PrintInfo(fmt.Sprintf("Using Ventoy engine: %s", ventoyScript))
	if err := engine.InstallVentoy(ventoyScript, selectedDisk.Path); err != nil {
		ui.PrintError(fmt.Sprintf("Bootloader setup failed: %v", err))
		os.Exit(1)
	}
	ui.PrintSuccess("Ventoy bootloader installed successfully.")

	// 5. Mount Partition and Copy ISO
	ui.PrintStep(4, 4, "Copying ISO Image to Bootable USB")
	partPath := engine.GetVentoyDataPartition(selectedDisk.Path)

	mountDir, err := os.MkdirTemp("", "bootmaker_mnt_*")
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create temporary mount directory: %v", err))
		os.Exit(1)
	}
	activeMount = mountDir
	defer func() {
		if activeMount != "" {
			_ = exec.Command("umount", "-l", activeMount).Run()
			_ = os.Remove(activeMount)
		}
	}()

	ui.PrintInfo(fmt.Sprintf("Mounting %s to %s...", partPath, mountDir))
	time.Sleep(1 * time.Second)
	if err := exec.Command("mount", partPath, mountDir).Run(); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to mount partition: %v", err))
		os.Exit(1)
	}

	destPath := filepath.Join(mountDir, filepath.Base(selectedISO))
	ui.PrintInfo(fmt.Sprintf("Writing: %s", filepath.Base(selectedISO)))

	if err := copier.CopyWithProgress(selectedISO, destPath); err != nil {
		ui.PrintError(fmt.Sprintf("Copy failed: %v", err))
		os.Exit(1)
	}

	ui.PrintInfo("Unmounting USB drive cleanly...")
	_ = exec.Command("umount", mountDir).Run()
	_ = os.Remove(mountDir)
	activeMount = ""

	// Complete Banner
	fmt.Println()
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println(ui.Green(ui.Bold("  🎉 CONGRATULATIONS! BOOTABLE USB IS READY! 🎉               ")))
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Printf(" • Target USB:   %s\n", ui.Bold(selectedDisk.Path))
	fmt.Printf(" • Boot Engine:  Ventoy (GPT / UEFI & Legacy BIOS)\n")
	fmt.Printf(" • Loaded ISO:   %s\n", ui.Bold(filepath.Base(selectedISO)))
	fmt.Println(" • You can boot any PC using this USB (Press F12, F11, or Esc at startup)")
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println()
}
