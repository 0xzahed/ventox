package safety

import (
	"context"
	"fmt"
	"os"

	"github.com/0xzahed/ventox/internal/domain"
)

// ValidatePlan performs complete non-destructive preflight validation on a proposed WritePlan.
func ValidatePlan(ctx context.Context, plan *domain.WritePlan) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// 1. Mandatory System Disk Guard
	isSys, reason, err := IsSystemDisk(plan.Target)
	if err != nil {
		return domain.NewOpError("SystemDiskCheck", plan.Target.Path, err)
	}
	if isSys {
		return fmt.Errorf("%w: %s. Operation blocked for safety", domain.ErrSystemDisk, reason)
	}

	// 2. Capacity Check
	if err := CheckCapacity(plan.Target, plan.Source, plan.Mode); err != nil {
		return err
	}

	// 3. Source Image Verification (if applicable)
	if plan.Mode != domain.ModeFormatOnly && plan.Source != nil {
		stat, err := os.Stat(plan.Source.Path)
		if err != nil {
			return fmt.Errorf("source image error: %w", err)
		}
		if stat.Size() == 0 {
			return fmt.Errorf("source image %s is empty (0 bytes)", plan.Source.Path)
		}
	}

	return nil
}
