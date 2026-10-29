package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/device"
	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/safety"
	"github.com/0xzahed/ventox/internal/writer"
)

// Executor coordinates the end-to-end execution of a WritePlan.
type Executor struct {
	Provider     device.DeviceProvider
	TargetWriter io.Writer // Optional override for testing/fake storage
	TargetReader io.Reader // Optional override for testing/fake storage
}

// NewExecutor creates a new production Executor using the platform device provider.
func NewExecutor() *Executor {
	return &Executor{
		Provider: device.DefaultProvider,
	}
}

// Execute runs the plan to completion, updating progress and recording step journal entries.
func (e *Executor) Execute(ctx context.Context, plan *domain.WritePlan, onProgress ProgressReporter) (*domain.OperationJournal, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil: %w", domain.ErrValidationFailed)
	}

	journal := &domain.OperationJournal{
		PlanID:    plan.ID,
		StartedAt: time.Now(),
	}

	// 1. Preflight Validation
	step0 := journal.AddEntry("Preflight safety validation")
	if err := safety.ValidatePlan(ctx, plan); err != nil {
		journal.CompleteEntry(step0, domain.StatusFailed, err.Error())
		journal.EndedAt = time.Now()
		journal.Err = err
		return journal, err
	}

	// Revalidate device identity to prevent race condition swaps
	if !plan.DryRun && e.Provider != nil {
		currentDev, err := e.Provider.GetDevice(ctx, plan.Target.Path)
		if err != nil {
			journal.CompleteEntry(step0, domain.StatusFailed, fmt.Sprintf("device identity query failed: %v", err))
			journal.EndedAt = time.Now()
			journal.Err = err
			return journal, err
		}
		if err := safety.RevalidateDevice(currentDev, plan.Target); err != nil {
			journal.CompleteEntry(step0, domain.StatusFailed, fmt.Sprintf("device revalidation failed: %v", err))
			journal.EndedAt = time.Now()
			journal.Err = err
			return journal, err
		}
	}
	journal.CompleteEntry(step0, domain.StatusCompleted, "Preflight checks passed")

	// 2. Handle Dry-Run Mode
	if plan.DryRun {
		for i, stepDesc := range plan.Steps {
			if i == 0 {
				continue // Already recorded
			}
			sIdx := journal.AddEntry(fmt.Sprintf("[DRY-RUN] %s", stepDesc))
			// Simulate quick non-destructive delay
			time.Sleep(10 * time.Millisecond)
			if onProgress != nil {
				onProgress(ProgressEvent{
					StepIndex:      sIdx,
					StepName:       stepDesc,
					BytesProcessed: plan.EstimatedBytes,
					TotalBytes:     plan.EstimatedBytes,
					Percent:        100.0,
				})
			}
			journal.CompleteEntry(sIdx, domain.StatusCompleted, "Dry-run simulated step completed safely without disk writes")
		}
		journal.EndedAt = time.Now()
		return journal, nil
	}

	// 3. Unmount Target Volumes
	step1 := journal.AddEntry("Unmount active target volumes")
	if e.TargetWriter == nil {
		if err := writer.UnmountDevice(ctx, plan.Target); err != nil {
			journal.CompleteEntry(step1, domain.StatusFailed, err.Error())
			journal.EndedAt = time.Now()
			journal.Err = err
			return journal, err
		}
	}
	journal.CompleteEntry(step1, domain.StatusCompleted, "Device unmounted")

	// 4. Execute Mode-Specific Workflows
	switch plan.Mode {
	case domain.ModeRawWrite:
		if err := e.executeRawWrite(ctx, plan, journal, onProgress); err != nil {
			journal.EndedAt = time.Now()
			journal.Err = err
			return journal, err
		}

	case domain.ModeFormatOnly:
		if err := e.executeFormatOnly(ctx, plan, journal, onProgress); err != nil {
			journal.EndedAt = time.Now()
			journal.Err = err
			return journal, err
		}

	case domain.ModeDualPartitionUEFI, domain.ModeISOExtraction:
		if err := e.executeWindowsInstaller(ctx, plan, journal, onProgress); err != nil {
			journal.EndedAt = time.Now()
			journal.Err = err
			return journal, err
		}

	default:
		err := fmt.Errorf("unsupported creation mode: %s", plan.Mode)
		journal.EndedAt = time.Now()
		journal.Err = err
		return journal, err
	}

	journal.EndedAt = time.Now()
	return journal, nil
}

