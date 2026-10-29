package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

var reader = bufio.NewReader(os.Stdin)

// Prompt asks a question on stdout with an optional default value and returns user input.
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

// SelectDevice prompts user to choose a target storage device from auto-detected ones or enter a manual path.
func SelectDevice(devs []domain.BlockDevice) (domain.BlockDevice, error) {
	if len(devs) == 0 {
		PrintWarning("No USB storage device automatically detected.")
		for {
			val, err := Prompt("Enter target device path (e.g. /dev/sdb, /dev/disk2)", "")
			if err != nil {
				return domain.BlockDevice{}, err
			}
			if val != "" {
				return domain.BlockDevice{Path: val, RawPath: val, IsRemovable: true}, nil
			}
		}
	}

	if len(devs) == 1 {
		d := devs[0]
		fmt.Println(Green(Bold("Auto-detected USB Drive:")))
		fmt.Printf("   Device: %s\n", Bold(d.Path))
		fmt.Printf("   Size:   %s\n", Cyan(d.FormattedSize()))
		if d.Model != "" || d.Vendor != "" {
			fmt.Printf("   Model:  %s %s\n", d.Vendor, d.Model)
		}
		if len(d.Mountpoints) > 0 {
			fmt.Printf("   Mounts: %s\n", Gray(strings.Join(d.Mountpoints, ", ")))
		}
		fmt.Println()

		choice, err := Prompt("Use this USB drive? [Y/n/custom]", "Y")
		if err != nil {
			return domain.BlockDevice{}, err
		}
		if strings.ToLower(choice) == "y" || strings.ToLower(choice) == "yes" {
			return d, nil
		}

		val, _ := Prompt("Enter custom device path (e.g. /dev/sdb)", "")
		return domain.BlockDevice{Path: val, RawPath: val, IsRemovable: true}, nil
	}

	fmt.Println(Bold("Multiple USB drives detected. Please select one:"))
	for i, d := range devs {
		fmt.Printf("  %s %s\n", Cyan(fmt.Sprintf("[%d]", i+1)), d.DisplayString())
	}
	fmt.Printf("  %s Enter custom device path\n\n", Cyan("[C]"))

	for {
		choice, err := Prompt("Select USB drive", "1")
		if err != nil {
			return domain.BlockDevice{}, err
		}
		if strings.EqualFold(choice, "c") {
			val, _ := Prompt("Enter custom device path (e.g. /dev/sdb)", "")
			return domain.BlockDevice{Path: val, RawPath: val, IsRemovable: true}, nil
		}
		idx := 0
		if _, scanErr := fmt.Sscanf(choice, "%d", &idx); scanErr == nil && idx >= 1 && idx <= len(devs) {
			return devs[idx-1], nil
		}
		PrintError("Invalid choice. Please select a valid number or 'C'.")
	}
}

// SelectImage prompts user to choose an image from discovered ones or enter a manual file path.
func SelectImage(images []domain.ImageMetadata) (string, error) {
	if len(images) == 0 {
		PrintWarning("No disk images found automatically in Downloads or Desktop.")
		for {
			path, err := Prompt("Enter full path to ISO / image file", "")
			if err != nil {
				return "", err
			}
			if _, statErr := os.Stat(path); statErr == nil {
				return path, nil
			}
			PrintError("File not found! Please check the path and try again.")
		}
	}

	if len(images) == 1 {
		img := images[0]
		fmt.Println(Green(Bold("Auto-detected ISO File:")))
		fmt.Printf("   Name: %s\n", Bold(img.Name))
		fmt.Printf("   Size: %s\n", Cyan(img.FormattedSize()))
		fmt.Printf("   Path: %s\n\n", Gray(img.Path))

		choice, err := Prompt("Use this ISO file? [Y/n/custom]", "Y")
		if err != nil {
			return "", err
		}
		if strings.ToLower(choice) == "y" || strings.ToLower(choice) == "yes" {
			return img.Path, nil
		}

		for {
			path, _ := Prompt("Enter path to custom ISO file", "")
			if _, statErr := os.Stat(path); statErr == nil {
				return path, nil
			}
			PrintError("File not found! Please enter a valid path.")
		}
	}

	fmt.Println(Bold("Auto-detected ISO files:"))
	for i, img := range images {
		fmt.Printf("  %s %s %s %s\n",
			Cyan(fmt.Sprintf("[%d]", i+1)),
			Bold(img.Name),
			Cyan(fmt.Sprintf("(%s)", img.FormattedSize())),
			Gray(img.Path),
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
				path, _ := Prompt("Enter path to custom ISO file", "")
				if _, statErr := os.Stat(path); statErr == nil {
					return path, nil
				}
				PrintError("File not found! Please enter a valid path.")
			}
		}

		idx, err := strconv.Atoi(choice)
		if err == nil && idx >= 1 && idx <= len(images) {
			return images[idx-1].Path, nil
		}
		PrintError("Invalid choice. Please enter a valid number or 'C'.")
	}
}

// ConfirmErase asks for explicit confirmation before writing or erasing the target drive.
func ConfirmErase(diskPath, imagePath string) bool {
	fmt.Println()
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println(Red(Bold(" ⚠️  CRITICAL WARNING: ALL DATA ON THE USB WILL BE ERASED!   ⚠️ ")))
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Printf(" Target Device: %s\n", Bold(diskPath))
	if imagePath != "" {
		fmt.Printf(" Source Image:  %s\n", Bold(imagePath))
	}
	fmt.Println(Red(Bold("===============================================================")))
	fmt.Println()

	resp, err := Prompt(fmt.Sprintf("Type %s to proceed and execute operation", Bold("YES")), "NO")
	if err != nil {
		return false
	}
	return strings.ToUpper(resp) == "YES"
}
