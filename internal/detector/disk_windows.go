//go:build windows

package detector

import (
	"context"

	"github.com/0xzahed/ventox/internal/device"
)

// DetectUSBDisks queries for USB-connected block devices on Windows
func DetectUSBDisks() ([]USBDisk, error) {
	devices, err := device.DefaultProvider.ListDevices(context.Background())
	if err != nil {
		return nil, err
	}

	var results []USBDisk
	for _, dev := range devices {
		if dev.IsRemovable || dev.BusType == "USB" {
			results = append(results, USBDisk{
				Path:   dev.Path,
				Name:   dev.Name,
				Size:   dev.FormattedSize(),
				Model:  dev.Model,
				Vendor: dev.Vendor,
			})
		}
	}
	return results, nil
}