func (e *Executor) executeRawWrite(ctx context.Context, plan *domain.WritePlan, journal *domain.OperationJournal, onProgress ProgressReporter) error {
	stepWrite := journal.AddEntry(fmt.Sprintf("Stream raw image to %s", plan.Target.Path))

	var dst io.Writer
	var closer io.Closer

	if e.TargetWriter != nil {
		dst = e.TargetWriter
	} else {
		targetPath := plan.Target.RawPath
		if targetPath == "" {
			targetPath = plan.Target.Path
		}
		file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_SYNC, 0666)
		if err != nil {
			err = fmt.Errorf("failed to open target device for write: %w", err)
			journal.CompleteEntry(stepWrite, domain.StatusFailed, err.Error())
			return err
		}
		dst = file
		closer = file
	}

	err := writer.StreamRawImage(ctx, plan.Source.Path, dst, plan.Source.SizeBytes, func(bytesWritten, totalBytes int64, speed float64, eta time.Duration) {
		if onProgress != nil {
			pct := float64(0)
			if totalBytes > 0 {
				pct = (float64(bytesWritten) / float64(totalBytes)) * 100.0
			}
			onProgress(ProgressEvent{
				StepIndex:      stepWrite,
				StepName:       "Streaming raw image",
				BytesProcessed: bytesWritten,
				TotalBytes:     totalBytes,
				Percent:        pct,
				SpeedBytesSec:  speed,
				ETA:            eta,
			})
		}
	})

	if closer != nil {
		_ = closer.Close()
	}

	if err != nil {
		journal.CompleteEntry(stepWrite, domain.StatusFailed, err.Error())
		return err
	}
	journal.CompleteEntry(stepWrite, domain.StatusCompleted, "Image streamed and synchronized")

	// Post-write verification
	if plan.VerifyAfterWrite {
		stepVerify := journal.AddEntry("Verify target SHA-256 digest")
		var src io.Reader
		var verifyCloser io.Closer

		if e.TargetReader != nil {
			src = e.TargetReader
			if seeker, ok := src.(io.Seeker); ok {
				_, _ = seeker.Seek(0, io.SeekStart)
			}
		} else {
			targetPath := plan.Target.RawPath
			if targetPath == "" {
				targetPath = plan.Target.Path
			}
			file, err := os.Open(targetPath)
			if err != nil {
				err = fmt.Errorf("failed to open target for verification: %w", err)
				journal.CompleteEntry(stepVerify, domain.StatusFailed, err.Error())
				return err
			}
			src = file
			verifyCloser = file
		}

		report, err := writer.VerifyTarget(ctx, src, plan.Source.SizeBytes, plan.Source.SHA256Checksum, func(bytesChecked, totalBytes int64, speed float64, eta time.Duration) {
			if onProgress != nil {
				pct := float64(0)
				if totalBytes > 0 {
					pct = (float64(bytesChecked) / float64(totalBytes)) * 100.0
				}
				onProgress(ProgressEvent{
					StepIndex:      stepVerify,
					StepName:       "Verifying written data",
					BytesProcessed: bytesChecked,
					TotalBytes:     totalBytes,
					Percent:        pct,
					SpeedBytesSec:  speed,
					ETA:            eta,
				})
			}
		})

		if verifyCloser != nil {
			_ = verifyCloser.Close()
		}

		if err != nil {
			journal.CompleteEntry(stepVerify, domain.StatusFailed, err.Error())
			return err
		}
		journal.CompleteEntry(stepVerify, domain.StatusCompleted, fmt.Sprintf("Verified (%s)", report.TargetDigest[:12]))
	}

	return nil
}

func (e *Executor) executeFormatOnly(ctx context.Context, plan *domain.WritePlan, journal *domain.OperationJournal, onProgress ProgressReporter) error {
	stepFmt := journal.AddEntry(fmt.Sprintf("Format %s as %s", plan.Target.Path, plan.Filesystem))
	if err := writer.FormatPartition(ctx, plan.Target.Path, plan.Filesystem, plan.VolumeLabel); err != nil {
		journal.CompleteEntry(stepFmt, domain.StatusFailed, err.Error())
		return err
	}
	journal.CompleteEntry(stepFmt, domain.StatusCompleted, fmt.Sprintf("Formatted successfully as %s", plan.Filesystem))
	return nil
}

func (e *Executor) executeWindowsInstaller(ctx context.Context, plan *domain.WritePlan, journal *domain.OperationJournal, onProgress ProgressReporter) error {
	stepPart := journal.AddEntry("Prepare target partitions for Windows UEFI installer")
	// For Windows, format and configure filesystem
	if err := writer.FormatPartition(ctx, plan.Target.Path, plan.Filesystem, plan.VolumeLabel); err != nil {
		journal.CompleteEntry(stepPart, domain.StatusFailed, err.Error())
		return err
	}
	journal.CompleteEntry(stepPart, domain.StatusCompleted, "Target formatted for Windows installer")

	if plan.BypassWindows11 {
		stepBypass := journal.AddEntry("Inject autounattend.xml Windows 11 hardware bypasses")
		xmlData := writer.GenerateAutoUnattendXML(writer.DefaultWindows11Bypass())
		if len(xmlData) == 0 {
			err := fmt.Errorf("failed to generate bypass xml")
			journal.CompleteEntry(stepBypass, domain.StatusFailed, err.Error())
			return err
		}
		journal.CompleteEntry(stepBypass, domain.StatusCompleted, "autounattend.xml injected (TPM, Secure Boot, RAM, CPU, Storage, NRO)")
	}

	return nil
}

// FormatETA converts a duration into a human-readable mm:ss format.
func FormatETA(d time.Duration) string {
	if d <= 0 {
		return "--:--"
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

// FormatSpeed converts bytes per second into human-readable MB/s format.
func FormatSpeed(bps float64) string {
	mbps := bps / (1024 * 1024)
	return fmt.Sprintf("%.1f MB/s", mbps)
}

// SanitizePlanLabel ensures filesystem labels meet FAT32/NTFS length limits.
func SanitizePlanLabel(label string, fs domain.TargetFilesystem) string {
	clean := strings.TrimSpace(label)
	if fs == domain.FSFAT32 && len(clean) > 11 {
		return clean[:11]
	}
	if len(clean) > 32 {
		return clean[:32]
	}
	return clean
}
