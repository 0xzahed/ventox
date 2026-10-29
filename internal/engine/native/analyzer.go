package native

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type OSType string

const (
	OSWindows OSType = "Windows"
	OSLinux   OSType = "Linux"
	OSUnknown OSType = "Unknown"
)

type ISOAnalysis struct {
	Type        OSType
	IsWindows   bool
	IsLinux     bool
	IsEFI       bool
	HasLargeWIM bool
	WIMSize     int64
	WIMPath     string
}

// AnalyzeISO temporarily loop-mounts the ISO image to inspect its filesystem structure.
func AnalyzeISO(isoPath string) (*ISOAnalysis, error) {
	tempMount, err := os.MkdirTemp("", "ventox_iso_anal_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp mount: %w", err)
	}
	defer func() {
		_ = exec.Command("umount", "-l", tempMount).Run()
		_ = os.Remove(tempMount)
	}()

	// Mount ISO read-only using loop device
	cmd := exec.Command("mount", "-o", "loop,ro", isoPath, tempMount)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to loop-mount ISO: %s (%w)", string(out), err)
	}

	analysis := &ISOAnalysis{
		Type: OSUnknown,
	}

	// 1. Check for Windows setup files
	wimCandidates := []string{
		"sources/install.wim",
		"sources/INSTALL.WIM",
		"sources/install.esd",
		"sources/INSTALL.ESD",
		"sources/install.swm",
	}

	for _, rel := range wimCandidates {
		target := filepath.Join(tempMount, rel)
		if stat, err := os.Stat(target); err == nil {
			analysis.IsWindows = true
			analysis.Type = OSWindows
			analysis.WIMSize = stat.Size()
			analysis.WIMPath = rel
			if stat.Size() > 4*1024*1024*1024 { // > 4 GiB
				analysis.HasLargeWIM = true
			}
			break
		}
	}

	// 2. Check for UEFI bootloader
	efiCandidates := []string{
		"efi/boot/bootx64.efi",
		"EFI/BOOT/BOOTX64.EFI",
		"efi/boot/bootaa64.efi",
		"EFI/BOOT/BOOTAA64.EFI",
		"efi/boot/bootia32.efi",
	}
	for _, rel := range efiCandidates {
		if _, err := os.Stat(filepath.Join(tempMount, rel)); err == nil {
			analysis.IsEFI = true
			break
		}
	}

	// 3. Check for Linux indicators
	if !analysis.IsWindows {
		linuxIndicators := []string{
			"isolinux",
			"casper",
			"live",
			"arch",
			"boot/grub",
			"pool",
			"dists",
		}
		for _, ind := range linuxIndicators {
			if _, err := os.Stat(filepath.Join(tempMount, ind)); err == nil {
				analysis.IsLinux = true
				analysis.Type = OSLinux
				break
			}
		}
	}

	// Sleep briefly before unmount
	time.Sleep(100 * time.Millisecond)
	return analysis, nil
}
