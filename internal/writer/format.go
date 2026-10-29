package writer

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

// ValidateFilesystemSupport checks whether the current host platform supports formatting the requested filesystem.
func ValidateFilesystemSupport(fs domain.TargetFilesystem) error {
	switch runtime.GOOS {
	case "linux":
		switch fs {
		case domain.FSFAT32:
			if _, err := exec.LookPath("mkfs.vfat"); err != nil {
				if _, err2 := exec.LookPath("mkfs.fat"); err2 != nil {
					return fmt.Errorf("mkfs.vfat/mkfs.fat not found; install dosfstools")
				}
			}
		case domain.FSExFAT:
			if _, err := exec.LookPath("mkfs.exfat"); err != nil {
				return fmt.Errorf("mkfs.exfat not found; install exfatprogs")
			}
		case domain.FSNTFS:
			if _, err := exec.LookPath("mkfs.ntfs"); err != nil {
				return fmt.Errorf("mkfs.ntfs not found; install ntfs-3g")
			}
		case domain.FSEXT4:
			if _, err := exec.LookPath("mkfs.ext4"); err != nil {
				return fmt.Errorf("mkfs.ext4 not found; install e2fsprogs")
			}
		case domain.FSAPFS:
			return fmt.Errorf("APFS creation is not supported on Linux")
		default:
			return fmt.Errorf("unsupported filesystem: %s", fs)
		}

	case "darwin":
		switch fs {
		case domain.FSFAT32, domain.FSExFAT, domain.FSAPFS:
			// Built into macOS diskutil
		case domain.FSNTFS:
			return fmt.Errorf("native NTFS formatting is read-only on macOS without third-party drivers")
		case domain.FSEXT4:
			return fmt.Errorf("ext4 formatting is not supported natively on macOS")
		default:
			return fmt.Errorf("unsupported filesystem on macOS: %s", fs)
		}

	case "windows":
		switch fs {
		case domain.FSFAT32, domain.FSExFAT, domain.FSNTFS:
			// Built into Windows format command
		case domain.FSEXT4, domain.FSAPFS:
			return fmt.Errorf("filesystem %s is not supported on Windows", fs)
		default:
			return fmt.Errorf("unsupported filesystem on Windows: %s", fs)
		}
	}

	return nil
}

// FormatPartition formats the specified partition or device with the requested filesystem and volume label.
func FormatPartition(ctx context.Context, targetPath string, fs domain.TargetFilesystem, label string) error {
	if err := ValidateFilesystemSupport(fs); err != nil {
		return err
	}

	if label == "" {
		label = "VENTOX"
	}
	// Sanitize label length for FAT32 (max 11 chars)
	if fs == domain.FSFAT32 && len(label) > 11 {
		label = label[:11]
	}

	switch runtime.GOOS {
	case "linux":
		var cmd *exec.Cmd
		switch fs {
		case domain.FSFAT32:
			cmd = exec.CommandContext(ctx, "mkfs.vfat", "-F", "32", "-n", label, targetPath)
		case domain.FSExFAT:
			cmd = exec.CommandContext(ctx, "mkfs.exfat", "-n", label, targetPath)
		case domain.FSNTFS:
			cmd = exec.CommandContext(ctx, "mkfs.ntfs", "-f", "-L", label, targetPath)
		case domain.FSEXT4:
			cmd = exec.CommandContext(ctx, "mkfs.ext4", "-F", "-L", label, targetPath)
		default:
			return fmt.Errorf("unsupported filesystem: %s", fs)
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("formatting %s as %s failed: %s (%w)", targetPath, fs, strings.TrimSpace(string(out)), err)
		}
		return nil

	case "darwin":
		var fsArg string
		switch fs {
		case domain.FSFAT32:
			fsArg = "FAT32"
		case domain.FSExFAT:
			fsArg = "ExFAT"
		case domain.FSAPFS:
			fsArg = "APFS"
		default:
			return fmt.Errorf("unsupported filesystem on macOS: %s", fs)
		}

		cmd := exec.CommandContext(ctx, "diskutil", "eraseVolume", fsArg, label, targetPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("diskutil eraseVolume failed: %s (%w)", strings.TrimSpace(string(out)), err)
		}
		return nil

	case "windows":
		cmd := exec.CommandContext(ctx, "format", targetPath, fmt.Sprintf("/FS:%s", fs), fmt.Sprintf("/V:%s", label), "/Q", "/Y")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("windows format command failed: %s (%w)", strings.TrimSpace(string(out)), err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
