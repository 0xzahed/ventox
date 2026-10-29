package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/0xzahed/ventox/internal/detector"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/engine/native"
	"github.com/0xzahed/ventox/internal/ui"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		cmd := strings.ToLower(args[0])
		switch cmd {
		case "version", "-v", "--version":
			ui.PrintVersion()
			return
		case "help", "-h", "--help":
			ui.PrintHelp()
			return
		case "list", "ls":
			runList()
			return
		case "format", "wipe":
			runFormat()
			return
		default:
			// If an unknown flag or option is passed
			if strings.HasPrefix(cmd, "-") {
				ui.PrintError(fmt.Sprintf("Unknown flag: %s", cmd))
				ui.PrintHelp()
				os.Exit(1)
			}
		}
	}

	// Default: Run interactive boot creation wizard
	runWizard()
}

func ensureRoot() {
	if os.Geteuid() != 0 {
		ui.Banner()
		ui.PrintInfo("Administrative privileges (sudo) required for direct disk operations.")
		ui.PrintInfo("Re-executing with sudo...")
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
}

func runList() {
	ui.Banner()
	fmt.Println(ui.Cyan(ui.Bold("--- CONNECTED USB STORAGE DEVICES ---")))
	disks, err := detector.DetectUSBDisks()
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to query USB disks: %v", err))
	} else if len(disks) == 0 {
		ui.PrintWarning("No USB storage drives currently connected.")
	} else {
		for i, d := range disks {
			fmt.Printf("  %s %s\n", ui.Cyan(fmt.Sprintf("[%d]", i+1)), d.DisplayString())
		}
	}

	fmt.Println()
	fmt.Println(ui.Cyan(ui.Bold("--- DETECTED ISO DISK IMAGES ---")))
	isos, err := detector.FindISOFallbacks()
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Error scanning ISOs: %v", err))
	} else if len(isos) == 0 {
		ui.PrintWarning("No .iso files found in user Downloads or Desktop.")
	} else {
		for i, iso := range isos {
			fmt.Printf("  %s %s %s\n     %s\n",
				ui.Cyan(fmt.Sprintf("[%d]", i+1)),
				ui.Bold(iso.Name),
				ui.Cyan(fmt.Sprintf("(%s)", iso.FormattedSize)),
				ui.Gray(iso.Path),
			)
		}
	}
	fmt.Println()
}

func runFormat() {
	ensureRoot()
	ui.Banner()

	// 1. Detect and Select USB Drive
	ui.PrintStep(1, 3, "Select USB Drive to Format")
	disks, err := detector.DetectUSBDisks()
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to query lsblk: %v", err))
	}

	selectedDisk, err := ui.SelectUSB(disks)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Error selecting USB: %v", err))
		os.Exit(1)
	}

	// 2. Select Format Option
	ui.PrintStep(2, 3, "Select Target Filesystem & Partition Style")
	engineOpts := engine.GetFormatOptions()
	var uiChoices []ui.FormatChoice
	for _, o := range engineOpts {
		uiChoices = append(uiChoices, ui.FormatChoice{
			ID:          int(o.Mode),
			Title:       o.Title,
			Description: o.Description,
		})
	}

	selectedChoice, err := ui.SelectFormatMode(uiChoices)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Error choosing format mode: %v", err))
		os.Exit(1)
	}

	label, _ := ui.Prompt("Enter Volume Label (e.g. VENTOX, MY_USB)", "VENTOX")

	// 3. Confirm
	ui.PrintStep(3, 3, "Confirm Drive Format")
	if !ui.ConfirmFormat(selectedDisk.Path, selectedChoice.Title, label) {
		ui.PrintWarning("Format cancelled by user. No changes were made.")
		os.Exit(0)
	}

	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Formatting %s as %s...", selectedDisk.Path, selectedChoice.Title))

	if err := engine.FormatDisk(selectedDisk.Path, engine.FormatMode(selectedChoice.ID), label); err != nil {
		ui.PrintError(fmt.Sprintf("Format failed: %v", err))
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println(ui.Green(ui.Bold("  ✔ FORMAT COMPLETE! USB DRIVE IS READY TO USE!               ")))
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Printf(" • Target USB:   %s\n", ui.Bold(selectedDisk.Path))
	fmt.Printf(" • Format Mode:  %s\n", ui.Bold(selectedChoice.Title))
	fmt.Printf(" • Volume Label: %s\n", ui.Bold(label))
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println()
}

func runWizard() {
	ensureRoot()
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

	// 4. Deploy Native Engine (Rufus-style Dual Partition UEFI + Win11 Bypass / Linux DD)
	ui.PrintStep(3, 3, "Deploying Bootable Media (Native Rufus-Style Engine)")
	if err := native.DeployNativeBoot(selectedDisk.Path, selectedISO); err != nil {
		ui.PrintError(fmt.Sprintf("Native deployment failed: %v", err))
		os.Exit(1)
	}

	// Complete Banner
	fmt.Println()
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println(ui.Green(ui.Bold("  🎉 CONGRATULATIONS! BOOTABLE USB IS READY! 🎉               ")))
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Printf(" • Target USB:   %s\n", ui.Bold(selectedDisk.Path))
	fmt.Printf(" • Boot Engine:  Ventox Native (Rufus-Style Dual Partition UEFI / DD)\n")
	fmt.Printf(" • Loaded ISO:   %s\n", ui.Bold(filepath.Base(selectedISO)))
	fmt.Println(" • Ready to boot on any modern PC (Press F12, F11, or Esc at boot)")
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println()
}
