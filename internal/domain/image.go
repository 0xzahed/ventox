package domain

import (
	"fmt"
	"time"
)

type ImageFormat string

const (
	FormatISO     ImageFormat = "ISO9660"
	FormatRawIMG  ImageFormat = "RawDiskImage"
	FormatUnknown ImageFormat = "Unknown"
)

type ImageClassification string

const (
	ClassWindowsInstaller ImageClassification = "WindowsInstaller"
	ClassLinuxHybrid      ImageClassification = "LinuxHybrid"
	ClassRawBootable      ImageClassification = "RawBootable"
	ClassUnknown          ImageClassification = "Unknown"
)

type ImageMetadata struct {
	Path           string
	Name           string
	SizeBytes      int64
	ModTime        time.Time
	Format         ImageFormat
	Classification ImageClassification
	VolumeLabel    string
	IsEFI          bool
	HasLargeWIM    bool
	WIMPath        string
	WIMSizeBytes   int64
	SHA256Checksum string
}

func (img ImageMetadata) FormattedSize() string {
	const unit = 1024
	b := img.SizeBytes
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
