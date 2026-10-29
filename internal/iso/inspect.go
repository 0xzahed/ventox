package iso

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

const (
	isoSectorSize     = 2048
	isoPVDOffset      = 16 * isoSectorSize // 32768 (0x8000)
	mbrPartitionStart = 446
	maxDescriptors    = 32
	fat32MaxFileSize  = 4 * 1024 * 1024 * 1024 // 4 GiB
)

// InspectImage analyzes a disk image file and extracts rich metadata,
// detecting bootability, EFI catalogs, Windows WIM sizes, and hybrid Linux structures.
func InspectImage(filePath string) (domain.ImageMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return domain.ImageMetadata{}, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return domain.ImageMetadata{}, fmt.Errorf("failed to stat image: %w", err)
	}

	meta := domain.ImageMetadata{
		Path:           filePath,
		Name:           filepath.Base(filePath),
		SizeBytes:      stat.Size(),
		ModTime:        stat.ModTime(),
		Format:         domain.FormatUnknown,
		Classification: domain.ClassUnknown,
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".img" || ext == ".raw" {
		meta.Format = domain.FormatRawIMG
	} else if ext == ".iso" {
		meta.Format = domain.FormatISO
	}

	// 1. Inspect Sector 0 (MBR) and Sector 1 (GPT)
	isMBRBootable, isGPT, hasEFIPartition := inspectMBRAndGPT(file)
	if hasEFIPartition {
		meta.IsEFI = true
	}

	// 2. Inspect ISO9660 / UDF Volume Descriptors
	isISO9660, isUDF, pvdLabel, elToritoEFI, isMicrosoft, rootLBA, rootLen := inspectVolumeDescriptors(file)
	if isISO9660 || isUDF {
		meta.Format = domain.FormatISO
	}
	if pvdLabel != "" {
		meta.VolumeLabel = pvdLabel
	}
	if elToritoEFI {
		meta.IsEFI = true
	}

	// 3. Inspect ISO filesystem tree (if ISO9660 root directory available)
	if rootLBA > 0 && rootLen > 0 {
		inspectISOFileSystem(file, rootLBA, rootLen, &meta)
	}

	// 4. Fallback inspection for UDF-only or modern Windows ISOs
	if meta.Classification == domain.ClassUnknown {
		inspectWindowsFallback(file, stat.Size(), isMicrosoft, pvdLabel, &meta)
	}

	// 5. If still unknown, check if it's a Linux hybrid or raw bootable image
	if meta.Classification == domain.ClassUnknown {
		if isMBRBootable || isGPT {
			if meta.Format == domain.FormatISO {
				meta.Classification = domain.ClassLinuxHybrid
			} else {
				meta.Classification = domain.ClassRawBootable
			}
		}
	}

	return meta, nil
}

// inspectMBRAndGPT checks sector 0 for MBR boot code and sector 1 for GPT header.
func inspectMBRAndGPT(r io.ReaderAt) (bootableMBR bool, hasGPT bool, hasEFIPart bool) {
	sector0 := make([]byte, 512)
	if _, err := r.ReadAt(sector0, 0); err != nil {
		return false, false, false
	}

	// Check MBR signature 0x55AA
	if sector0[510] != 0x55 || sector0[511] != 0xAA {
		return false, false, false
	}

	// Check partition entries (4 entries of 16 bytes starting at 446)
	for i := 0; i < 4; i++ {
		offset := mbrPartitionStart + (i * 16)
		status := sector0[offset]
		partType := sector0[offset+4]

		if status == 0x80 {
			bootableMBR = true
		}
		if partType == 0xEF { // EFI System Partition
			hasEFIPart = true
		}
		if partType == 0xEE { // Protective MBR for GPT
			hasGPT = true
		}
	}

	// Check GPT Header at sector 1 (offset 512)
	sector1 := make([]byte, 512)
	if _, err := r.ReadAt(sector1, 512); err == nil {
		if string(sector1[:8]) == "EFI PART" {
			hasGPT = true
			hasEFIPart = true
		}
	}

	return bootableMBR, hasGPT, hasEFIPart
}

