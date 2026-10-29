package engine

type FormatMode int

const (
	ModeVentoyGPT FormatMode = iota + 1
	ModeVentoyMBR
	ModeCleanExFAT
	ModeCleanFAT32
	ModeCleanNTFS
	ModeCleanEXT4
)

type FormatOption struct {
	Mode        FormatMode
	Title       string
	Description string
}

func GetFormatOptions() []FormatOption {
	return []FormatOption{
		{
			Mode:        ModeVentoyGPT,
			Title:       "Ventoy Bootable (GPT / UEFI)",
			Description: "Recommended for modern PCs, Windows 11 & UEFI systems",
		},
		{
			Mode:        ModeVentoyMBR,
			Title:       "Ventoy Bootable (MBR / Legacy BIOS)",
			Description: "For older PCs and maximum backwards compatibility",
		},
		{
			Mode:        ModeCleanExFAT,
			Title:       "Clean Standard USB (exFAT)",
			Description: "Universal storage for Windows/Mac/Linux (Supports files > 4GB)",
		},
		{
			Mode:        ModeCleanFAT32,
			Title:       "Clean Standard USB (FAT32)",
			Description: "Traditional USB storage (File size limited to 4GB)",
		},
		{
			Mode:        ModeCleanNTFS,
			Title:       "Clean Standard USB (NTFS)",
			Description: "Native Windows filesystem with permissions & journaling",
		},
		{
			Mode:        ModeCleanEXT4,
			Title:       "Clean Standard USB (ext4 / APFS)",
			Description: "High-performance Unix/Linux native filesystem",
		},
	}
}
