package domain

type CreationMode string

const (
	ModeRawWrite           CreationMode = "RawImageStream"
	ModeDualPartitionUEFI  CreationMode = "DualPartitionUEFI"
	ModeISOExtraction      CreationMode = "ISOExtraction"
	ModeFormatOnly         CreationMode = "FormatOnly"
)

type PartitionScheme string

const (
	SchemeGPT PartitionScheme = "GPT"
	SchemeMBR PartitionScheme = "MBR"
)

type TargetFilesystem string

const (
	FSExFAT TargetFilesystem = "exFAT"
	FSFAT32 TargetFilesystem = "FAT32"
	FSNTFS  TargetFilesystem = "NTFS"
	FSEXT4  TargetFilesystem = "ext4"
	FSAPFS  TargetFilesystem = "APFS"
)

type WritePlan struct {
	ID               string
	Target           BlockDevice
	Source           *ImageMetadata
	Mode             CreationMode
	Scheme           PartitionScheme
	Filesystem       TargetFilesystem
	VolumeLabel      string
	BypassWindows11  bool
	VerifyAfterWrite bool
	DryRun           bool
	EstimatedBytes   int64
	Steps            []string
}
