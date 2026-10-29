//go:build linux

package detector

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type rawBlockDevice struct {
	Name        string           `json:"name"`
	Path        string           `json:"path"`
	Size        string           `json:"size"`
	Type        string           `json:"type"`
	Tran        string           `json:"tran"`
	Model       string           `json:"model"`
	Vendor      string           `json:"vendor"`
	Label       string           `json:"label"`
	Mountpoints []string         `json:"mountpoints"`
	Children    []rawBlockDevice `json:"children"`
}

type lsblkOutput struct {
	Blockdevices []rawBlockDevice `json:"blockdevices"`
}

// DetectUSBDisks queries lsblk for USB-connected block devices on Linux
func DetectUSBDisks() ([]USBDisk, error) {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,PATH,SIZE,TYPE,TRAN,MODEL,VENDOR,LABEL,MOUNTPOINTS")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk execution failed: %w", err)
	}

	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	var results []USBDisk
	for _, dev := range parsed.Blockdevices {
		if strings.ToLower(dev.Type) == "disk" && strings.ToLower(dev.Tran) == "usb" {
			disk := USBDisk{
				Path:   dev.Path,
				Name:   dev.Name,
				Size:   dev.Size,
				Model:  strings.TrimSpace(dev.Model),
				Vendor: strings.TrimSpace(dev.Vendor),
			}

			for _, m := range dev.Mountpoints {
				if m != "" {
					disk.Mountpoints = append(disk.Mountpoints, m)
				}
			}

			for _, child := range dev.Children {
				disk.Partitions = append(disk.Partitions, child.Path)
				for _, cm := range child.Mountpoints {
					if cm != "" {
						disk.Mountpoints = append(disk.Mountpoints, cm)
					}
				}
			}

			results = append(results, disk)
		}
	}

	return results, nil
}
