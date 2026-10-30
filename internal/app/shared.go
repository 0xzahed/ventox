package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

// usbDevices keeps only removable or USB-attached devices.
func usbDevices(devs []domain.BlockDevice) []domain.BlockDevice {
	var out []domain.BlockDevice
	for _, d := range devs {
		if d.IsRemovable || d.BusType == "USB" {
			out = append(out, d)
		}
	}
	return out
}

// selectTarget resolves the target device from --device or an interactive menu.
// fallback builds the device description used when the provider cannot describe an explicit path.
func selectTarget(ctx context.Context, deps Deps, cfg Config, step, total int, title string, fallback func(path string) domain.BlockDevice) (domain.BlockDevice, error) {
	if cfg.DevicePath != "" {
		dev, err := deps.Devices.GetDevice(ctx, cfg.DevicePath)
		if err != nil {
			return fallback(cfg.DevicePath), nil
		}
		return dev, nil
	}

	ui.Banner()
	ui.PrintStep(step, total, title)
	devs, err := deps.Devices.ListDevices(ctx)
	if err != nil && len(devs) == 0 {
		return domain.BlockDevice{}, fmt.Errorf("failed to list devices: %w", err)
	}
	selected, err := ui.SelectDevice(usbDevices(devs))
	if err != nil {
		return domain.BlockDevice{}, fmt.Errorf("device selection failed: %w", err)
	}
	return selected, nil
}

// confirm asks the user to approve a plan unless --yes was given.
func confirm(cfg Config, plan *domain.WritePlan) error {
	if cfg.AutoConfirm || ui.ConfirmPlan(plan) {
		return nil
	}
	return ErrCancelled
}

// execute runs a plan, rendering progress for humans or emitting the journal as JSON.
func execute(ctx context.Context, deps Deps, cfg Config, plan *domain.WritePlan) (err error) {
	var progress engine.ProgressBroadcaster
	if !cfg.JSONOutput {
		progress.Subscribe(engine.ProgressReporter(func(p engine.ProgressEvent) {
			ui.RenderProgressBar(p.BytesProcessed, p.TotalBytes, p.Percent, p.SpeedBytesSec, p.ETA, p.StepName)
		}))
		defer ui.ClearProgressBar()
	}

	journal, execErr := deps.NewExecutor().Execute(ctx, plan, progress.OnProgress)
	if execErr != nil {
		if errors.Is(execErr, domain.ErrCancelled) {
			ui.PrintWarning("Interrupt received; disk operation cancelled.")
		}
		return fmt.Errorf("execution failed: %w", execErr)
	}
	if cfg.JSONOutput {
		return writeJSON(deps.Stdout, journal)
	}
	return nil
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON output: %w", err)
	}
	return nil
}
