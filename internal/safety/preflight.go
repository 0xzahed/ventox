package safety

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/0xzahed/ventox/internal/domain"
)

// Validator is one link in the preflight Chain of Responsibility.
// Each validator inspects a single concern and returns nil to pass the plan on.
type Validator interface {
	Validate(ctx context.Context, plan *domain.WritePlan) error
}

// ValidatorFunc adapts a plain function to the Validator interface.
type ValidatorFunc func(ctx context.Context, plan *domain.WritePlan) error

func (f ValidatorFunc) Validate(ctx context.Context, plan *domain.WritePlan) error {
	return f(ctx, plan)
}

// Chain runs validators in order and stops at the first failure.
type Chain []Validator

// Validate implements Validator so chains can be nested.
func (c Chain) Validate(ctx context.Context, plan *domain.WritePlan) error {
	for _, v := range c {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := v.Validate(ctx, plan); err != nil {
			return err
		}
	}
	return nil
}

// DefaultChain returns the mandatory preflight validators, ordered by severity.
func DefaultChain() Chain {
	return Chain{
		ValidatorFunc(validateNotSystemDisk),
		ValidatorFunc(validateCapacity),
		ValidatorFunc(validateSourceImage),
	}
}

// ValidatePlan performs complete non-destructive preflight validation on a proposed WritePlan.
func ValidatePlan(ctx context.Context, plan *domain.WritePlan) error {
	if plan == nil {
		return fmt.Errorf("plan cannot be nil: %w", domain.ErrValidationFailed)
	}
	return DefaultChain().Validate(ctx, plan)
}

func validateNotSystemDisk(_ context.Context, plan *domain.WritePlan) error {
	isSys, reason, err := IsSystemDisk(plan.Target)
	if err != nil {
		return domain.NewOpError("SystemDiskCheck", plan.Target.Path, err)
	}
	if isSys {
		return fmt.Errorf("%w: %s. Operation blocked for safety", domain.ErrSystemDisk, reason)
	}
	return nil
}

func validateCapacity(_ context.Context, plan *domain.WritePlan) error {
	return CheckCapacity(plan.Target, plan.Source, plan.Mode)
}

func validateSourceImage(_ context.Context, plan *domain.WritePlan) error {
	if plan.Mode == domain.ModeFormatOnly || plan.Source == nil {
		return nil
	}
	stat, err := os.Stat(plan.Source.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("source image %s not found: %w", plan.Source.Path, domain.ErrValidationFailed)
		}
		return fmt.Errorf("source image error: %w", err)
	}
	if stat.Size() == 0 {
		return fmt.Errorf("source image %s is empty (0 bytes): %w", plan.Source.Path, domain.ErrValidationFailed)
	}
	return nil
}
