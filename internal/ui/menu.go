package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/0xzahed/ventox/internal/detector"
)

var reader = bufio.NewReader(os.Stdin)

func Prompt(question string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", question, Cyan(defaultValue))
	} else {
		fmt.Printf("%s: ", question)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(input)
	if trimmed == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return trimmed, nil
}

// SelectUSB prompts user to choose a USB drive from auto-detected ones or manual path
func SelectUSB(disks []detector.USBDisk) (detector.USBDisk, error) {
	if len(disks) == 0 {
		PrintWarning("No USB flash drive automatically detected.")
		for {
			val, err := Prompt("Please insert a USB drive and type 'R' to refresh, or enter device path (e.g. /dev/sda)", "R")
			if err != nil {
				return detector.USBDisk{}, err
			}
			if strings.EqualFold(val, "R") {
				refreshed, err := detector.DetectUSBDisks()
				if err == nil && len(refreshed) > 0 {
					return SelectUSB(refreshed)
				}
				PrintWarning("Still no USB detected.")
				continue
			}

			// Manual device entered
			if strings.HasPrefix(val, "/dev/") {
				return detector.USBDisk{
					Path: val,
					Size: "Unknown",
				}, nil
			}
			PrintError("Invalid device path. Must start with /dev/ (e.g. /dev/sda)")
		}
	}

	if len(disks) == 1 {
		disk := disks[0]
		fmt.Println(Green(Bold("Auto-detected USB Drive:")))
		fmt.Printf("   Device: %s\n", Bold(disk.Path))
		fmt.Printf("   Size:   %s\n", Cyan(disk.Size))
		if disk.Vendor != "" || disk.Model != "" {
			fmt.Printf("   Model:  %s %s\n", disk.Vendor, disk.Model)
		}
		if len(disk.Mountpoints) > 0 {
			fmt.Printf("   Mounts: %s\n", Gray(strings.Join(disk.Mountpoints, ", ")))
		}
		fmt.Println()

		choice, err := Prompt("Use this USB drive? [Y/n/custom]", "Y")
		if err != nil {
			return detector.USBDisk{}, err
		}
		choice = strings.ToLower(choice)
		if choice == "y" || choice == "yes" {
			return disk, nil
		}

		if choice == "c" || choice == "custom" || choice == "n" || choice == "no" {
			val, _ := Prompt("Enter custom device path (e.g. /dev/sdb)", "")
			return detector.USBDisk{Path: val, Size: "Custom"}, nil
		}
		return disk, nil
	}

	// Multiple disks
	fmt.Println(Bold("Multiple USB drives detected. Please select one:"))
	for i, d := range disks {
		fmt.Printf("  %s %s\n", Cyan(fmt.Sprintf("[%d]", i+1)), d.DisplayString())
	}
	fmt.Printf("  %s Enter custom device path\n\n", Cyan("[C]"))

	for {
		choice, err := Prompt("Select USB drive", "1")
		if err != nil {
			return detector.USBDisk{}, err
		}

		if strings.EqualFold(choice, "c") {
			val, _ := Prompt("Enter custom device path (e.g. /dev/sdb)", "")
			return detector.USBDisk{Path: val, Size: "Custom"}, nil
		}

		idx, err := strconv.Atoi(choice)
		if err == nil && idx >= 1 && idx <= len(disks) {
			return disks[idx-1], nil
		}
		PrintError("Invalid choice. Please enter a valid number or 'C'.")
	}
}

