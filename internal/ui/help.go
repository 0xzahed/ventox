package ui

import (
	"fmt"
	"runtime"
)

const (
	Version   = "v0.2.0"
	GitCommit = "2068c47"
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

	fmt.Println(Bold("EXAMPLES:"))
	fmt.Printf("  %s %s\n", Gray("$"), "ventox          # Create bootable Windows 11 or Linux USB")
	fmt.Printf("  %s %s\n", Gray("$"), "ventox format   # Select format mode (Ventoy GPT/MBR, exFAT, FAT32, NTFS, ext4)")
	fmt.Printf("  %s %s\n", Gray("$"), "ventox list     # Quick scan of connected USBs and ISOs")
	fmt.Printf("  %s %s\n\n", Gray("$"), "ventox version  # View version info")

	fmt.Printf("For documentation and contributions, visit: %s\n\n", Cyan(RepoURL))
}
