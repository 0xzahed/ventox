//go:build linux

package device

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/safety"
)

type linuxProvider struct{}

func newPlatformProvider() DeviceProvider {
	return &linuxProvider{}
}

type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name        string        `json:"name"`
	Path        string        `json:"path"`
	Size        json.Number   `json:"size"`
	Type        string        `json:"type"`
	Mountpoints []string      `json:"mountpoints"`
	RM          bool          `json:"rm"`
	Model       string        `json:"model"`
	Vendor      string        `json:"vendor"`
	Serial      string        `json:"serial"`
	Tran        string        `json:"tran"`
	Children    []lsblkDevice `json:"children"`
}

func (p *linuxProvider) GetRawPath(devicePath string) string {
	return devicePath
}

func (p *linuxProvider) ListDevices(ctx context.Context) ([]domain.BlockDevice, error) {
	cmd := exec.CommandContext(ctx, "lsblk", "-b", "-J", "-o", "NAME,PATH,SIZE,TYPE,MOUNTPOINTS,RM,MODEL,VENDOR,SERIAL,TRAN")
	out, err := cmd.Output()
	if err != nil {
		// Fallback to sysfs if lsblk is missing or fails
		return p.listFromSysfs(ctx)
	}

	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	var devices []domain.BlockDevice
	for _, raw := range parsed.BlockDevices {
		if raw.Type != "disk" {
			continue
		}

		size, _ := raw.Size.Int64()
		devPath := raw.Path
		if devPath == "" {
			devPath = "/dev/" + raw.Name
		}

		sectorSize := readSectorSize(raw.Name)
		isRemovable := raw.RM || strings.ToLower(raw.Tran) == "usb"

		var parts []domain.PartitionInfo
		var allMounts []string
		for _, m := range raw.Mountpoints {
			if m != "" {
				allMounts = append(allMounts, m)
			}
		}

		for _, child := range raw.Children {
			childSize, _ := child.Size.Int64()
			childPath := child.Path
			if childPath == "" {
				childPath = "/dev/" + child.Name
			}
			var childMounts []string
			for _, m := range child.Mountpoints {
				if m != "" {
					childMounts = append(childMounts, m)
					allMounts = append(allMounts, m)
				}
			}
			parts = append(parts, domain.PartitionInfo{
				Path:        childPath,
				Name:        child.Name,
				SizeBytes:   childSize,
				Mountpoints: childMounts,
			})
		}

		dev := domain.BlockDevice{
			Path:            devPath,
			RawPath:         devPath,
			Name:            raw.Name,
			SizeBytes:       size,
			SectorSizeBytes: sectorSize,
			Vendor:          strings.TrimSpace(raw.Vendor),
			Model:           strings.TrimSpace(raw.Model),
			SerialNumber:    strings.TrimSpace(raw.Serial),
			BusType:         strings.ToUpper(strings.TrimSpace(raw.Tran)),
			IsRemovable:     isRemovable,
			Mountpoints:     allMounts,
			Partitions:      parts,
		}

		// Check system disk status
		if isSys, _, _ := safety.IsSystemDisk(dev); isSys {
			dev.IsSystem = true
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

func (p *linuxProvider) GetDevice(ctx context.Context, path string) (domain.BlockDevice, error) {
	devices, err := p.ListDevices(ctx)
	if err == nil {
		cleanTarget := filepath.Clean(path)
		for _, dev := range devices {
			if filepath.Clean(dev.Path) == cleanTarget {
				return dev, nil
			}
		}
	}

	// Direct lookup for disk image files or block devices
	if stat, err := os.Stat(path); err == nil {
		size := stat.Size()
		if stat.Mode()&os.ModeDevice != 0 {
			// Query block device size via blockdev
			out, err := exec.CommandContext(ctx, "blockdev", "--getsize64", path).Output()
			if err == nil {
				if parsed, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil && parsed > 0 {
					size = parsed
				}
			}
		}
		baseName := filepath.Base(path)
		sectorSize := readSectorSize(baseName)
		dev := domain.BlockDevice{
			Path:            path,
			RawPath:         path,
			Name:            baseName,
			SizeBytes:       size,
			SectorSizeBytes: sectorSize,
			IsRemovable:     true,
		}
		if isSys, _, _ := safety.IsSystemDisk(dev); isSys {
			dev.IsSystem = true
		}
		return dev, nil
	}

	return domain.BlockDevice{}, fmt.Errorf("%w: device %s not found", domain.ErrDeviceRemoved, path)
}

func readSectorSize(name string) int64 {
	path := fmt.Sprintf("/sys/block/%s/queue/hw_sector_size", name)
	data, err := os.ReadFile(path)
	if err == nil {
		if val, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil && val > 0 {
			return val
		}
	}
	return 512
}

func (p *linuxProvider) listFromSysfs(ctx context.Context) ([]domain.BlockDevice, error) {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, fmt.Errorf("failed to read /sys/block: %w", err)
	}

	var devices []domain.BlockDevice
	for _, entry := range entries {
		name := entry.Name()
		// Filter out loop, ram, dm devices
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "dm-") {
			continue
		}

		devPath := "/dev/" + name
		sizeSectors := readInt64(fmt.Sprintf("/sys/block/%s/size", name))
		sectorSize := readSectorSize(name)
		removable := readInt64(fmt.Sprintf("/sys/block/%s/removable", name)) == 1

		dev := domain.BlockDevice{
			Path:            devPath,
			RawPath:         devPath,
			Name:            name,
			SizeBytes:       sizeSectors * 512,
			SectorSizeBytes: sectorSize,
			IsRemovable:     removable,
		}

		if isSys, _, _ := safety.IsSystemDisk(dev); isSys {
			dev.IsSystem = true
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

func readInt64(path string) int64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	val, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	return val
}
