package iso

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xzahed/ventox/internal/domain"
)

func TestComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.iso")
	data := []byte("Ventox native boot engine sha256 verification test data")
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	expectedHash := sha256.Sum256(data)
	expectedHex := hex.EncodeToString(expectedHash[:])

	var lastBytesRead int64
	hash, err := ComputeSHA256(context.Background(), testFile, func(bytesRead, totalBytes int64) {
		lastBytesRead = bytesRead
	})
	if err != nil {
		t.Fatalf("ComputeSHA256 failed: %v", err)
	}

	if hash != expectedHex {
		t.Errorf("hash mismatch: got %s, want %s", hash, expectedHex)
	}
	if lastBytesRead != int64(len(data)) {
		t.Errorf("progress mismatch: got %d bytes read, want %d", lastBytesRead, len(data))
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	_, err = ComputeSHA256(ctx, testFile, nil)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestInspectSyntheticISO(t *testing.T) {
	tmpDir := t.TempDir()
	isoPath := filepath.Join(tmpDir, "windows_installer.iso")

	// Create a synthetic ISO file:
	// Sector size = 2048.
	// Sector 0..15: reserved (32768 bytes)
	// Sector 16: Primary Volume Descriptor (Type 1)
	// Sector 17: Boot Record (Type 0, El Torito)
	// Sector 18: Terminator (Type 255)
	// Sector 19: El Torito Boot Catalog
	// Sector 20: Root Directory Extent (contains SOURCES directory)
	// Sector 21: SOURCES Directory Extent (contains INSTALL.WIM > 4 GiB)

	totalSectors := 25
	isoData := make([]byte, totalSectors*2048)

	// Sector 16: PVD
	pvd := isoData[16*2048 : 17*2048]
	pvd[0] = 1 // PVD
	copy(pvd[1:6], "CD001")
	pvd[6] = 1 // Version
	copy(pvd[40:72], "VENTOX_WIN11_TEST              ")

	// Root dir record at pvd[156:190] (34 bytes)
	pvd[156] = 34                                  // record len
	binary.LittleEndian.PutUint32(pvd[158:162], 20) // extent LBA = sector 20
	binary.LittleEndian.PutUint32(pvd[166:170], 2048) // data len = 2048

	// Sector 17: Boot Record (El Torito)
	br := isoData[17*2048 : 18*2048]
	br[0] = 0 // Boot Record
	copy(br[1:6], "CD001")
	br[6] = 1
	copy(br[7:39], "EL TORITO SPECIFICATION         ")
	binary.LittleEndian.PutUint32(br[71:75], 19) // Catalog LBA = sector 19

	// Sector 18: Terminator
	term := isoData[18*2048 : 19*2048]
	term[0] = 255
	copy(term[1:6], "CD001")
	term[6] = 1

	// Sector 19: El Torito Boot Catalog
	cat := isoData[19*2048 : 20*2048]
	cat[0] = 0x01 // Header validation
	cat[30] = 0x55
	cat[31] = 0xAA
	// Initial default entry (offset 32..63)
	cat[32] = 0x88 // Bootable
	cat[33] = 0xEF // EFI System Type!

	// Sector 20: Root Directory (contains "SOURCES" entry pointing to sector 21)
	root := isoData[20*2048 : 21*2048]
	// Entry 1: "." (offset 0)
	root[0] = 34
	root[25] = 0x02 // dir
	root[32] = 1
	root[33] = 0
	// Entry 2: "SOURCES" (offset 34)
	sEntry := root[34:]
	sEntry[0] = 42 // record len: 33 + 7 ("SOURCES") + 2 padding = 42
	binary.LittleEndian.PutUint32(sEntry[2:6], 21) // LBA = sector 21
	binary.LittleEndian.PutUint32(sEntry[10:14], 2048) // len = 2048
	sEntry[25] = 0x02 // dir
	sEntry[32] = 7    // name len
	copy(sEntry[33:40], "SOURCES")

	// Sector 21: SOURCES Directory (contains "INSTALL.WIM" and "BOOT.WIM")
	sources := isoData[21*2048 : 22*2048]
	// Entry 1: "."
	sources[0] = 34
	sources[25] = 0x02
	sources[32] = 1
	sources[33] = 0
	// Entry 2: "BOOT.WIM" (offset 34)
	bEntry := sources[34:]
	bEntry[0] = 44
	bEntry[32] = 8
	copy(bEntry[33:41], "BOOT.WIM")
	binary.LittleEndian.PutUint32(bEntry[10:14], 500*1024*1024) // 500 MB

	// Entry 3: "INSTALL.WIM" (offset 34 + 44 = 78)
	wEntry := sources[78:]
	wEntry[0] = 46
	wEntry[32] = 11
	copy(wEntry[33:44], "INSTALL.WIM")
	// Set data size to 0xFFFFFFFF (representing > 4 GiB in 32-bit field)
	binary.LittleEndian.PutUint32(wEntry[10:14], 0xFFFFFFFF)

	if err := os.WriteFile(isoPath, isoData, 0644); err != nil {
		t.Fatalf("failed to write synthetic ISO: %v", err)
	}

	meta, err := InspectImage(isoPath)
	if err != nil {
		t.Fatalf("InspectImage failed: %v", err)
	}

	if meta.Format != domain.FormatISO {
		t.Errorf("expected FormatISO, got %v", meta.Format)
	}
	if meta.Classification != domain.ClassWindowsInstaller {
		t.Errorf("expected ClassWindowsInstaller, got %v", meta.Classification)
	}
	if !meta.IsEFI {
		t.Errorf("expected IsEFI to be true from El Torito catalog")
	}
	if !meta.HasLargeWIM {
		t.Errorf("expected HasLargeWIM to be true for 0xFFFFFFFF size")
	}
	if meta.VolumeLabel != "VENTOX_WIN11_TEST" {
		t.Errorf("expected volume label 'VENTOX_WIN11_TEST', got '%s'", meta.VolumeLabel)
	}
	if meta.WIMPath != "sources/install.wim" {
		t.Errorf("expected WIMPath 'sources/install.wim', got '%s'", meta.WIMPath)
	}
}

func TestInspectRawImage(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "archlinux.raw")

	// Create a raw disk image with MBR bootable partition and EFI partition
	data := make([]byte, 1024*1024) // 1 MiB
	data[510] = 0x55
	data[511] = 0xAA

	// Partition 1 at 446
	data[446] = 0x80 // Bootable
	data[446+4] = 0x83 // Linux partition

	// Partition 2 at 446 + 16 = 462
	data[462] = 0x00
	data[462+4] = 0xEF // EFI partition

	if err := os.WriteFile(imgPath, data, 0644); err != nil {
		t.Fatalf("failed to write raw image: %v", err)
	}

	meta, err := InspectImage(imgPath)
	if err != nil {
		t.Fatalf("InspectImage failed: %v", err)
	}

	if meta.Format != domain.FormatRawIMG {
		t.Errorf("expected FormatRawIMG, got %v", meta.Format)
	}
	if meta.Classification != domain.ClassRawBootable {
		t.Errorf("expected ClassRawBootable, got %v", meta.Classification)
	}
	if !meta.IsEFI {
		t.Errorf("expected IsEFI to be true from 0xEF partition type")
	}
}

