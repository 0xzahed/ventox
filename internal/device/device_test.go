package device

import (
	"context"
	"io"
	"testing"

	"github.com/0xzahed/ventox/internal/device/fake"
	"github.com/0xzahed/ventox/internal/domain"
)

func TestFakeProviderOperations(t *testing.T) {
	fp := fake.NewFakeProvider()

	targetDev := domain.BlockDevice{
		Path:            "/dev/fake-usb0",
		Name:            "fake-usb0",
		SizeBytes:       32 * 1024 * 1024 * 1024,
		SectorSizeBytes: 512,
		Vendor:          "SanDisk",
		Model:           "Ultra",
		SerialNumber:    "FAKE-SN-12345",
		BusType:         "USB",
		IsRemovable:     true,
	}
	simDisk := fake.NewSimulatedDisk(32 * 1024 * 1024 * 1024)
	fp.AddDevice(targetDev, simDisk)

	ctx := context.Background()
	devs, err := fp.ListDevices(ctx)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}
	if len(devs) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devs))
	}

	got, err := fp.GetDevice(ctx, "/dev/fake-usb0")
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if got.SerialNumber != "FAKE-SN-12345" {
		t.Errorf("serial mismatch: got %s", got.SerialNumber)
	}

	// Test writing to simulated disk
	payload := []byte("Ventox native block write payload")
	n, err := simDisk.Write(payload)
	if err != nil || n != len(payload) {
		t.Fatalf("Write failed: n=%d err=%v", n, err)
	}

	// Seek and read back
	_, err = simDisk.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	buf := make([]byte, len(payload))
	n, err = simDisk.Read(buf)
	if err != nil || n != len(payload) {
		t.Fatalf("Read failed: n=%d err=%v", n, err)
	}
	if string(buf) != string(payload) {
		t.Errorf("readback mismatch: got %s, want %s", string(buf), string(payload))
	}

	// Test fault injection: sudden disconnect
	simDisk.SimulateDisconnect = true
	_, err = simDisk.Write([]byte("more data"))
	if err != domain.ErrDeviceRemoved {
		t.Errorf("expected ErrDeviceRemoved on disconnect, got: %v", err)
	}

	// Test removal from provider
	fp.RemoveDevice("/dev/fake-usb0")
	_, err = fp.GetDevice(ctx, "/dev/fake-usb0")
	if err == nil {
		t.Errorf("expected error after device removed from provider")
	}
}

func TestSimulatedDiskShortWrite(t *testing.T) {
	simDisk := fake.NewSimulatedDisk(1024 * 1024)
	simDisk.SimulateShortWrite = true

	largeBuffer := make([]byte, 8192)
	n, err := simDisk.Write(largeBuffer)
	if err != io.ErrShortWrite {
		t.Errorf("expected io.ErrShortWrite, got: %v", err)
	}
	if n != 4096 {
		t.Errorf("expected 4096 bytes written on short write, got: %d", n)
	}
}
