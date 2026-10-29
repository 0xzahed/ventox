package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xzahed/ventox/internal/device/fake"
	"github.com/0xzahed/ventox/internal/domain"
)

func TestCreatePlanWindowsInstaller(t *testing.T) {
	target := domain.BlockDevice{
		Path:        "/dev/sdb",
		SizeBytes:   64 * 1024 * 1024 * 1024, // 64 GiB
		IsRemovable: true,
	}
	src := &domain.ImageMetadata{
		Path:           "/home/user/Downloads/Win11_23H2.iso",
		Name:           "Win11_23H2.iso",
		SizeBytes:      5500 * 1024 * 1024,
		Format:         domain.FormatISO,
		Classification: domain.ClassWindowsInstaller,
		IsEFI:          true,
		HasLargeWIM:    true,
		WIMPath:        "sources/install.wim",
		VolumeLabel:    "CCCOMA_X64FRE_EN-US_DV9",
	}

	plan, err := CreatePlan(PlanOptions{
		Target:           target,
		Source:           src,
		BypassWindows11:  true,
		VerifyAfterWrite: true,
	})
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if plan.Mode != domain.ModeDualPartitionUEFI {
		t.Errorf("expected ModeDualPartitionUEFI for large WIM, got: %s", plan.Mode)
	}
	if plan.Scheme != domain.SchemeGPT {
		t.Errorf("expected SchemeGPT, got: %s", plan.Scheme)
	}
	if plan.Filesystem != domain.FSNTFS {
		t.Errorf("expected FSNTFS, got: %s", plan.Filesystem)
	}
	if !plan.BypassWindows11 {
		t.Errorf("expected BypassWindows11 to be true")
	}
}

func TestCreatePlanLinuxRawStream(t *testing.T) {
	target := domain.BlockDevice{
		Path:        "/dev/sdb",
		SizeBytes:   32 * 1024 * 1024 * 1024,
		IsRemovable: true,
	}
	src := &domain.ImageMetadata{
		Path:           "/home/user/Downloads/ubuntu-24.04-desktop-amd64.iso",
		Name:           "ubuntu-24.04-desktop-amd64.iso",
		SizeBytes:      4 * 1024 * 1024 * 1024,
		Format:         domain.FormatISO,
		Classification: domain.ClassLinuxHybrid,
		VolumeLabel:    "Ubuntu 24.04 LTS amd64",
	}

	plan, err := CreatePlan(PlanOptions{
		Target:           target,
		Source:           src,
		VerifyAfterWrite: true,
	})
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if plan.Mode != domain.ModeRawWrite {
		t.Errorf("expected ModeRawWrite for Linux hybrid, got: %s", plan.Mode)
	}
}

func TestCreatePlanFormatOnly(t *testing.T) {
	target := domain.BlockDevice{
		Path:        "/dev/sdb",
		SizeBytes:   16 * 1024 * 1024 * 1024,
		IsRemovable: true,
	}

	plan, err := CreatePlan(PlanOptions{
		Target:          target,
		Source:          nil,
		RequestedFS:     domain.FSExFAT,
		RequestedScheme: domain.SchemeGPT,
		VolumeLabel:     "MY_STORAGE",
	})
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if plan.Mode != domain.ModeFormatOnly {
		t.Errorf("expected ModeFormatOnly, got: %s", plan.Mode)
	}
	if plan.Filesystem != domain.FSExFAT {
		t.Errorf("expected FSExFAT, got: %s", plan.Filesystem)
	}
	if plan.VolumeLabel != "MY_STORAGE" {
		t.Errorf("expected volume label MY_STORAGE, got: %s", plan.VolumeLabel)
	}
}

func TestExecutorDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "test.iso")
	_ = os.WriteFile(srcFile, []byte("fake image data"), 0644)

	target := domain.BlockDevice{
		Path:        "/dev/sdb",
		SizeBytes:   16 * 1024 * 1024 * 1024,
		IsRemovable: true,
	}
	src := &domain.ImageMetadata{
		Path:           srcFile,
		SizeBytes:      15,
		Classification: domain.ClassLinuxHybrid,
	}

	plan, err := CreatePlan(PlanOptions{
		Target: target,
		Source: src,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	var progressCalled bool
	executor := NewExecutor()
	journal, err := executor.Execute(context.Background(), plan, func(event ProgressEvent) {
		progressCalled = true
	})
	if err != nil {
		t.Fatalf("Dry-run execute failed: %v", err)
	}

	if !progressCalled {
		t.Errorf("expected progress reporter to be called during dry-run")
	}
	if len(journal.Entries) == 0 {
		t.Errorf("expected journal entries in dry-run")
	}
	for _, entry := range journal.Entries {
		if entry.Status != domain.StatusCompleted {
			t.Errorf("entry %s had status %s, expected Completed", entry.Name, entry.Status)
		}
	}
}

func TestExecutorSimulatedDiskStreamingAndVerification(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "boot.iso")
	payload := []byte("Ventox native streaming verification test payload 1234567890")
	if err := os.WriteFile(srcFile, payload, 0644); err != nil {
		t.Fatalf("failed to write test ISO: %v", err)
	}

	hasher := sha256.New()
	hasher.Write(payload)
	expectedDigest := hex.EncodeToString(hasher.Sum(nil))

	target := domain.BlockDevice{
		Path:        "/dev/simdisk0",
		SizeBytes:   1024 * 1024 * 1024, // 1 GiB
		IsRemovable: true,
	}
	src := &domain.ImageMetadata{
		Path:           srcFile,
		SizeBytes:      int64(len(payload)),
		Classification: domain.ClassRawBootable,
		SHA256Checksum: expectedDigest,
	}

	plan, err := CreatePlan(PlanOptions{
		Target:           target,
		Source:           src,
		VerifyAfterWrite: true,
	})
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	fakeDisk := fake.NewSimulatedDisk(1024 * 1024 * 1024)
	fakeProv := fake.NewFakeProvider()
	fakeProv.AddDevice(target, fakeDisk)

	executor := &Executor{
		Provider:     fakeProv,
		TargetWriter: fakeDisk,
		TargetReader: fakeDisk,
	}

	var reportedBytes int64
	journal, err := executor.Execute(context.Background(), plan, func(event ProgressEvent) {
		reportedBytes = event.BytesProcessed
	})
	if err != nil {
		t.Fatalf("Execute on simulated disk failed: %v", err)
	}

	if reportedBytes != int64(len(payload)) {
		t.Errorf("reported bytes mismatch: got %d, want %d", reportedBytes, len(payload))
	}
	if fakeDisk.BytesWritten() != int64(len(payload)) {
		t.Errorf("simulated disk bytes written mismatch: got %d, want %d", fakeDisk.BytesWritten(), len(payload))
	}

	// Verify journal has recorded all steps as Completed
	for _, entry := range journal.Entries {
		if entry.Status != domain.StatusCompleted {
			t.Errorf("step %s failed: %s", entry.Name, entry.Message)
		}
	}
}

func TestExecutorBlocksSystemDisk(t *testing.T) {
	systemTarget := domain.BlockDevice{
		Path:        "/dev/nvme0n1",
		SizeBytes:   512 * 1024 * 1024 * 1024,
		Mountpoints: []string{"/"}, // Root filesystem!
		IsSystem:    true,
	}
	src := &domain.ImageMetadata{
		Path:      "/tmp/fake.iso",
		SizeBytes: 1024,
	}

	plan, _ := CreatePlan(PlanOptions{
		Target: systemTarget,
		Source: src,
	})

	executor := NewExecutor()
	_, err := executor.Execute(context.Background(), plan, nil)
	if err == nil {
		t.Fatalf("expected error executing against root system disk, got nil")
	}
}
