package writer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
)

// VerifyTarget streams data back from a target device and validates its SHA-256 digest against expected.
func VerifyTarget(ctx context.Context, target io.Reader, totalBytes int64, expectedSHA256 string, onProgress ProgressCallback) (*domain.VerificationReport, error) {
	if totalBytes <= 0 {
		return nil, fmt.Errorf("invalid total bytes for verification: %d", totalBytes)
	}

	hasher := sha256.New()
	buf := make([]byte, chunkBufferSize)
	var readTotal int64
	startTime := time.Now()
	lastReportTime := startTime

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		toRead := int64(len(buf))
		if readTotal+toRead > totalBytes {
			toRead = totalBytes - readTotal
		}
		if toRead <= 0 {
			break
		}

		n, err := target.Read(buf[:toRead])
		if n > 0 {
			hasher.Write(buf[:n])
			readTotal += int64(n)

			now := time.Now()
			if onProgress != nil && (now.Sub(lastReportTime) >= 200*time.Millisecond || readTotal == totalBytes) {
				elapsed := now.Sub(startTime).Seconds()
				var speed float64
				var eta time.Duration
				if elapsed > 0 {
					speed = float64(readTotal) / elapsed
					remainingBytes := totalBytes - readTotal
					if speed > 0 {
						eta = time.Duration(float64(remainingBytes)/speed) * time.Second
					}
				}
				onProgress(readTotal, totalBytes, speed, eta)
				lastReportTime = now
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read error during target verification: %w", err)
		}
	}

	actualDigest := hex.EncodeToString(hasher.Sum(nil))
	duration := time.Since(startTime)

	report := &domain.VerificationReport{
		SourceDigest: strings.ToLower(strings.TrimSpace(expectedSHA256)),
		TargetDigest: actualDigest,
		BytesChecked: readTotal,
		Duration:     duration,
	}

	if expectedSHA256 != "" {
		if !strings.EqualFold(report.SourceDigest, report.TargetDigest) {
			report.Verified = false
			return report, fmt.Errorf("%w: expected %s, got %s", domain.ErrVerificationFailed, report.SourceDigest, report.TargetDigest)
		}
	}

	report.Verified = true
	return report, nil
}
