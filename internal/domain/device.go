package domain

import (
	"fmt"
	"strings"
)

type PartitionInfo struct {
	Path        string
	Name        string
	SizeBytes   int64
	Filesystem  string
	Label       string
	Mountpoints []string
}

type BlockDevice struct {
	Path            string
	RawPath         string
	Name            string
	SizeBytes       int64
	SectorSizeBytes int64
	Vendor          string
	Model           string
	SerialNumber    string
	BusType         string
	IsRemovable     bool
	IsSystem        bool
	Mountpoints     []string
	Partitions      []PartitionInfo
}

func (d BlockDevice) FormattedSize() string {
	const unit = 1024
	b := d.SizeBytes
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func (d BlockDevice) DisplayString() string {
	info := fmt.Sprintf("%s (%s)", d.Path, d.FormattedSize())
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
