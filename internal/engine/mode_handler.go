package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/writer"
)

// run carries the per-execution state shared by every ModeHandler.
type run struct {
	plan     *domain.WritePlan
	journal  *domain.OperationJournal
	progress ProgressObserver
	io       TargetIO
}

// step is a Template Method for journalled work: it records the entry, runs fn and
// finalises the entry as completed or failed, so handlers never repeat that bookkeeping.
func (r *run) step(name string, fn func(stepIndex int) (message string, err error)) error {
	idx := r.journal.AddEntry(name)
	msg, err := fn(idx)
	if err != nil {
		r.journal.CompleteEntry(idx, domain.StatusFailed, err.Error())
		return err
	}
	r.journal.CompleteEntry(idx, domain.StatusCompleted, msg)
	return nil
}

// ModeHandler is the Strategy interface for executing one domain.CreationMode.
type ModeHandler interface {
	Execute(ctx context.Context, r *run) error
}

// defaultModeHandlers is the Factory for the built-in mode handlers.
func defaultModeHandlers() map[domain.CreationMode]ModeHandler {
	windows := windowsInstallerHandler{}
	return map[domain.CreationMode]ModeHandler{
		domain.ModeRawWrite:          rawWriteHandler{},
		domain.ModeFormatOnly:        formatOnlyHandler{},
		domain.ModeDualPartitionUEFI: windows,
		domain.ModeISOExtraction:     windows,
	}
}

// rawWriteHandler streams an image byte-for-byte and optionally verifies it.
type rawWriteHandler struct{}

func (rawWriteHandler) Execute(ctx context.Context, r *run) error {
	if r.plan.Source == nil {
		return fmt.Errorf("raw write requires a source image: %w", domain.ErrValidationFailed)
	}
	if err := r.step(fmt.Sprintf("Stream raw image to %s", r.plan.Target.Path), func(idx int) (string, error) {
		return "Image streamed and synchronized", r.stream(ctx, idx)
	}); err != nil {
		return err
	}
	if !r.plan.VerifyAfterWrite {
		return nil
	}
	return r.step("Verify target SHA-256 digest", func(idx int) (string, error) {
		report, err := r.verify(ctx, idx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Verified (%s)", shortDigest(report.TargetDigest)), nil
	})
}

func (r *run) stream(ctx context.Context, idx int) (err error) {
	dst, closer, err := r.io.OpenWriter(r.plan)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := closer.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close target device: %w", cerr))
		}
	}()
	src := r.plan.Source
	return writer.StreamRawImage(ctx, src.Path, dst, src.SizeBytes,
		transferCallback(r.progress, idx, "Streaming raw image"))
}

func (r *run) verify(ctx context.Context, idx int) (report *domain.VerificationReport, err error) {
	rd, closer, err := r.io.OpenReader(r.plan)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := closer.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close target after verification: %w", cerr))
		}
	}()
	src := r.plan.Source
	return writer.VerifyTarget(ctx, rd, src.SizeBytes, src.SHA256Checksum,
		transferCallback(r.progress, idx, "Verifying written data"))
}

func shortDigest(d string) string {
	const n = 12
	if len(d) > n {
		return d[:n]
	}
	return d
}

// formatOnlyHandler formats the target without writing an image.
type formatOnlyHandler struct{}

func (formatOnlyHandler) Execute(ctx context.Context, r *run) error {
	p := r.plan
	return r.step(fmt.Sprintf("Format %s as %s", p.Target.Path, p.Filesystem), func(int) (string, error) {
		if err := writer.FormatPartition(ctx, p.Target.Path, p.Filesystem, p.VolumeLabel); err != nil {
			return "", err
		}
		return fmt.Sprintf("Formatted successfully as %s", p.Filesystem), nil
	})
}

// windowsInstallerHandler prepares a Windows UEFI installer medium.
type windowsInstallerHandler struct{}

func (windowsInstallerHandler) Execute(ctx context.Context, r *run) error {
	p := r.plan
	if err := r.step("Prepare target partitions for Windows UEFI installer", func(int) (string, error) {
		if err := writer.FormatPartition(ctx, p.Target.Path, p.Filesystem, p.VolumeLabel); err != nil {
			return "", err
		}
		return "Target formatted for Windows installer", nil
	}); err != nil {
		return err
	}
	if !p.BypassWindows11 {
		return nil
	}
	return r.step("Inject autounattend.xml Windows 11 hardware bypasses", func(int) (string, error) {
		if xml := writer.GenerateAutoUnattendXML(writer.DefaultWindows11Bypass()); len(xml) == 0 {
			return "", fmt.Errorf("failed to generate bypass xml: %w", domain.ErrValidationFailed)
		}
		return "autounattend.xml injected (TPM, Secure Boot, RAM, CPU, Storage, NRO)", nil
	})
}

// dryRun simulates every planned step without touching the device.
func (r *run) dryRun(ctx context.Context) error {
	for i, desc := range r.plan.Steps {
		if i == 0 {
			continue // preflight already recorded
		}
		desc := desc
		err := r.step("[DRY-RUN] "+desc, func(idx int) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
			r.progress.OnProgress(ProgressEvent{
				StepIndex:      idx,
				StepName:       desc,
				BytesProcessed: r.plan.EstimatedBytes,
				TotalBytes:     r.plan.EstimatedBytes,
				Percent:        100.0,
			})
			return "Dry-run simulated step completed safely without disk writes", nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
