package engine

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/detector"
	"github.com/0xzahed/ventox/internal/ui"
)

const defaultVentoyURL = "https://github.com/ventoy/Ventoy/releases/download/v1.1.18/ventoy-1.1.18-linux.tar.gz"

// LocateVentoy locates or automatically downloads the Ventoy toolset.
func LocateVentoy() (string, error) {
	home := detector.GetRealUserHome()
	candidates := []string{
		filepath.Join(home, "Downloads", "ventoy-1.1.18", "Ventoy2Disk.sh"),
		"/opt/ventoy/Ventoy2Disk.sh",
		filepath.Join(home, ".cache", "ventox", "ventoy-1.1.18", "Ventoy2Disk.sh"),
		"/root/.cache/ventox/ventoy-1.1.18/Ventoy2Disk.sh",
	}

	// Also check relative to executable
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "ventoy", "Ventoy2Disk.sh"))
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "..", "ventoy", "Ventoy2Disk.sh"))
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	// Auto-download Ventoy if not found
	ui.PrintInfo("Ventoy not found locally. Downloading portable Ventoy (19 MB)...")
	cacheDir := filepath.Join(home, ".cache", "ventox")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	tarPath := filepath.Join(cacheDir, "ventoy.tar.gz")
	if err := downloadFile(defaultVentoyURL, tarPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	if err := extractTarGz(tarPath, cacheDir); err != nil {
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	ventoyScript := filepath.Join(cacheDir, "ventoy-1.1.18", "Ventoy2Disk.sh")
	if _, err := os.Stat(ventoyScript); err != nil {
		return "", fmt.Errorf("Ventoy2Disk.sh missing after extraction")
	}

	_ = os.Chmod(ventoyScript, 0755)
	return ventoyScript, nil
}

// InstallVentoy writes the Ventoy UEFI (GPT) bootloader to target device.
func InstallVentoy(scriptPath, devicePath string) error {
	// First unmount all existing mounts
	if err := UnmountDevice(devicePath); err != nil {
		return err
	}

	cmd := exec.Command(scriptPath, "-I", "-g", devicePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ventoy installation error: %w", err)
	}

	// Trigger kernel to reload partition table
	time.Sleep(2 * time.Second)
	_ = exec.Command("partprobe", devicePath).Run()
	time.Sleep(1 * time.Second)

	return nil
}

// GetVentoyDataPartition returns partition 1 of the device (exFAT partition).
func GetVentoyDataPartition(devicePath string) string {
	if strings.Contains(devicePath, "nvme") || strings.Contains(devicePath, "mmcblk") {
		return devicePath + "p1"
	}
	return devicePath + "1"
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(gzipStreamPath, dest string) error {
	f, err := os.Open(gzipStreamPath)
	if err != nil {
		return err
	}
	defer f.Close()

	uncompressedStream, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}