// SelectISO prompts user to select an ISO file from auto-detected ones or manual path
func SelectISO(isos []detector.ISOFile) (string, error) {
	if len(isos) == 0 {
		PrintWarning("No .iso files found in Downloads or Desktop.")
		for {
			path, err := Prompt("Please enter the full path to your .iso file", "")
			if err != nil {
				return "", err
			}
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
			PrintError("File not found! Please check the path and try again.")
		}
	}

	if len(isos) == 1 {
		iso := isos[0]
		fmt.Println(Green(Bold("Auto-detected ISO File:")))
		fmt.Printf("   Name: %s\n", Bold(iso.Name))
		fmt.Printf("   Size: %s\n", Cyan(iso.FormattedSize))
		fmt.Printf("   Path: %s\n\n", Gray(iso.Path))

		choice, err := Prompt("Use this ISO file? [Y/n/custom]", "Y")
		if err != nil {
			return "", err
		}
		choice = strings.ToLower(choice)
		if choice == "y" || choice == "yes" {
			return iso.Path, nil
		}

		// Prompt custom
		for {
			path, _ := Prompt("Enter path to custom .iso file", "")
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
			PrintError("File not found! Please enter a valid path.")
		}
	}

	// Multiple ISOs found
	fmt.Println(Bold("Auto-detected ISO files:"))
	for i, iso := range isos {
		fmt.Printf("  %s %s %s %s\n",
			Cyan(fmt.Sprintf("[%d]", i+1)),
			Bold(iso.Name),
			Cyan(fmt.Sprintf("(%s)", iso.FormattedSize)),
			Gray(iso.Path),
		)
	}
	fmt.Printf("  %s Enter custom ISO path\n\n", Cyan("[C]"))

	for {
		choice, err := Prompt("Select ISO file", "1")
		if err != nil {
			return "", err
		}

		if strings.EqualFold(choice, "c") {
			for {
				path, _ := Prompt("Enter path to custom .iso file", "")
				if _, err := os.Stat(path); err == nil {
					return path, nil
				}
				PrintError("File not found! Please enter a valid path.")
			}
		}

		idx, err := strconv.Atoi(choice)
		if err == nil && idx >= 1 && idx <= len(isos) {
			return isos[idx-1].Path, nil
		}
		PrintError("Invalid choice. Please enter a valid number or 'C'.")
	}
}

// ConfirmErase asks for explicit confirmation before erasing the target USB drive
func ConfirmErase(diskPath, isoPath string) bool {
	fmt.Println()
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println(Red(Bold(" ⚠️  CRITICAL WARNING: ALL DATA ON THE USB WILL BE ERASED!   ⚠️ ")))
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Printf(" Target Device: %s\n", Bold(diskPath))
	fmt.Printf(" Source ISO:    %s\n", Bold(isoPath))
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println()

	resp, err := Prompt(fmt.Sprintf("Type %s to proceed and write bootable image", Bold("YES")), "NO")
	if err != nil {
		return false
	}
	return strings.ToUpper(resp) == "YES"
}

type FormatChoice struct {
	ID          int
	Title       string
	Description string
}

// SelectFormatMode prompts user to choose format mode
func SelectFormatMode(choices []FormatChoice) (FormatChoice, error) {
	fmt.Println(Bold("Select Format Target & Partition Style:"))
	for _, c := range choices {
		fmt.Printf("  %s %s\n     %s\n",
			Cyan(fmt.Sprintf("[%d]", c.ID)),
			Bold(c.Title),
			Gray(c.Description),
		)
	}
	fmt.Println()

	for {
		choice, err := Prompt("Select mode [1-6]", "1")
		if err != nil {
			return FormatChoice{}, err
		}
		idx, err := strconv.Atoi(choice)
		if err == nil {
			for _, c := range choices {
				if c.ID == idx {
					return c, nil
				}
			}
		}
		PrintError("Invalid choice. Please select a number from 1 to 6.")
	}
}

// ConfirmFormat asks for explicit confirmation before formatting a disk
func ConfirmFormat(diskPath, formatTitle, label string) bool {
	fmt.Println()
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println(Red(Bold(" ⚠️  CRITICAL WARNING: ALL DATA ON THE USB WILL BE ERASED!   ⚠️ ")))
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Printf(" Target Device: %s\n", Bold(diskPath))
	fmt.Printf(" Format Mode:   %s\n", Bold(formatTitle))
	fmt.Printf(" Volume Label:  %s\n", Bold(label))
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println()

	resp, err := Prompt(fmt.Sprintf("Type %s to format and wipe drive", Bold("YES")), "NO")
	if err != nil {
		return false
	}
	return strings.ToUpper(resp) == "YES"
}
