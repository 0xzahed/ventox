package writer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
)

func TestStreamRawImage(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.raw")
	data := []byte("Ventox native streaming raw image test chunk data")
	if err := os.WriteFile(srcFile, data, 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	var buf bytes.Buffer
	var lastBytes int64
	err := StreamRawImage(context.Background(), srcFile, &buf, int64(len(data)), func(written, total int64, speed float64, eta time.Duration) {
		lastBytes = written
	})
	if err != nil {
		t.Fatalf("StreamRawImage failed: %v", err)
	}

	if buf.String() != string(data) {
		t.Errorf("streamed content mismatch: got %s, want %s", buf.String(), string(data))
	}
	if lastBytes != int64(len(data)) {
		t.Errorf("last reported bytes mismatch: got %d, want %d", lastBytes, len(data))
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var cancelBuf bytes.Buffer
	err = StreamRawImage(ctx, srcFile, &cancelBuf, int64(len(data)), nil)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestVerifyTarget(t *testing.T) {
	data := []byte("Verification test payload bytes")
	hasher := sha256.New()
	hasher.Write(data)
	correctHash := hex.EncodeToString(hasher.Sum(nil))

	report, err := VerifyTarget(context.Background(), bytes.NewReader(data), int64(len(data)), correctHash, nil)
	if err != nil {
		t.Fatalf("VerifyTarget failed on valid hash: %v", err)
	}
	if !report.Verified {
		t.Errorf("expected report.Verified to be true")
	}

	// Test mismatch
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
	report, err = VerifyTarget(context.Background(), bytes.NewReader(data), int64(len(data)), wrongHash, nil)
	if err == nil {
		t.Errorf("expected error on hash mismatch, got nil")
	}
	if report.Verified {
		t.Errorf("expected report.Verified to be false on hash mismatch")
	}
}

func TestGenerateAutoUnattendXML(t *testing.T) {
	opts := DefaultWindows11Bypass()
	xml := GenerateAutoUnattendXML(opts)

	expectedKeys := []string{
		"BypassTPMCheck",
		"BypassSecureBootCheck",
		"BypassRAMCheck",
		"BypassCPUCheck",
		"BypassStorageCheck",
		"BypassNRO",
	}

	for _, k := range expectedKeys {
		if !strings.Contains(xml, k) {
			t.Errorf("autounattend.xml missing bypass key: %s", k)
		}
	}

	if !strings.Contains(xml, "<unattend") || !strings.Contains(xml, "</unattend>") {
		t.Errorf("malformed XML root tags in autounattend.xml")
	}
}

func TestValidateFilesystemSupport(t *testing.T) {
	// FAT32 and exFAT checks
	_ = ValidateFilesystemSupport(domain.FSFAT32)
	_ = ValidateFilesystemSupport(domain.FSExFAT)

	// Check unsupported FS error handling
	err := ValidateFilesystemSupport(domain.TargetFilesystem("NON_EXISTENT_FS"))
	if err == nil {
		t.Errorf("expected error for unsupported filesystem, got nil")
	}
}
