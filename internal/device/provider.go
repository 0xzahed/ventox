package device

import (
	"context"

	"github.com/0xzahed/ventox/internal/domain"
)

// DeviceProvider defines the platform-agnostic interface for querying storage devices.
type DeviceProvider interface {
	// ListDevices returns all detected storage devices, with metadata and system flags.
	ListDevices(ctx context.Context) ([]domain.BlockDevice, error)

	// GetDevice returns details for a specific device path.
	GetDevice(ctx context.Context, path string) (domain.BlockDevice, error)

	// GetRawPath returns the optimized character/raw device path if supported (e.g., /dev/rdisk on macOS).
	GetRawPath(devicePath string) string
}

// DefaultProvider returns the active platform DeviceProvider.
var DefaultProvider DeviceProvider = newPlatformProvider()
