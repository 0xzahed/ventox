package safety

import (
	"fmt"

	"github.com/0xzahed/ventox/internal/domain"
)

const PartitionTableOverheadBytes = 16 * 1024 * 1024 // 16 MiB overhead

// CheckCapacity ensures the target storage device has adequate physical capacity.
func CheckCapacity(target domain.BlockDevice, source *domain.ImageMetadata, mode domain.CreationMode) error {
	if mode == domain.ModeFormatOnly || source == nil {
		if target.SizeBytes < 64*1024*1024 { // minimum 64 MiB for formatting
			return fmt.Errorf("%w: device is smaller than 64 MiB (%s)", domain.ErrInsufficientCapacity, target.FormattedSize())
		}
		return nil
	}

	required := source.SizeBytes
	if mode == domain.ModeDualPartitionUEFI {
		// Dual partition has FAT32 BOOT (1024 MiB) + NTFS INSTALL (full image + overhead)
		required = source.SizeBytes + PartitionTableOverheadBytes + (1024 * 1024 * 1024)
	} else {
		required = source.SizeBytes + PartitionTableOverheadBytes
	}

	if target.SizeBytes < required {
		return fmt.Errorf("%w: device capacity (%s) is smaller than required image capacity (%s)",
			domain.ErrInsufficientCapacity,
			target.FormattedSize(),
			formatBytes(required),
		)
	}

	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
