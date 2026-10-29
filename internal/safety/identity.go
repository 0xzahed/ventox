package safety

import (
	"fmt"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

// RevalidateDevice compares device properties immediately before destructive operations
// to prevent device-switch races where another disk is assigned the same path.
func RevalidateDevice(current, baseline domain.BlockDevice) error {
	if current.Path != baseline.Path {
		return fmt.Errorf("%w: path changed from %s to %s",
			domain.ErrIdentityMismatch, baseline.Path, current.Path)
	}

	if current.SizeBytes != baseline.SizeBytes {
		return fmt.Errorf("%w: size changed from %d to %d bytes",
			domain.ErrIdentityMismatch, baseline.SizeBytes, current.SizeBytes)
	}

	if baseline.Model != "" && current.Model != "" {
		if !strings.EqualFold(strings.TrimSpace(current.Model), strings.TrimSpace(baseline.Model)) {
			return fmt.Errorf("%w: model changed from %q to %q",
				domain.ErrIdentityMismatch, baseline.Model, current.Model)
		}
	}

	if baseline.Vendor != "" && current.Vendor != "" {
		if !strings.EqualFold(strings.TrimSpace(current.Vendor), strings.TrimSpace(baseline.Vendor)) {
			return fmt.Errorf("%w: vendor changed from %q to %q",
				domain.ErrIdentityMismatch, baseline.Vendor, current.Vendor)
		}
	}

	return nil
}
