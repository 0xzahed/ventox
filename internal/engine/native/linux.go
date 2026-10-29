package native

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/0xzahed/ventox/internal/copier"
	"github.com/0xzahed/ventox/internal/engine"
	"github.com/0xzahed/ventox/internal/ui"
)

// WriteLinuxUSB writes a Linux hybrid ISO directly to the raw block device (DD mode)
// with live speed, byte-accurate progress bar, and cache synchronization.
// Fully cross-platform across Linux and macOS (/dev/rdiskX).
func WriteLinuxUSB(diskPath, isoPath string) error {
	ui.PrintInfo("Unmounting target USB device partitions...")
	if err := engine.UnmountDevice(diskPath); err != nil {
		return err
	}

	srcFile, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("failed to open ISO image: %w", err)
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat ISO image: %w", err)
	}

	rawPath := GetRawDevicePath(diskPath)
	destDev, err := os.OpenFile(rawPath, os.O_WRONLY, 0660)
	if err != nil {
		return fmt.Errorf("failed to open raw block device %s: %w", rawPath, err)
	}
	defer destDev.Close()

	pw := &copier.ProgressWriter{
		TotalBytes: srcStat.Size(),
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	ui.PrintInfo(fmt.Sprintf("Flashing raw hybrid Linux image to %s...", rawPath))

	// 4MB streaming buffer
	buf := make([]byte, 4*1024*1024)
	if _, err := io.CopyBuffer(destDev, io.TeeReader(srcFile, pw), buf); err != nil {
		return fmt.Errorf("error during raw block write: %w", err)
	}

	fmt.Println()
	ui.PrintInfo("Flushing filesystem cache (sync)...")
	_ = destDev.Sync()
	_ = exec.Command("sync").Run()

	ui.PrintInfo("Finalizing disk partition tables...")
	time.Sleep(1 * time.Second)
	PostRawWriteCleanup(diskPath)

	return nil
}
