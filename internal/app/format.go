package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

type formatCommand struct{ deps Deps }

func (c formatCommand) Run(ctx context.Context, cfg Config) error {
	target, err := selectTarget(ctx, c.deps, cfg, 1, 3, "Select USB Drive to Format",
		func(path string) domain.BlockDevice { return domain.BlockDevice{Path: path, RawPath: path} })
	if err != nil {
		return err
	}

	fs := domain.TargetFilesystem(cfg.FS)
	if fs == "" {
		ui.PrintStep(2, 3, "Select Target Filesystem")
		fmt.Println("Available filesystems: exFAT (Recommended), FAT32, NTFS, ext4")
		choice, err := ui.Prompt("Enter filesystem [exfat/fat32/ntfs/ext4]", "exfat")
		if err != nil {
			return fmt.Errorf("failed to read filesystem choice: %w", err)
		}
		fs = parseFilesystem(choice)
	}

	label := cfg.Label
	if label == "" {
		if label, err = ui.Prompt("Enter Volume Label", "VENTOX"); err != nil {
			return fmt.Errorf("failed to read volume label: %w", err)
		}
	}

	plan, err := engine.CreatePlan(engine.PlanOptions{
		Target:          target,
		RequestedFS:     fs,
		RequestedScheme: domain.SchemeGPT,
		VolumeLabel:     label,
		DryRun:          cfg.DryRun,
	})
	if err != nil {
		return fmt.Errorf("failed to generate format plan: %w", err)
	}
	if err := confirm(cfg, plan); err != nil {
		return err
	}

	ui.PrintInfo(fmt.Sprintf("Executing format plan (%s as %s)...", target.Path, fs))
	if err := execute(ctx, c.deps, cfg, plan); err != nil {
		return fmt.Errorf("format failed: %w", err)
	}
	if !cfg.JSONOutput {
		ui.PrintSuccess(fmt.Sprintf("Drive %s successfully formatted as %s (Label: %s)", target.Path, fs, label))
	}
	return nil
}

// parseFilesystem maps user input to a filesystem, defaulting to exFAT.
func parseFilesystem(input string) domain.TargetFilesystem {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "fat32":
		return domain.FSFAT32
	case "ntfs":
		return domain.FSNTFS
	case "ext4":
		return domain.FSEXT4
	default:
		return domain.FSExFAT
	}
}
