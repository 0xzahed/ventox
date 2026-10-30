package engine

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

	selectStrategy(opts).Apply(plan, opts)
	return plan, nil
}

func generatePlanID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("plan-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("plan-%s", hex.EncodeToString(b))
}
