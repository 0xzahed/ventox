package detector

import (
	"fmt"
	"strings"
)

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
	var details []string
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
