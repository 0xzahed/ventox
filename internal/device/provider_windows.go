//go:build windows

package device

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/safety"
)

type windowsProvider struct{}

func newPlatformProvider() DeviceProvider {
	return &windowsProvider{}
}

func (p *windowsProvider) GetRawPath(devicePath string) string {
	return devicePath
}

type winDiskJSON struct {
	Number            int    `json:"Number"`
	FriendlyName      string `json:"FriendlyName"`
	SerialNumber      string `json:"SerialNumber"`
	Size              int64  `json:"Size"`
	BusType           string `json:"BusType"`
	OperationalStatus string `json:"OperationalStatus"`
	IsSystem          bool   `json:"IsSystem"`
	IsBoot            bool   `json:"IsBoot"`
}

func (p *windowsProvider) ListDevices(ctx context.Context) ([]domain.BlockDevice, error) {
	// Query Get-Disk via PowerShell
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-Disk | Select-Object Number, FriendlyName, SerialNumber, Size, BusType, OperationalStatus, IsSystem, IsBoot | ConvertTo-Json -Compress")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to query windows disks: %w", err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}

	var rawDisks []winDiskJSON
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal(out, &rawDisks); err != nil {
			return nil, fmt.Errorf("failed to parse disk list: %w", err)
		}
	} else {
		var single winDiskJSON
		if err := json.Unmarshal(out, &single); err != nil {
			return nil, fmt.Errorf("failed to parse single disk: %w", err)
		}
		rawDisks = append(rawDisks, single)
	}

	var devices []domain.BlockDevice
	for _, rd := range rawDisks {
		devPath := fmt.Sprintf(`\\.\PhysicalDrive%d`, rd.Number)
		isRemovable := strings.EqualFold(rd.BusType, "USB")
		isSys := rd.IsSystem || rd.IsBoot

		dev := domain.BlockDevice{
			Path:            devPath,
			RawPath:         devPath,
			Name:            fmt.Sprintf("Disk %d", rd.Number),
			SizeBytes:       rd.Size,
			SectorSizeBytes: 512,
			Model:           strings.TrimSpace(rd.FriendlyName),
			SerialNumber:    strings.TrimSpace(rd.SerialNumber),
			BusType:         strings.ToUpper(strings.TrimSpace(rd.BusType)),
			IsRemovable:     isRemovable,
			IsSystem:        isSys,
		}

		if isSys, _, _ := safety.IsSystemDisk(dev); isSys {
			dev.IsSystem = true
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

func (p *windowsProvider) GetDevice(ctx context.Context, path string) (domain.BlockDevice, error) {
	devices, err := p.ListDevices(ctx)
	if err != nil {
		return domain.BlockDevice{}, err
	}

	cleanTarget := strings.ToLower(filepath.Clean(path))
	for _, dev := range devices {
		if strings.ToLower(filepath.Clean(dev.Path)) == cleanTarget {
			return dev, nil
		}
	}

	return domain.BlockDevice{}, fmt.Errorf("%w: device %s not found", domain.ErrDeviceRemoved, path)
}
