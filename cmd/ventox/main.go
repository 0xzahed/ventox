package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/0xzahed/ventox/internal/device"
	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/iso"
	"github.com/0xzahed/ventox/internal/ui"
)

type cliConfig struct {
	command      string
	dryRun       bool
	verify       bool
	bypassWin11  bool
	devicePath   string
	isoPath      string
	fs           string
	label        string
	autoConfirm  bool
	jsonOutput   bool
}

func main() {
	// Fast path for help and version without root check
	for _, a := range os.Args[1:] {
		if a == "-h" || a == "--help" || a == "help" {
			ui.PrintHelp()
			return
		}
		if a == "-v" || a == "--version" || a == "version" {
			ui.PrintVersion()
			return
		}
	}

	cfg := parseFlags()

	switch cfg.command {
	case "list", "ls":
		runList(cfg)
		return
	case "format", "wipe":
		runFormat(cfg)
		return
	default:
		runBootCreator(cfg)
	}
}

func parseFlags() cliConfig {
	fs := flag.NewFlagSet("ventox", flag.ContinueOnError)
	fs.Usage = ui.PrintHelp

	dryRun := fs.Bool("dry-run", false, "Simulate operation without writing")
	verify := fs.Bool("verify", false, "Verify target data checksum after writing")
	bypassWin11 := fs.Bool("bypass-win11", true, "Enable Windows 11 hardware bypasses")
	devicePath := fs.String("device", "", "Target USB device path")
	isoPath := fs.String("iso", "", "Source disk image / ISO path")
	targetFS := fs.String("fs", "", "Target filesystem (exfat, fat32, ntfs, ext4, apfs)")
	label := fs.String("label", "", "Filesystem volume label")
	autoConfirm := fs.Bool("yes", false, "Skip confirmation prompts")
	fs.BoolVar(autoConfirm, "y", false, "Skip confirmation prompts")
	jsonOutput := fs.Bool("json", false, "Structured JSON output")

	// Check first argument for subcommand
	var command string
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = strings.ToLower(args[0])
		args = args[1:]
	}

	if err := fs.Parse(args); err == flag.ErrHelp {
		os.Exit(0)
	}

	return cliConfig{
		command:     command,
		dryRun:      *dryRun,
		verify:      *verify,
		bypassWin11: *bypassWin11,
		devicePath:  *devicePath,
		isoPath:     *isoPath,
		fs:          *targetFS,
		label:       *label,
		autoConfirm: *autoConfirm,
		jsonOutput:  *jsonOutput,
	}
}

