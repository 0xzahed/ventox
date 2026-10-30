package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/device"
	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/safety"
	"github.com/0xzahed/ventox/internal/writer"
)

// Executor coordinates the end-to-end execution of a WritePlan.
// The zero value plus a Provider is usable; NewExecutor offers functional options.
type Executor struct {
	Provider     device.DeviceProvider
	TargetWriter io.Writer // Optional override for testing/fake storage
	TargetReader io.Reader // Optional override for testing/fake storage

	handlers map[domain.CreationMode]ModeHandler
}

// ExecutorOption configures an Executor (Functional Options pattern).
type ExecutorOption func(*Executor)

// WithProvider sets the device provider used for identity revalidation.
func WithProvider(p device.DeviceProvider) ExecutorOption {
	return func(e *Executor) { e.Provider = p }
}

// WithTargetIO overrides the destination writer/reader, e.g. for simulated storage.
func WithTargetIO(w io.Writer, r io.Reader) ExecutorOption {
	return func(e *Executor) { e.TargetWriter, e.TargetReader = w, r }
}

// WithModeHandler registers or replaces the handler for a creation mode.
func WithModeHandler(mode domain.CreationMode, h ModeHandler) ExecutorOption {
	return func(e *Executor) {
		if e.handlers == nil {
			e.handlers = defaultModeHandlers()
		}
		e.handlers[mode] = h
	}
}

// NewExecutor creates a production Executor using the platform device provider.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{Provider: device.DefaultProvider, handlers: defaultModeHandlers()}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Executor) handlerFor(mode domain.CreationMode) (ModeHandler, error) {
	handlers := e.handlers
	if handlers == nil {
		handlers = defaultModeHandlers()
	}
	h, ok := handlers[mode]
	if !ok {
		return nil, fmt.Errorf("unsupported creation mode %q: %w", mode, domain.ErrValidationFailed)
	}
	return h, nil
}

// Execute runs the plan to completion, updating progress and recording step journal entries.
// The returned journal is non-nil whenever execution started, even on failure.
func (e *Executor) Execute(ctx context.Context, plan *domain.WritePlan, onProgress ProgressReporter) (*domain.OperationJournal, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil: %w", domain.ErrValidationFailed)
	}

	journal := &domain.OperationJournal{PlanID: plan.ID, StartedAt: time.Now()}
	err := e.execute(ctx, &run{
		plan:     plan,
		journal:  journal,
		progress: onProgress,
		io:       deviceTargetIO{writer: e.TargetWriter, reader: e.TargetReader},
	})
	journal.EndedAt = time.Now()
	if err != nil {
		err = classifyCancellation(err)
		journal.Err = err
		return journal, err
	}
	return journal, nil
}

func (e *Executor) execute(ctx context.Context, r *run) error {
	plan := r.plan

	if err := r.step("Preflight safety validation", func(int) (string, error) {
		if err := safety.ValidatePlan(ctx, plan); err != nil {
			return "", err
		}
		if err := e.revalidateIdentity(ctx, plan); err != nil {
			return "", err
		}
		return "Preflight checks passed", nil
	}); err != nil {
		return err
	}

	if plan.DryRun {
		return r.dryRun(ctx)
	}

	handler, err := e.handlerFor(plan.Mode)
	if err != nil {
		return err
	}

	// Substituted storage is not a real device and has nothing to unmount.
	if e.TargetWriter == nil {
		if err := r.step("Unmount active target volumes", func(int) (string, error) {
			if err := writer.UnmountDevice(ctx, plan.Target); err != nil {
				return "", err
			}
			return "Device unmounted", nil
		}); err != nil {
			return err
		}
	}

	return handler.Execute(ctx, r)
}

// revalidateIdentity guards against device-swap races between planning and writing.
func (e *Executor) revalidateIdentity(ctx context.Context, plan *domain.WritePlan) error {
	if plan.DryRun || e.Provider == nil {
		return nil
	}
	current, err := e.Provider.GetDevice(ctx, plan.Target.Path)
	if err != nil {
		return fmt.Errorf("device identity query failed: %w", err)
	}
	if err := safety.RevalidateDevice(current, plan.Target); err != nil {
		return fmt.Errorf("device revalidation failed: %w", err)
	}
	return nil
}

// classifyCancellation marks context cancellation as domain.ErrCancelled while
// preserving the original error for errors.Is checks.
func classifyCancellation(err error) error {
	if errors.Is(err, context.Canceled) && !errors.Is(err, domain.ErrCancelled) {
		return fmt.Errorf("%w: %w", domain.ErrCancelled, err)
	}
	return err
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
