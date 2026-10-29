package copier

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/ui"
)

type ProgressWriter struct {
	TotalBytes   int64
	CopiedBytes  int64
	StartTime    time.Time
	LastUpdate   time.Time
	lastPrintLen int
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.CopiedBytes += int64(n)

	now := time.Now()
	// Update terminal at most every 150ms to keep it smooth without flickering
	if now.Sub(pw.LastUpdate) >= 150*time.Millisecond || pw.CopiedBytes == pw.TotalBytes {
		pw.renderProgress(now)
		pw.LastUpdate = now
	}

	return n, nil
}

func (pw *ProgressWriter) renderProgress(now time.Time) {
	elapsed := now.Sub(pw.StartTime).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}

	speed := float64(pw.CopiedBytes) / elapsed // bytes/sec
	speedMB := speed / (1024 * 1024)

	percent := float64(pw.CopiedBytes) / float64(pw.TotalBytes) * 100
	if percent > 100 {
		percent = 100
	}

	var etaStr string
	if speed > 0 && pw.CopiedBytes < pw.TotalBytes {
		remainingBytes := pw.TotalBytes - pw.CopiedBytes
		remainingSecs := int(float64(remainingBytes) / speed)
		mins := remainingSecs / 60
		secs := remainingSecs % 60
		etaStr = fmt.Sprintf("ETA: %02dm %02ds", mins, secs)
	} else {
		etaStr = "Finalizing..."
	}

	// 25-character wide progress bar
	const barWidth = 25
	filled := int(percent / 100 * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	status := fmt.Sprintf("\r%s [%s] %s %s | %s | %s",
		ui.Cyan(ui.Bold("Copying:")),
		ui.Green(bar),
		ui.Yellow(fmt.Sprintf("%5.1f%%", percent)),
		ui.Gray(fmt.Sprintf("(%.2f GB / %.2f GB)",
			float64(pw.CopiedBytes)/(1024*1024*1024),
			float64(pw.TotalBytes)/(1024*1024*1024))),
		ui.Cyan(fmt.Sprintf("%.1f MB/s", speedMB)),
		ui.Magenta(etaStr),
	)

	fmt.Print(status)
}

// CopyWithProgress copies src file to dest file with a live terminal progress indicator
func CopyWithProgress(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source ISO: %w", err)
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source ISO: %w", err)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	pw := &ProgressWriter{
		TotalBytes: srcStat.Size(),
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}

	// 1MB buffer for fast sequential USB write
	buf := make([]byte, 1024*1024)
	if _, err := io.CopyBuffer(destFile, io.TeeReader(srcFile, pw), buf); err != nil {
		return fmt.Errorf("error during copy: %w", err)
	}

	fmt.Println() // Move to next line after progress bar finishes

	ui.PrintInfo("Flushing filesystem cache (sync) to ensure USB integrity...")
	_ = destFile.Sync()

	syncCmd := exec.Command("sync")
	_ = syncCmd.Run()

	ui.PrintSuccess("All data flushed to USB successfully.")
	return nil
}
