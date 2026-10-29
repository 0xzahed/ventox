//go:build windows

package engine

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/0xzahed/ventox/internal/ui"
)

// FormatDisk formats or resets the target device according to the chosen mode on Windows.
func FormatDisk(devicePath string, mode FormatMode, label string) error {
	if label == "" {
		label = "VENTOX"
	}
	label = strings.ToUpper(strings.TrimSpace(label))

	var fs string
	switch mode {
	case ModeCleanExFAT:
		fs = "exFAT"
	case ModeCleanFAT32:
		fs = "FAT32"
		if len(label) > 11 {
			label = label[:11]
		}
	case ModeCleanNTFS:
		fs = "NTFS"
	default:
		return fmt.Errorf("mode %d is not supported natively on Windows", mode)
	}

	ui.PrintInfo(fmt.Sprintf("Formatting %s as %s...", devicePath, fs))
	cmd := exec.Command("format", devicePath, "/FS:"+fs, "/V:"+label, "/Q", "/Y")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("format error: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}