func ensureRoot(dryRun bool) {
	if dryRun || runtime.GOOS == "windows" {
		return
	}
	if os.Geteuid() != 0 {
		ui.Banner()
		ui.PrintInfo("Administrative privileges (sudo) required for direct block device access.")
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

func runList(cfg cliConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	devices, err := device.DefaultProvider.ListDevices(ctx)
	if err != nil && !cfg.jsonOutput {
		ui.PrintWarning(fmt.Sprintf("Device scan returned warning: %v", err))
	}

	images, err := iso.DiscoverImages(ctx)
	if err != nil && !cfg.jsonOutput {
		ui.PrintWarning(fmt.Sprintf("Image discovery warning: %v", err))
	}

	if cfg.jsonOutput {
		out := map[string]interface{}{
			"devices": devices,
			"images":  images,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return
	}

	ui.Banner()
	fmt.Println(ui.Cyan(ui.Bold("--- CONNECTED USB STORAGE DEVICES ---")))
	var usbCount int
	for _, d := range devices {
		if d.IsRemovable || d.BusType == "USB" {
			usbCount++
			fmt.Printf("  %s %s\n", ui.Cyan(fmt.Sprintf("[%d]", usbCount)), d.DisplayString())
		}
	}
	if usbCount == 0 {
		ui.PrintWarning("No USB storage drives currently connected.")
	}

	fmt.Println()
	fmt.Println(ui.Cyan(ui.Bold("--- DETECTED ISO DISK IMAGES ---")))
	if len(images) == 0 {
		ui.PrintWarning("No .iso or .img files found in user Downloads or Desktop.")
	} else {
		for i, img := range images {
			fmt.Printf("  %s %s %s\n     %s\n",
				ui.Cyan(fmt.Sprintf("[%d]", i+1)),
				ui.Bold(img.Name),
				ui.Cyan(fmt.Sprintf("(%s)", img.FormattedSize())),
				ui.Gray(img.Path),
			)
		}
	}
	fmt.Println()
}

func runFormat(cfg cliConfig) {
	ensureRoot(cfg.dryRun)

	ctx := context.Background()

	// 1. Select Device
	var targetDev domain.BlockDevice
	if cfg.devicePath != "" {
		dev, err := device.DefaultProvider.GetDevice(ctx, cfg.devicePath)
		if err != nil {
			// Construct direct block device info
			targetDev = domain.BlockDevice{Path: cfg.devicePath, RawPath: cfg.devicePath}
		} else {
			targetDev = dev
		}
	} else {
		ui.Banner()
		ui.PrintStep(1, 3, "Select USB Drive to Format")
		devs, _ := device.DefaultProvider.ListDevices(ctx)
		var usbList []domain.BlockDevice
		for _, d := range devs {
			if d.IsRemovable || d.BusType == "USB" {
				usbList = append(usbList, d)
			}
		}
		selected, err := selectDeviceInteractive(usbList)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Device selection failed: %v", err))
			os.Exit(1)
		}
		targetDev = selected
	}

	// 2. Select Filesystem
	targetFS := domain.TargetFilesystem(cfg.fs)
	if targetFS == "" {
		ui.PrintStep(2, 3, "Select Target Filesystem")
		fmt.Println("Available filesystems: exFAT (Recommended), FAT32, NTFS, ext4")
		fsChoice, _ := ui.Prompt("Enter filesystem [exfat/fat32/ntfs/ext4]", "exfat")
		switch strings.ToLower(strings.TrimSpace(fsChoice)) {
		case "fat32":
			targetFS = domain.FSFAT32
		case "ntfs":
			targetFS = domain.FSNTFS
		case "ext4":
			targetFS = domain.FSEXT4
		default:
			targetFS = domain.FSExFAT
		}
	}

	label := cfg.label
	if label == "" {
		lbl, _ := ui.Prompt("Enter Volume Label", "VENTOX")
		label = lbl
	}

	// 3. Create Plan
	plan, err := engine.CreatePlan(engine.PlanOptions{
		Target:          targetDev,
		Source:          nil,
		RequestedFS:     targetFS,
		RequestedScheme: domain.SchemeGPT,
		VolumeLabel:     label,
		DryRun:          cfg.dryRun,
	})
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to generate format plan: %v", err))
		os.Exit(1)
	}

	// 4. Confirm
	if !cfg.autoConfirm && !ui.ConfirmPlan(plan) {
		ui.PrintWarning("Operation cancelled by user.")
		return
	}

	// 5. Execute
	ui.PrintInfo(fmt.Sprintf("Executing format plan (%s as %s)...", targetDev.Path, targetFS))
	execContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	executor := engine.NewExecutor()
	journal, err := executor.Execute(execContext, plan, func(p engine.ProgressEvent) {
		ui.RenderProgressBar(p.BytesProcessed, p.TotalBytes, p.Percent, p.SpeedBytesSec, p.ETA, p.StepName)
	})
	ui.ClearProgressBar()

	if err != nil {
		ui.PrintError(fmt.Sprintf("Format failed: %v", err))
		os.Exit(1)
	}

	if cfg.jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(journal)
		return
	}

	ui.PrintSuccess(fmt.Sprintf("Drive %s successfully formatted as %s (Label: %s)", targetDev.Path, targetFS, label))
}

