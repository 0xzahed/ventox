package ui

import (
	"fmt"
	"runtime"
)

const (
	Version   = "v0.3.0"
	GitCommit = "46a8f83"
	Author    = "Abu Zahed (0xzahed)"
	RepoURL   = "https://github.com/0xzahed/ventox"
)

func PrintVersion() {
	Banner()
	fmt.Printf("%s %s\n", Bold("Ventox Version:"), Cyan(Version))
	fmt.Printf("%s  %s\n", Bold("Git Commit:    "), Gray(GitCommit))
	fmt.Printf("%s  %s (%s/%s)\n", Bold("Go Runtime:    "), runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("%s      %s\n", Bold("Author:        "), Author)
	fmt.Printf("%s     %s\n", Bold("Repository:    "), Cyan(RepoURL))
	fmt.Printf("%s     %s\n\n", Bold("License:       "), Green("MIT"))
}

func PrintHelp() {
	Banner()
	fmt.Println(Bold("USAGE:"))
	fmt.Printf("  %s %s\n\n", Cyan("ventox"), Gray("[command] [flags]"))

	fmt.Println(Bold("COMMANDS:"))
	fmt.Printf("  %s          %s\n", Cyan(Bold("(default)    ")), "Launch the interactive bootable USB creation wizard")
	fmt.Printf("  %s, %s   %s\n", Cyan(Bold("format")), Cyan(Bold("wipe ")), "Format or restore a USB drive (GPT/MBR, exFAT, FAT32, NTFS, ext4)")
	fmt.Printf("  %s, %s       %s\n", Cyan(Bold("list  ")), Cyan(Bold("ls   ")), "List all detected USB drives and scanned ISO images")
	fmt.Printf("  %s, %s  %s\n", Cyan(Bold("version")), Cyan(Bold("-v   ")), "Print version and system runtime information")
	fmt.Printf("  %s, %s     %s\n\n", Cyan(Bold("help   ")), Cyan(Bold("-h   ")), "Show this help screen")

	fmt.Println(Bold("FLAGS:"))
	fmt.Printf("  %s         %s\n", Cyan("--dry-run"), "Run preflight checks and plan without writing to target disk")
	fmt.Printf("  %s          %s\n", Cyan("--verify"), "Perform post-write SHA-256 readback checksum verification")
	fmt.Printf("  %s    %s\n", Cyan("--bypass-win11"), "Enable Windows 11 hardware bypasses (TPM, Secure Boot, RAM, NRO)")
	fmt.Printf("  %s   %s\n", Cyan("--device <path>"), "Specify target USB device directly (e.g., /dev/sdb, /dev/disk2)")
	fmt.Printf("  %s      %s\n", Cyan("--iso <path>"), "Specify source ISO / disk image directly")
	fmt.Printf("  %s       %s\n", Cyan("--fs <type>"), "Target filesystem for formatting: exfat, fat32, ntfs, ext4, apfs")
	fmt.Printf("  %s     %s\n", Cyan("--label <name>"), "Custom filesystem volume label")
	fmt.Printf("  %s, %s        %s\n", Cyan("-y"), Cyan("--yes"), "Skip interactive confirmation prompts")
	fmt.Printf("  %s            %s\n\n", Cyan("--json"), "Output plan and operation journals as structured JSON")

	fmt.Println(Bold("EXAMPLES:"))
	fmt.Printf("  %s %s\n", Gray("$"), "ventox                                         # Interactive boot creator")
	fmt.Printf("  %s %s\n", Gray("$"), "ventox --iso win11.iso --device /dev/sdb       # Deploy Windows 11 installer")
	fmt.Printf("  %s %s\n", Gray("$"), "ventox --dry-run --iso win11.iso               # Simulate without writing")
	fmt.Printf("  %s %s\n", Gray("$"), "ventox format --device /dev/sdb --fs exfat     # Format USB as exFAT")
	fmt.Printf("  %s %s\n", Gray("$"), "ventox list --json                             # Output USB/ISO list in JSON")
	fmt.Printf("  %s %s\n\n", Gray("$"), "ventox version                                 # View version info")

	fmt.Printf("For documentation and contributions, visit: %s\n\n", Cyan(RepoURL))
}