// inspectVolumeDescriptors iterates ISO9660 descriptors starting at sector 16.
func inspectVolumeDescriptors(r io.ReaderAt) (isISO bool, isUDF bool, label string, elToritoEFI bool, isMicrosoft bool, rootLBA uint32, rootLen uint32) {
	buf := make([]byte, isoSectorSize)

	for sector := 16; sector < 16+maxDescriptors; sector++ {
		offset := int64(sector) * isoSectorSize
		if _, err := r.ReadAt(buf, offset); err != nil {
			break
		}

		magic := string(buf[1:6])
		vdType := buf[0]

		if magic == "CD001" {
			isISO = true
			if vdType == 1 { // Primary Volume Descriptor
				// Volume Identifier is at offset 40 (32 bytes)
				label = strings.TrimSpace(strings.Trim(string(buf[40:72]), "\x00 "))
				// Root Directory Record is at offset 156 (34 bytes)
				rootLBA = binary.LittleEndian.Uint32(buf[158:162])
				rootLen = binary.LittleEndian.Uint32(buf[166:170])
			} else if vdType == 0 { // Boot Record (El Torito)
				bootSysID := strings.TrimSpace(string(buf[7:39]))
				if strings.Contains(strings.ToUpper(bootSysID), "EL TORITO") {
					catalogLBA := binary.LittleEndian.Uint32(buf[71:75])
					efi, ms := checkElToritoCatalog(r, catalogLBA)
					if efi {
						elToritoEFI = true
					}
					if ms {
						isMicrosoft = true
					}
				}
			} else if vdType == 255 { // Volume Descriptor Set Terminator
				break
			}
		} else if magic == "BEA01" || magic == "NSR02" || magic == "NSR03" {
			isUDF = true
		}
	}

	return isISO, isUDF, label, elToritoEFI, isMicrosoft, rootLBA, rootLen
}

// checkElToritoCatalog reads the El Torito boot catalog to check for EFI boot entries and vendor ID.
func checkElToritoCatalog(r io.ReaderAt, catalogLBA uint32) (isEFI bool, isMicrosoft bool) {
	if catalogLBA == 0 {
		return false, false
	}
	buf := make([]byte, isoSectorSize)
	if _, err := r.ReadAt(buf, int64(catalogLBA)*isoSectorSize); err != nil {
		return false, false
	}

	// Header entry (32 bytes): buf[0] == 0x01, buf[30] == 0x55, buf[31] == 0xAA
	if buf[0] != 0x01 || buf[30] != 0x55 || buf[31] != 0xAA {
		return false, false
	}

	headerID := strings.TrimSpace(string(buf[4:28]))
	if strings.Contains(headerID, "Microsoft Corporation") {
		isMicrosoft = true
	}

	// Initial default entry (offset 32..63): System Type at offset 33: 0xEF = EFI
	if buf[33] == 0xEF {
		isEFI = true
	}

	// Check subsequent section entries (up to 16 entries)
	for i := 1; i < 16; i++ {
		entryOffset := i * 32
		if entryOffset+32 > len(buf) {
			break
		}
		indicator := buf[entryOffset]
		sysType := buf[entryOffset+1]

		if sysType == 0xEF && (indicator == 0x90 || indicator == 0x91 || indicator == 0x88) {
			isEFI = true
		}
	}

	return isEFI, isMicrosoft
}

type isoDirEntry struct {
	Name     string
	IsDir    bool
	LBA      uint32
	DataSize uint32
}

// readISODirectory parses directory records from an ISO9660 extent.
func readISODirectory(r io.ReaderAt, lba uint32, length uint32) ([]isoDirEntry, error) {
	if length == 0 || length > 10*1024*1024 { // Cap at 10 MiB to prevent memory exhaustion
		return nil, fmt.Errorf("invalid directory length: %d", length)
	}

	buf := make([]byte, length)
	if _, err := r.ReadAt(buf, int64(lba)*isoSectorSize); err != nil && err != io.EOF {
		return nil, err
	}

	var entries []isoDirEntry
	offset := 0

	for offset < len(buf) {
		recLen := int(buf[offset])
		if recLen == 0 {
			// Zero padding to next 2048-byte sector boundary
			nextSector := ((offset / isoSectorSize) + 1) * isoSectorSize
			if nextSector <= offset || nextSector >= len(buf) {
				break
			}
			offset = nextSector
			continue
		}

		if offset+recLen > len(buf) || recLen < 33 {
			break
		}

		rec := buf[offset : offset+recLen]
		extLBA := binary.LittleEndian.Uint32(rec[2:6])
		dataLen := binary.LittleEndian.Uint32(rec[10:14])
		flags := rec[25]
		isDir := (flags & 0x02) != 0
		nameLen := int(rec[32])

		if 33+nameLen <= recLen {
			nameBytes := rec[33 : 33+nameLen]
			var name string
			if nameLen == 1 && nameBytes[0] == 0 {
				name = "."
			} else if nameLen == 1 && nameBytes[0] == 1 {
				name = ".."
			} else {
				name = string(nameBytes)
				// Strip version suffix ";1" if present
				if idx := strings.Index(name, ";"); idx != -1 {
					name = name[:idx]
				}
			}

			entries = append(entries, isoDirEntry{
				Name:     strings.ToUpper(name),
				IsDir:    isDir,
				LBA:      extLBA,
				DataSize: dataLen,
			})
		}

		offset += recLen
	}

	return entries, nil
}