func runBootCreator(cfg cliConfig) {
	ensureRoot(cfg.dryRun)

	ctx := context.Background()

	// 1. Select Target USB Device
	var targetDev domain.BlockDevice
	if cfg.devicePath != "" {
		dev, err := device.DefaultProvider.GetDevice(ctx, cfg.devicePath)
		if err != nil {
			targetDev = domain.BlockDevice{
				Path:        cfg.devicePath,
				RawPath:     device.DefaultProvider.GetRawPath(cfg.devicePath),
				IsRemovable: true,
			}
		} else {
			targetDev = dev
		}
	} else {
		ui.Banner()
		ui.PrintStep(1, 4, "Detecting USB Drives")
		devs, _ := device.DefaultProvider.ListDevices(ctx)
		var usbList []domain.BlockDevice
		for _, d := range devs {
			if d.IsRemovable || d.BusType == "USB" {
				usbList = append(usbList, d)
			}
		}
		selected, err := selectDeviceInteractive(usbList)
		if err != nil {
			ui.PrintError(fmt.Sprintf("USB selection error: %v", err))
			os.Exit(1)
		}
		targetDev = selected
	}
	if !cfg.jsonOutput {
		ui.PrintSuccess(fmt.Sprintf("Target USB Selected: %s", targetDev.DisplayString()))
	}

	// 2. Select Source ISO Image
	var sourceMeta domain.ImageMetadata
	if cfg.isoPath != "" {
		meta, err := iso.InspectImage(cfg.isoPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to inspect ISO: %v", err))
			os.Exit(1)
		}
		sourceMeta = meta
	} else {
		ui.PrintStep(2, 4, "Scanning for ISO Files")
		images, _ := iso.DiscoverImages(ctx)
		selectedPath, err := selectImageInteractive(images)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Image selection error: %v", err))
			os.Exit(1)
		}
		meta, err := iso.InspectImage(selectedPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to inspect selected ISO: %v", err))
			os.Exit(1)
		}
		sourceMeta = meta
	}
	if !cfg.jsonOutput {
		ui.PrintSuccess(fmt.Sprintf("Image Loaded: %s (%s, %s)", sourceMeta.Name, sourceMeta.FormattedSize(), sourceMeta.Classification))
	}

	// 3. Create Architecture Plan
	plan, err := engine.CreatePlan(engine.PlanOptions{
		Target:           targetDev,
		Source:           &sourceMeta,
		BypassWindows11:  cfg.bypassWin11,
		VerifyAfterWrite: cfg.verify,
		DryRun:           cfg.dryRun,
	})
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create operation plan: %v", err))
		os.Exit(1)
	}

	// 4. Confirmation Dialog
	if !cfg.autoConfirm && !ui.ConfirmPlan(plan) {
		ui.PrintWarning("Operation cancelled by user. No changes were made.")
		return
	}

	// 5. Execute Plan
	if !cfg.jsonOutput {
		ui.PrintStep(3, 3, fmt.Sprintf("Executing Native Engine (%s)", plan.Mode))
	}

	execContext, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println()
		ui.PrintWarning("Interrupt received! Cancelling active disk operation...")
		cancel()
	}()

	executor := engine.NewExecutor()
	journal, err := executor.Execute(execContext, plan, func(p engine.ProgressEvent) {
		if !cfg.jsonOutput {
			ui.RenderProgressBar(p.BytesProcessed, p.TotalBytes, p.Percent, p.SpeedBytesSec, p.ETA, p.StepName)
		}
	})
	if !cfg.jsonOutput {
		ui.ClearProgressBar()
	}

	if err != nil {
		ui.PrintError(fmt.Sprintf("Deployment failed: %v", err))
		os.Exit(1)
	}

	if cfg.jsonOutput {
		_ = json.NewEncoder(os.Stdout).Encode(journal)
		return
	}

	// Complete Banner
	fmt.Println()
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println(ui.Green(ui.Bold("  🎉 CONGRATULATIONS! BOOTABLE USB IS READY! 🎉               ")))
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Printf(" • Target USB:   %s\n", ui.Bold(targetDev.Path))
	fmt.Printf(" • Boot Engine:  Ventox Native (%s)\n", plan.Mode)
	fmt.Printf(" • Loaded Image: %s (%s)\n", ui.Bold(sourceMeta.Name), sourceMeta.FormattedSize())
	if plan.BypassWindows11 {
		fmt.Printf(" • Bypasses:     TPM 2.0, Secure Boot, RAM, CPU, Storage, NRO\n")
	}
	if plan.VerifyAfterWrite {
		fmt.Printf(" • Verification: SHA-256 Verified\n")
	}
	fmt.Println(" • Ready to boot on any modern PC (Press F12, F11, or Esc at boot)")
	fmt.Println(ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════")))
	fmt.Println()
}

