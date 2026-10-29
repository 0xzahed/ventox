package iso

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ComputeSHA256 calculates the SHA-256 digest of a disk image with context cancellation and progress reporting.
func ComputeSHA256(ctx context.Context, filePath string, onProgress func(bytesRead, totalBytes int64)) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat file for checksum: %w", err)
	}
	totalBytes := stat.Size()

	hasher := sha256.New()
	buf := make([]byte, 1024*1024) // 1 MiB buffer
	var readTotal int64

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, err := file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
			readTotal += int64(n)
			if onProgress != nil {
				onProgress(readTotal, totalBytes)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read error during checksum computation: %w", err)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
