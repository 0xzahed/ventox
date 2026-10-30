package app

import (
	"context"
	"fmt"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/iso"
	"github.com/0xzahed/ventox/internal/ui"
)

type bootCommand struct{ deps Deps }

func (c bootCommand) Run(ctx context.Context, cfg Config) error {
	target, err := selectTarget(ctx, c.deps, cfg, 1, 4, "Detecting USB Drives",
		func(path string) domain.BlockDevice {
			return domain.BlockDevice{Path: path, RawPath: c.deps.Devices.GetRawPath(path), IsRemovable: true}
		})
	if err != nil {
		return err
	}
	if !cfg.JSONOutput {
		ui.PrintSuccess(fmt.Sprintf("Target USB Selected: %s", target.DisplayString()))
	}

	source, err := c.selectImage(ctx, cfg)
	if err != nil {
		return err
	}
	if !cfg.JSONOutput {
		ui.PrintSuccess(fmt.Sprintf("Image Loaded: %s (%s, %s)", source.Name, source.FormattedSize(), source.Classification))
	}

	plan, err := engine.CreatePlan(engine.PlanOptions{
		Target:           target,
		Source:           &source,
		BypassWindows11:  cfg.BypassWin11,
		VerifyAfterWrite: cfg.Verify,
		DryRun:           cfg.DryRun,
	})
	if err != nil {
		return fmt.Errorf("failed to create operation plan: %w", err)
	}
	if err := confirm(cfg, plan); err != nil {
		return err
	}

	if !cfg.JSONOutput {
		ui.PrintStep(3, 3, fmt.Sprintf("Executing Native Engine (%s)", plan.Mode))
	}
	if err := execute(ctx, c.deps, cfg, plan); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}
	if !cfg.JSONOutput {
		printSuccessBanner(target, source, plan)
	}
	return nil
}

func (c bootCommand) selectImage(ctx context.Context, cfg Config) (domain.ImageMetadata, error) {
	path := cfg.ISOPath
	if path == "" {
		ui.PrintStep(2, 4, "Scanning for ISO Files")
		images, err := iso.DiscoverImages(ctx)
		if err != nil && len(images) == 0 {
			return domain.ImageMetadata{}, fmt.Errorf("image discovery failed: %w", err)
		}
		if path, err = ui.SelectImage(images); err != nil {
			return domain.ImageMetadata{}, fmt.Errorf("image selection error: %w", err)
		}
	}
	meta, err := iso.InspectImage(path)
	if err != nil {
		return domain.ImageMetadata{}, fmt.Errorf("failed to inspect ISO %q: %w", path, err)
	}
	return meta, nil
}

func printSuccessBanner(target domain.BlockDevice, source domain.ImageMetadata, plan *domain.WritePlan) {
	rule := ui.Green(ui.Bold("═══════════════════════════════════════════════════════════════"))
	fmt.Println()
	fmt.Println(rule)
	fmt.Println(ui.Green(ui.Bold("  🎉 CONGRATULATIONS! BOOTABLE USB IS READY! 🎉               ")))
	fmt.Println(rule)
	fmt.Printf(" • Target USB:   %s\n", ui.Bold(target.Path))
	fmt.Printf(" • Boot Engine:  Ventox Native (%s)\n", plan.Mode)
	fmt.Printf(" • Loaded Image: %s (%s)\n", ui.Bold(source.Name), source.FormattedSize())
	if plan.BypassWindows11 {
		fmt.Printf(" • Bypasses:     TPM 2.0, Secure Boot, RAM, CPU, Storage, NRO\n")
	}
	if plan.VerifyAfterWrite {
		fmt.Printf(" • Verification: SHA-256 Verified\n")
	}
	fmt.Println(" • Ready to boot on any modern PC (Press F12, F11, or Esc at boot)")
	fmt.Println(rule)
	fmt.Println()
}
