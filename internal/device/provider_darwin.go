//go:build darwin

package device

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
	"github.com/0xzahed/ventox/internal/safety"
)

type darwinProvider struct{}

func newPlatformProvider() DeviceProvider {
	return &darwinProvider{}
}

func (p *darwinProvider) GetRawPath(devicePath string) string {
	// macOS raw character device optimization: /dev/diskN -> /dev/rdiskN
	clean := filepath.Clean(devicePath)
	if strings.HasPrefix(clean, "/dev/disk") {
		return strings.Replace(clean, "/dev/disk", "/dev/rdisk", 1)
	}
	return devicePath
}

// diskutilPlist represents minimal plist structure for diskutil list -plist
type diskutilListPlist struct {
	AllDisksAndPartitions []diskutilDiskEntry `xml:"dict>array>dict"`
}

type diskutilDiskEntry struct {
	DeviceIdentifier string                 `xml:"string"`
	Size             int64                  `xml:"integer"`
	Partitions       []diskutilPartitionEntry `xml:"array>dict"`
	MountPoint       string                 `xml:"string"`
}

type diskutilPartitionEntry struct {
	DeviceIdentifier string `xml:"string"`
	Size             int64  `xml:"integer"`
	MountPoint       string `xml:"string"`
	VolumeName       string `xml:"string"`
}

func (p *darwinProvider) ListDevices(ctx context.Context) ([]domain.BlockDevice, error) {
	cmd := exec.CommandContext(ctx, "diskutil", "list", "-plist", "physical")
	out, err := cmd.Output()
	if err != nil {
		// Try without "physical" argument on older macOS versions
		cmd = exec.CommandContext(ctx, "diskutil", "list", "-plist")
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("diskutil list failed: %w", err)
		}
	}

	diskIDs := extractDiskIdentifiers(out)
	var devices []domain.BlockDevice

	for _, diskID := range diskIDs {
		// Only inspect whole disks, not slice partitions (e.g. disk2, not disk2s1)
		if strings.Contains(diskID, "s") {
			continue
		}

		dev, err := p.inspectDisk(ctx, diskID)
		if err != nil {
			continue
		}

		if isSys, _, _ := safety.IsSystemDisk(dev); isSys {
			dev.IsSystem = true
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

func (p *darwinProvider) GetDevice(ctx context.Context, path string) (domain.BlockDevice, error) {
	devices, err := p.ListDevices(ctx)
	if err != nil {
		return domain.BlockDevice{}, err
	}

	cleanTarget := filepath.Clean(path)
	for _, dev := range devices {
		if filepath.Clean(dev.Path) == cleanTarget || filepath.Clean(dev.RawPath) == cleanTarget {
			return dev, nil
		}
	}

	return domain.BlockDevice{}, fmt.Errorf("%w: device %s not found", domain.ErrDeviceRemoved, path)
}

func (p *darwinProvider) inspectDisk(ctx context.Context, diskID string) (domain.BlockDevice, error) {
	cmd := exec.CommandContext(ctx, "diskutil", "info", "-plist", diskID)
	out, err := cmd.Output()
	if err != nil {
		return domain.BlockDevice{}, err
	}

	// Simple plist parsing using key/value extraction
	info := parsePlistKeyValue(out)

	devPath := "/dev/" + diskID
	rawPath := "/dev/r" + diskID
	size := info["TotalSize"].(int64)
	sectorSize := info["DeviceBlockSize"].(int64)
	if sectorSize == 0 {
		sectorSize = 512
	}

	busProtocol, _ := info["BusProtocol"].(string)
	removable, _ := info["RemovableMedia"].(bool)
	ejectable, _ := info["Ejectable"].(bool)
	internal, _ := info["Internal"].(bool)
	mediaName, _ := info["MediaName"].(string)

	isRemovable := removable || ejectable || (!internal && busProtocol == "USB")

	var mountpoints []string
	if mp, ok := info["MountPoint"].(string); ok && mp != "" {
		mountpoints = append(mountpoints, mp)
	}

	return domain.BlockDevice{
		Path:            devPath,
		RawPath:         rawPath,
		Name:            diskID,
		SizeBytes:       size,
		SectorSizeBytes: sectorSize,
		Model:           mediaName,
		BusType:         busProtocol,
		IsRemovable:     isRemovable,
		Mountpoints:     mountpoints,
	}, nil
}

func extractDiskIdentifiers(plistData []byte) []string {
	var ids []string
	decoder := xml.NewDecoder(bytes.NewReader(plistData))
	var inDisksArray bool

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "key" {
				var keyName string
				_ = decoder.DecodeElement(&keyName, &t)
				if keyName == "WholeDisks" {
					inDisksArray = true
				} else if inDisksArray && keyName != "WholeDisks" {
					inDisksArray = false
				}
			} else if inDisksArray && t.Name.Local == "string" {
				var diskID string
				_ = decoder.DecodeElement(&diskID, &t)
				if diskID != "" {
					ids = append(ids, diskID)
				}
			}
		}
	}
	return ids
}

func parsePlistKeyValue(data []byte) map[string]interface{} {
	result := make(map[string]interface{})
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var currentKey string

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				var k string
				_ = decoder.DecodeElement(&k, &t)
				currentKey = k
			case "string":
				var s string
				_ = decoder.DecodeElement(&s, &t)
				if currentKey != "" {
					result[currentKey] = s
				}
			case "integer":
				var i int64
				_ = decoder.DecodeElement(&i, &t)
				if currentKey != "" {
					result[currentKey] = i
				}
			case "true":
				if currentKey != "" {
					result[currentKey] = true
				}
			case "false":
				if currentKey != "" {
					result[currentKey] = false
				}
			}
		}
	}
	return result
}
