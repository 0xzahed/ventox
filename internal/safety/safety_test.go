package safety

import (
	"context"
	"errors"
	"testing"

	"github.com/0xzahed/ventox/internal/domain"
)

func TestSystemDiskProtection(t *testing.T) {
	// Simulated system disk with root mount
	rootDisk := domain.BlockDevice{
		Path:        "/dev/nvme0n1",
		Name:        "nvme0n1",
		SizeBytes:   512 * 1024 * 1024 * 1024,
		Mountpoints: []string{"/"},
	}

	isSys, reason, err := IsSystemDisk(rootDisk)
	if err != nil {
		t.Fatalf("IsSystemDisk error: %v", err)
	}
	if !isSys {
		t.Errorf("Expected root disk to be identified as system disk, got false")
	}
	t.Logf("Root disk correctly blocked: %s", reason)

	// Simulated child partition with boot mount
	childSysDisk := domain.BlockDevice{
		Path:      "/dev/sda",
		Name:      "sda",
		SizeBytes: 256 * 1024 * 1024 * 1024,
		Partitions: []domain.PartitionInfo{
			{
				Path:        "/dev/sda1",
				Mountpoints: []string{"/boot/efi"},
			},
		},
	}
	isSys, reason, _ = IsSystemDisk(childSysDisk)
	if !isSys {
		t.Errorf("Expected disk with /boot/efi child to be identified as system disk")
	}
	t.Logf("Child system partition correctly blocked: %s", reason)

	// Normal USB drive
	usbDisk := domain.BlockDevice{
		Path:        "/dev/sdb",
		Name:        "sdb",
		SizeBytes:   32 * 1024 * 1024 * 1024,
		Mountpoints: []string{"/run/media/panda/USB"},
	}
	isSys, _, _ = IsSystemDisk(usbDisk)
	if isSys {
		t.Errorf("Expected standard USB drive NOT to be flagged as system disk")
	}
}

func TestCapacityCheck(t *testing.T) {
	target := domain.BlockDevice{
		Path:      "/dev/sdb",
		SizeBytes: 8 * 1024 * 1024 * 1024, // 8 GiB
	}

	tooLargeISO := &domain.ImageMetadata{
		Path:      "/path/to/large.iso",
		SizeBytes: 10 * 1024 * 1024 * 1024, // 10 GiB
	}

	err := CheckCapacity(target, tooLargeISO, domain.ModeRawWrite)
	if err == nil {
		t.Errorf("Expected capacity error when ISO exceeds target capacity")
	}

	fittingISO := &domain.ImageMetadata{
		Path:      "/path/to/fitting.iso",
		SizeBytes: 4 * 1024 * 1024 * 1024, // 4 GiB
	}
	if err := CheckCapacity(target, fittingISO, domain.ModeRawWrite); err != nil {
		t.Errorf("Expected fitting ISO to pass capacity check, got: %v", err)
	}
}

func TestIdentityRevalidation(t *testing.T) {
	baseline := domain.BlockDevice{
		Path:      "/dev/sdb",
		SizeBytes: 16000000000,
		Vendor:    "SanDisk",
		Model:     "Cruzer Blade",
	}

	identical := baseline
	if err := RevalidateDevice(identical, baseline); err != nil {
		t.Errorf("Expected identical device to revalidate, got: %v", err)
	}

	changedSize := baseline
	changedSize.SizeBytes = 32000000000
	if err := RevalidateDevice(changedSize, baseline); err == nil {
		t.Errorf("Expected error when device size changed")
	}

	changedModel := baseline
	changedModel.Model = "Kingston DataTraveler"
	if err := RevalidateDevice(changedModel, baseline); err == nil {
		t.Errorf("Expected error when device model changed")
	}
}

func TestValidatePlan(t *testing.T) {
	ctx := context.Background()
	plan := &domain.WritePlan{
		Target: domain.BlockDevice{
			Path:        "/dev/nvme0n1",
			Mountpoints: []string{"/"},
		},
		Mode: domain.ModeFormatOnly,
	}

	err := ValidatePlan(ctx, plan)
	if err == nil {
		t.Fatalf("Expected plan validation to fail for system disk target")
	}
}

func TestChainStopsAtFirstFailure(t *testing.T) {
	boom := errors.New("boom")
	ran := false
	chain := Chain{
		ValidatorFunc(func(context.Context, *domain.WritePlan) error { return boom }),
		ValidatorFunc(func(context.Context, *domain.WritePlan) error { ran = true; return nil }),
	}
	if err := chain.Validate(context.Background(), &domain.WritePlan{}); !errors.Is(err, boom) {
		t.Errorf("want boom, got %v", err)
	}
	if ran {
		t.Errorf("chain continued after failure")
	}
}

func TestValidatePlanNil(t *testing.T) {
	if err := ValidatePlan(context.Background(), nil); !errors.Is(err, domain.ErrValidationFailed) {
		t.Errorf("want ErrValidationFailed, got %v", err)
	}
}
