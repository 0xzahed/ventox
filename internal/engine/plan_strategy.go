package engine

import (
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

const (
	stepPreflight = "Preflight safety check and device identity revalidation"
	stepUnmount   = "Unmount all active target mounts"

	formatOnlyEstimatedBytes = 32 * 1024 * 1024 // ~32 MiB partition & FS metadata
)

// planStrategy is the Strategy interface for populating a WritePlan.
// Each implementation owns exactly one creation scenario (Single Responsibility)
// and new scenarios are added without touching existing ones (Open/Closed).
type planStrategy interface {
	// Supports reports whether the strategy handles the given options.
	Supports(opts PlanOptions) bool
	// Apply fills the mode-specific fields of the plan.
	Apply(plan *domain.WritePlan, opts PlanOptions)
}

// planStrategies is the ordered strategy registry; the first supporting strategy wins.
// rawImageStrategy is the catch-all and must stay last.
var planStrategies = []planStrategy{
	formatOnlyStrategy{},
	windowsInstallerStrategy{},
	linuxRawStrategy{},
	rawImageStrategy{},
}

// selectStrategy is a Factory that resolves the strategy for the given options.
func selectStrategy(opts PlanOptions) planStrategy {
	for _, s := range planStrategies {
		if s.Supports(opts) {
			return s
		}
	}
	return rawImageStrategy{}
}

// formatOnlyStrategy plans a format of the device when no source image is given.
type formatOnlyStrategy struct{}

func (formatOnlyStrategy) Supports(opts PlanOptions) bool { return opts.Source == nil }

func (formatOnlyStrategy) Apply(plan *domain.WritePlan, opts PlanOptions) {
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
	plan.EstimatedBytes = formatOnlyEstimatedBytes
	plan.Steps = newStepBuilder(stepPreflight, stepUnmount).
		Addf("Partition disk with %s partition table", plan.Scheme).
		Addf("Format filesystem as %s (Label: %s)", plan.Filesystem, plan.VolumeLabel).
		Add("Synchronize storage buffers and verify filesystem metadata").
		Build()
}

// windowsInstallerStrategy plans a Windows UEFI installer medium.
type windowsInstallerStrategy struct{}

func (windowsInstallerStrategy) Supports(opts PlanOptions) bool {
	return opts.Source != nil && opts.Source.Classification == domain.ClassWindowsInstaller
}

func (windowsInstallerStrategy) Apply(plan *domain.WritePlan, opts PlanOptions) {
	src := opts.Source
	plan.EstimatedBytes = src.SizeBytes

	switch {
	case opts.RequestedMode != "":
		plan.Mode = opts.RequestedMode
	case src.HasLargeWIM:
		plan.Mode = domain.ModeDualPartitionUEFI
	default:
		plan.Mode = domain.ModeISOExtraction
	}

	plan.Scheme = domain.SchemeGPT
	plan.Filesystem = domain.FSNTFS
	plan.VolumeLabel = firstNonEmpty(strings.TrimSpace(opts.VolumeLabel), src.VolumeLabel, "WIN_INSTALL")

	dual := plan.Mode == domain.ModeDualPartitionUEFI
	plan.Steps = newStepBuilder(stepPreflight, stepUnmount).
		AddIf(dual,
			"Create dual-partition layout (Partition 1: FAT32 BOOT, Partition 2: NTFS INSTALL)",
			"Format Partition 1 (FAT32, UEFI Boot) and Partition 2 (NTFS, Installation Data)",
			"Extract and deploy Windows bootloader to Partition 1",
			"Extract and stream Windows installation assets to Partition 2").
		AddIf(!dual,
			"Create GPT partition table and format as NTFS",
			"Extract all Windows installer assets to target filesystem").
		AddIf(opts.BypassWindows11,
			"Inject autounattend.xml bypasses for TPM 2.0, Secure Boot, RAM, and Microsoft Account (NRO)").
		Add("Synchronize buffers and flush hardware write caches").
		AddIf(opts.VerifyAfterWrite, "Verify partition tables and critical boot files").
		Build()
}

// linuxRawStrategy plans a byte-for-byte write of hybrid/bootable images.
type linuxRawStrategy struct{}

func (linuxRawStrategy) Supports(opts PlanOptions) bool {
	if opts.Source == nil {
		return false
	}
	c := opts.Source.Classification
	return c == domain.ClassLinuxHybrid || c == domain.ClassRawBootable
}

func (linuxRawStrategy) Apply(plan *domain.WritePlan, opts PlanOptions) {
	plan.EstimatedBytes = opts.Source.SizeBytes
	plan.Mode = domain.ModeRawWrite
	plan.Scheme = domain.SchemeMBR
	plan.Filesystem = domain.FSFAT32
	plan.VolumeLabel = opts.Source.VolumeLabel
	plan.Steps = newStepBuilder(stepPreflight, stepUnmount).
		Addf("Stream raw image byte-for-byte to %s (4 MiB aligned chunk streaming)", opts.Target.Path).
		Add("Synchronize storage buffers and flush hardware write caches").
		AddIf(opts.VerifyAfterWrite, "Read back target blocks and verify SHA-256 checksum against source image").
		Build()
}

// rawImageStrategy is the fallback for images of unknown classification.
type rawImageStrategy struct{}

func (rawImageStrategy) Supports(PlanOptions) bool { return true }

func (rawImageStrategy) Apply(plan *domain.WritePlan, opts PlanOptions) {
	if opts.Source != nil {
		plan.EstimatedBytes = opts.Source.SizeBytes
	}
	plan.Mode = domain.ModeRawWrite
	plan.Scheme = domain.SchemeGPT
	plan.Steps = newStepBuilder(stepPreflight, stepUnmount).
		Addf("Stream raw image byte-for-byte to %s", opts.Target.Path).
		Add("Synchronize storage buffers and flush hardware write caches").
		AddIf(opts.VerifyAfterWrite, "Read back target blocks and verify SHA-256 checksum").
		Build()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
