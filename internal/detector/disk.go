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

type USBDisk struct {
	Path        string
	Name        string
	Size        string
	Model       string
	Vendor      string
	Partitions  []string
	Mountpoints []string
}

func (d USBDisk) DisplayString() string {
	info := fmt.Sprintf("%s (%s)", d.Path, d.Size)
	details := []string{}
	if strings.TrimSpace(d.Vendor) != "" {
		details = append(details, strings.TrimSpace(d.Vendor))
	}
	if strings.TrimSpace(d.Model) != "" {
		details = append(details, strings.TrimSpace(d.Model))
	}
	if len(details) > 0 {
		info += " - " + strings.Join(details, " ")
	}
	if len(d.Mountpoints) > 0 {
		info += fmt.Sprintf(" [Mounted: %s]", strings.Join(d.Mountpoints, ", "))
	}
	return info
}

// DetectUSBDisks queries lsblk for USB-connected block devices
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
		// Only select disks with USB transport
		if strings.ToLower(dev.Type) == "disk" && strings.ToLower(dev.Tran) == "usb" {
			disk := USBDisk{
				Path:   dev.Path,
				Name:   dev.Name,
				Size:   dev.Size,
				Model:  strings.TrimSpace(dev.Model),
				Vendor: strings.TrimSpace(dev.Vendor),
			}

			// Collect mountpoints and partitions
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
