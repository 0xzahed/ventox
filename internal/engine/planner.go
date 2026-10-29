package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
)

// PlanOptions configures the generation of a WritePlan.
type PlanOptions struct {
	Target           domain.BlockDevice
	Source           *domain.ImageMetadata
	RequestedMode    domain.CreationMode
	RequestedScheme  domain.PartitionScheme
	RequestedFS      domain.TargetFilesystem
	VolumeLabel      string
	BypassWindows11  bool
	VerifyAfterWrite bool
	DryRun           bool
}

// CreatePlan produces a deterministic, immutable WritePlan based on target capabilities,
// image classification, and safety constraints.
func CreatePlan(opts PlanOptions) (*domain.WritePlan, error) {
	if opts.Target.Path == "" {
		return nil, fmt.Errorf("target device cannot be empty: %w", domain.ErrValidationFailed)
	}

	planID := generatePlanID()
	plan := &domain.WritePlan{
		ID:               planID,
		Target:           opts.Target,
		Source:           opts.Source,
		BypassWindows11:  opts.BypassWindows11,
		VerifyAfterWrite: opts.VerifyAfterWrite,
		DryRun:           opts.DryRun,
	}

	// 1. Format-only Mode (no source image provided)
	if opts.Source == nil {
		plan.Mode = domain.ModeFormatOnly
		plan.Scheme = opts.RequestedScheme
		if plan.Scheme == "" {
			plan.Scheme = domain.SchemeGPT
		}
		plan.Filesystem = opts.RequestedFS
		if plan.Filesystem == "" {
			plan.Filesystem = domain.FSExFAT
		}
		plan.VolumeLabel = strings.TrimSpace(opts.VolumeLabel)
		if plan.VolumeLabel == "" {
			plan.VolumeLabel = "VENTOX_USB"
		}
		plan.EstimatedBytes = 32 * 1024 * 1024 // ~32 MiB partition & FS metadata
		plan.Steps = []string{
			"Preflight safety check and device identity revalidation",
			"Unmount all active target mounts",
			fmt.Sprintf("Partition disk with %s partition table", plan.Scheme),
			fmt.Sprintf("Format filesystem as %s (Label: %s)", plan.Filesystem, plan.VolumeLabel),
			"Synchronize storage buffers and verify filesystem metadata",
		}
		return plan, nil
	}

	// 2. Image-backed Creation Modes
	plan.EstimatedBytes = opts.Source.SizeBytes

	switch opts.Source.Classification {
	case domain.ClassWindowsInstaller:
		if opts.RequestedMode != "" {
			plan.Mode = opts.RequestedMode
		} else if opts.Source.HasLargeWIM {
			plan.Mode = domain.ModeDualPartitionUEFI
		} else {
			plan.Mode = domain.ModeISOExtraction
		}

		plan.Scheme = domain.SchemeGPT
		plan.Filesystem = domain.FSNTFS
		plan.VolumeLabel = strings.TrimSpace(opts.VolumeLabel)
		if plan.VolumeLabel == "" {
			if opts.Source.VolumeLabel != "" {
				plan.VolumeLabel = opts.Source.VolumeLabel
			} else {
				plan.VolumeLabel = "WIN_INSTALL"
			}
		}

		steps := []string{
			"Preflight safety check and device identity revalidation",
			"Unmount all active target mounts",
		}

		if plan.Mode == domain.ModeDualPartitionUEFI {
			steps = append(steps,
				"Create dual-partition layout (Partition 1: FAT32 BOOT, Partition 2: NTFS INSTALL)",
				"Format Partition 1 (FAT32, UEFI Boot) and Partition 2 (NTFS, Installation Data)",
				"Extract and deploy Windows bootloader to Partition 1",
				"Extract and stream Windows installation assets to Partition 2",
			)
		} else {
			steps = append(steps,
				"Create GPT partition table and format as NTFS",
				"Extract all Windows installer assets to target filesystem",
			)
		}

		if opts.BypassWindows11 {
			steps = append(steps, "Inject autounattend.xml bypasses for TPM 2.0, Secure Boot, RAM, and Microsoft Account (NRO)")
		}

		steps = append(steps, "Synchronize buffers and flush hardware write caches")
		if opts.VerifyAfterWrite {
			steps = append(steps, "Verify partition tables and critical boot files")
		}
		plan.Steps = steps

	case domain.ClassLinuxHybrid, domain.ClassRawBootable:
		plan.Mode = domain.ModeRawWrite
		plan.Scheme = domain.SchemeMBR
		plan.Filesystem = domain.FSFAT32
		plan.VolumeLabel = opts.Source.VolumeLabel

		steps := []string{
			"Preflight safety check and device identity revalidation",
			"Unmount all active target mounts",
			fmt.Sprintf("Stream raw image byte-for-byte to %s (4 MiB aligned chunk streaming)", opts.Target.Path),
			"Synchronize storage buffers and flush hardware write caches",
		}
		if opts.VerifyAfterWrite {
			steps = append(steps, "Read back target blocks and verify SHA-256 checksum against source image")
		}
		plan.Steps = steps

	default: // ClassUnknown or fallback
		plan.Mode = domain.ModeRawWrite
		plan.Scheme = domain.SchemeGPT
		steps := []string{
			"Preflight safety check and device identity revalidation",
			"Unmount all active target mounts",
			fmt.Sprintf("Stream raw image byte-for-byte to %s", opts.Target.Path),
			"Synchronize storage buffers and flush hardware write caches",
		}
		if opts.VerifyAfterWrite {
			steps = append(steps, "Read back target blocks and verify SHA-256 checksum")
		}
		plan.Steps = steps
	}

	return plan, nil
}

func generatePlanID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("plan-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("plan-%s", hex.EncodeToString(b))
}