// inspectISOFileSystem traverses the root ISO9660 directory looking for Windows or Linux markers.
func inspectISOFileSystem(r io.ReaderAt, rootLBA uint32, rootLen uint32, meta *domain.ImageMetadata) {
	rootEntries, err := readISODirectory(r, rootLBA, rootLen)
	if err != nil {
		return
	}

	for _, entry := range rootEntries {
		switch entry.Name {
		case "EFI":
			meta.IsEFI = true
		case "SOURCES":
			// Windows sources directory
			sourcesEntries, err := readISODirectory(r, entry.LBA, entry.DataSize)
			if err == nil {
				for _, se := range sourcesEntries {
					if se.Name == "BOOT.WIM" {
						meta.Classification = domain.ClassWindowsInstaller
					}
					if se.Name == "INSTALL.WIM" || se.Name == "INSTALL.ESD" || se.Name == "INSTALL.SWM" {
						meta.Classification = domain.ClassWindowsInstaller
						meta.WIMPath = "sources/" + strings.ToLower(se.Name)
						meta.WIMSizeBytes = int64(se.DataSize)
						if uint64(se.DataSize) >= fat32MaxFileSize || se.DataSize == 0xFFFFFFFF {
							meta.HasLargeWIM = true
						}
					}
				}
			}
		case "ISOLINUX", "CASPER", "LIVE", "BOOT":
			if meta.Classification != domain.ClassWindowsInstaller {
				meta.Classification = domain.ClassLinuxHybrid
			}
		}
	}
}

// inspectWindowsFallback probes the image for Windows identifiers when UDF or non-standard ISO9660 is used.
func inspectWindowsFallback(r io.ReaderAt, fileSize int64, isMicrosoft bool, label string, meta *domain.ImageMetadata) {
	labelUpper := strings.ToUpper(label)
	isWinLabel := strings.Contains(labelUpper, "_X64FRE_") ||
		strings.Contains(labelUpper, "_ARM64FRE_") ||
		strings.Contains(labelUpper, "_X86FRE_") ||
		strings.HasPrefix(labelUpper, "CCCOMA_") ||
		strings.HasPrefix(labelUpper, "CCSA_") ||
		strings.HasPrefix(labelUpper, "CPRA_")

	if isMicrosoft || isWinLabel {
		meta.Classification = domain.ClassWindowsInstaller
		meta.Format = domain.FormatISO
		meta.WIMPath = "sources/install.wim"
		meta.IsEFI = true
		if fileSize > int64(4.5*1024*1024*1024) {
			meta.HasLargeWIM = true
		}
		return
	}

	// Probe the first 4 MiB for common Windows setup strings
	probeSize := int64(4 * 1024 * 1024)
	if probeSize > fileSize {
		probeSize = fileSize
	}

	probeBuf := make([]byte, probeSize)
	n, err := r.ReadAt(probeBuf, 0)
	if err != nil && err != io.EOF && n == 0 {
		return
	}
	probeBuf = probeBuf[:n]

	// Look for signature markers
	hasSources := bytes.Contains(probeBuf, []byte("sources")) || bytes.Contains(probeBuf, []byte("SOURCES"))
	hasBootMgr := bytes.Contains(probeBuf, []byte("bootmgr")) || bytes.Contains(probeBuf, []byte("BOOTMGR"))
	hasWIM := bytes.Contains(probeBuf, []byte("install.wim")) || bytes.Contains(probeBuf, []byte("INSTALL.WIM"))
	hasESD := bytes.Contains(probeBuf, []byte("install.esd")) || bytes.Contains(probeBuf, []byte("INSTALL.ESD"))
	hasEFI := bytes.Contains(probeBuf, []byte("EFI")) || bytes.Contains(probeBuf, []byte("efi"))

	if hasBootMgr && (hasSources || hasWIM || hasESD) {
		meta.Classification = domain.ClassWindowsInstaller
		meta.Format = domain.FormatISO
		if hasEFI {
			meta.IsEFI = true
		}
		if hasWIM {
			meta.WIMPath = "sources/install.wim"
			// If file size > 4.5 GiB and contains install.wim, it's almost certainly > 4 GiB
			if fileSize > int64(4.5*1024*1024*1024) {
				meta.HasLargeWIM = true
			}
		} else if hasESD {
			meta.WIMPath = "sources/install.esd"
		}
	}
}
