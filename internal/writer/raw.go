package writer

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
)

const chunkBufferSize = 4 * 1024 * 1024 // 4 MiB aligned streaming chunk buffer

// ProgressCallback is invoked periodically with progress statistics.
type ProgressCallback func(bytesWritten, totalBytes int64, speedBps float64, eta time.Duration)

// StreamRawImage streams a source disk image byte-for-byte to a destination writer,
// with context cancellation, short-write detection, throughput calculation, and sync flushing.
func StreamRawImage(ctx context.Context, srcPath string, dst io.Writer, expectedSize int64, onProgress ProgressCallback) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source image %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	stat, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source image: %w", err)
	}

	totalBytes := stat.Size()
	if expectedSize > 0 && expectedSize < totalBytes {
		totalBytes = expectedSize
	}

	buf := make([]byte, chunkBufferSize)
	var writtenTotal int64
	startTime := time.Now()
	lastReportTime := startTime

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		toRead := int64(len(buf))
		if writtenTotal+toRead > totalBytes {
			toRead = totalBytes - writtenTotal
		}
		if toRead <= 0 {
			break
		}

		nRead, readErr := srcFile.Read(buf[:toRead])
		if nRead > 0 {
			nWritten, writeErr := dst.Write(buf[:nRead])
			if writeErr != nil {
				return fmt.Errorf("%w: %v", domain.ErrWriteFailed, writeErr)
			}
			if nWritten != nRead {
				return fmt.Errorf("%w: short write (expected %d bytes, wrote %d)", domain.ErrWriteFailed, nRead, nWritten)
			}

			writtenTotal += int64(nWritten)

			now := time.Now()
			if onProgress != nil && (now.Sub(lastReportTime) >= 200*time.Millisecond || writtenTotal == totalBytes) {
				elapsed := now.Sub(startTime).Seconds()
				var speed float64
				var eta time.Duration
				if elapsed > 0 {
					speed = float64(writtenTotal) / elapsed
					remainingBytes := totalBytes - writtenTotal
					if speed > 0 {
						eta = time.Duration(float64(remainingBytes)/speed) * time.Second
					}
				}
				onProgress(writtenTotal, totalBytes, speed, eta)
				lastReportTime = now
			}
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read error on source image: %w", readErr)
		}
	}

	// Flush buffers if writer supports Syncer interface
	if syncer, ok := dst.(interface{ Sync() error }); ok {
		if err := syncer.Sync(); err != nil {
			return fmt.Errorf("failed to sync device buffers: %w", err)
		}
	}

	return nil
}
