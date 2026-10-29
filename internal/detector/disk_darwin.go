//go:build darwin

package detector

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// DetectUSBDisks queries diskutil for external USB-connected drives on macOS
func DetectUSBDisks() ([]USBDisk, error) {
	cmd := exec.Command("diskutil", "list", "external", "physical")
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("diskutil", "list")
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("diskutil failed: %w", err)
		}
	}

	var candidateDisks []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "/dev/disk") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				candidateDisks = append(candidateDisks, parts[0])
			}
		}
	}

	var results []USBDisk
	for _, diskPath := range candidateDisks {
		infoCmd := exec.Command("diskutil", "info", diskPath)
		infoOut, err := infoCmd.Output()
		if err != nil {
			continue
		}

		infoText := string(infoOut)
		// Check that device protocol is USB or external
		if !strings.Contains(infoText, "USB") && !strings.Contains(infoText, "External") {
			continue
		}

		disk := USBDisk{
			Path: diskPath,
			Name: strings.TrimPrefix(diskPath, "/dev/"),
		}

		infoScanner := bufio.NewScanner(strings.NewReader(infoText))
		for infoScanner.Scan() {
			l := strings.TrimSpace(infoScanner.Text())
			if strings.HasPrefix(l, "Disk Size:") || strings.HasPrefix(l, "Total Size:") {
				parts := strings.Split(l, ":")
				if len(parts) > 1 {
					disk.Size = strings.TrimSpace(strings.Split(parts[1], "(")[0])
				}
			}
			if strings.HasPrefix(l, "Device / Media Name:") {
				parts := strings.Split(l, ":")
				if len(parts) > 1 {
					disk.Model = strings.TrimSpace(parts[1])
				}
			}
			if strings.HasPrefix(l, "Mount Point:") {
				parts := strings.Split(l, ":")
				if len(parts) > 1 {
					mp := strings.TrimSpace(parts[1])
					if mp != "" && !strings.Contains(mp, "Not applicable") {
						disk.Mountpoints = append(disk.Mountpoints, mp)
					}
				}
			}
		}

		results = append(results, disk)
	}

	return results, nil
}