func TestDiscoverImages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	isoFile := filepath.Join(tmpDir, "ubuntu.iso")
	imgFile := filepath.Join(tmpDir, "raspi.img")
	txtFile := filepath.Join(tmpDir, "notes.txt")

	_ = os.WriteFile(isoFile, []byte("fake iso"), 0644)
	_ = os.WriteFile(imgFile, []byte("fake img"), 0644)
	_ = os.WriteFile(txtFile, []byte("not an image"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	results, err := DiscoverImages(ctx, tmpDir)
	if err != nil {
		t.Fatalf("DiscoverImages failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 discovered images, got %d", len(results))
	}

	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
	}

	if !names["ubuntu.iso"] || !names["raspi.img"] {
		t.Errorf("expected ubuntu.iso and raspi.img, got %+v", names)
	}
}

func TestRealWindows11ISOIfExists(t *testing.T) {
	realISO := "/home/panda/Downloads/Windows11_Client_x64_en-gb_26300_9457.iso"
	if _, err := os.Stat(realISO); err != nil {
		t.Skip("real Windows 11 ISO not present, skipping")
	}

	meta, err := InspectImage(realISO)
	if err != nil {
		t.Fatalf("InspectImage failed on real Windows 11 ISO: %v", err)
	}

	t.Logf("Real Windows 11 ISO metadata: %+v", meta)
	if meta.Classification != domain.ClassWindowsInstaller {
		t.Errorf("expected ClassWindowsInstaller, got %s", meta.Classification)
	}
	if !meta.HasLargeWIM {
		t.Errorf("expected HasLargeWIM to be true for 8.44 GB Windows 11 ISO")
	}
	if !meta.IsEFI {
		t.Errorf("expected IsEFI to be true")
	}
}