func selectDeviceInteractive(devs []domain.BlockDevice) (domain.BlockDevice, error) {
	if len(devs) == 0 {
		ui.PrintWarning("No USB storage device automatically detected.")
		for {
			val, err := ui.Prompt("Enter target device path (e.g. /dev/sdb, /dev/disk2)", "")
			if err != nil {
				return domain.BlockDevice{}, err
			}
			if val != "" {
				return domain.BlockDevice{Path: val, RawPath: val, IsRemovable: true}, nil
			}
		}
	}

	if len(devs) == 1 {
		d := devs[0]
		fmt.Println(ui.Green(ui.Bold("Auto-detected USB Drive:")))
		fmt.Printf("   Device: %s\n", ui.Bold(d.Path))
		fmt.Printf("   Size:   %s\n", ui.Cyan(d.FormattedSize()))
		if d.Model != "" || d.Vendor != "" {
			fmt.Printf("   Model:  %s %s\n", d.Vendor, d.Model)
		}
		if len(d.Mountpoints) > 0 {
			fmt.Printf("   Mounts: %s\n", ui.Gray(strings.Join(d.Mountpoints, ", ")))
		}
		fmt.Println()

		choice, err := ui.Prompt("Use this USB drive? [Y/n/custom]", "Y")
		if err != nil {
			return domain.BlockDevice{}, err
		}
		if strings.ToLower(choice) == "y" || strings.ToLower(choice) == "yes" {
			return d, nil
		}

		val, _ := ui.Prompt("Enter custom device path (e.g. /dev/sdb)", "")
		return domain.BlockDevice{Path: val, RawPath: val, IsRemovable: true}, nil
	}

	fmt.Println(ui.Bold("Multiple USB drives detected. Please select one:"))
	for i, d := range devs {
		fmt.Printf("  %s %s\n", ui.Cyan(fmt.Sprintf("[%d]", i+1)), d.DisplayString())
	}
	fmt.Printf("  %s Enter custom device path\n\n", ui.Cyan("[C]"))

	for {
		choice, err := ui.Prompt("Select USB drive", "1")
		if err != nil {
			return domain.BlockDevice{}, err
		}
		if strings.EqualFold(choice, "c") {
			val, _ := ui.Prompt("Enter custom device path (e.g. /dev/sdb)", "")
			return domain.BlockDevice{Path: val, RawPath: val, IsRemovable: true}, nil
		}
		idx := 0
		if _, scanErr := fmt.Sscanf(choice, "%d", &idx); scanErr == nil && idx >= 1 && idx <= len(devs) {
			return devs[idx-1], nil
		}
		ui.PrintError("Invalid choice. Please select a valid number or 'C'.")
	}
}

func selectImageInteractive(images []domain.ImageMetadata) (string, error) {
	if len(images) == 0 {
		ui.PrintWarning("No disk images found automatically in Downloads or Desktop.")
		for {
			val, err := ui.Prompt("Enter full path to .iso or .img file", "")
			if err != nil {
				return "", err
			}
			if _, statErr := os.Stat(val); statErr == nil {
				return val, nil
			}
			ui.PrintError("File not found! Please enter a valid path.")
		}
	}

	if len(images) == 1 {
		img := images[0]
		fmt.Println(ui.Green(ui.Bold("Auto-detected Image:")))
		fmt.Printf("   Name: %s\n", ui.Bold(img.Name))
		fmt.Printf("   Size: %s\n", ui.Cyan(img.FormattedSize()))
		fmt.Printf("   Path: %s\n\n", ui.Gray(img.Path))

		choice, err := ui.Prompt("Use this image? [Y/n/custom]", "Y")
		if err != nil {
			return "", err
		}
		if strings.ToLower(choice) == "y" || strings.ToLower(choice) == "yes" {
			return img.Path, nil
		}

		for {
			val, _ := ui.Prompt("Enter custom image path", "")
			if _, statErr := os.Stat(val); statErr == nil {
				return val, nil
			}
			ui.PrintError("File not found! Please enter a valid path.")
		}
	}

	fmt.Println(ui.Bold("Auto-detected disk images:"))
	for i, img := range images {
		fmt.Printf("  %s %s %s\n     %s\n",
			ui.Cyan(fmt.Sprintf("[%d]", i+1)),
			ui.Bold(img.Name),
			ui.Cyan(fmt.Sprintf("(%s)", img.FormattedSize())),
			ui.Gray(img.Path),
		)
	}
	fmt.Printf("  %s Enter custom image path\n\n", ui.Cyan("[C]"))

	for {
		choice, err := ui.Prompt("Select image", "1")
		if err != nil {
			return "", err
		}
		if strings.EqualFold(choice, "c") {
			for {
				val, _ := ui.Prompt("Enter custom image path", "")
				if _, statErr := os.Stat(val); statErr == nil {
					return val, nil
				}
				ui.PrintError("File not found! Please enter a valid path.")
			}
		}
		idx := 0
		if _, scanErr := fmt.Sscanf(choice, "%d", &idx); scanErr == nil && idx >= 1 && idx <= len(images) {
			return images[idx-1].Path, nil
		}
		ui.PrintError("Invalid choice. Please select a valid number or 'C'.")
	}
}
