# Platform Compatibility Matrix

This document provides a verified, concrete matrix of Ventox capabilities across host operating systems and hardware architectures. Ventox uses a zero-external-dependency, native Go architecture with platform-specific adapters.

---

## 1. Operating System & Architecture Matrix

| Capability / Feature | Linux (`x86_64`, `arm64`) | macOS Darwin (`arm64` Apple Silicon, `x86_64` Intel) | Windows (`x86_64`) |
| :--- | :--- | :--- | :--- |
| **Storage Device Discovery** | `lsblk` JSON with `/sys/block` fallback | `diskutil list -plist` physical disk filter | PowerShell `Get-Disk` JSON / WMI |
| **Optimized Block I/O** | Direct block device `/dev/sdX` streaming | Raw character device `/dev/rdiskN` (10x-20x faster) | Direct volume handle streaming |
| **System Disk Guard** | Root (`/`), `/boot`, `/boot/efi`, `/home`, swap | Root startup disk (`/`), `/System/Volumes/Data` | System Drive (`C:`), Boot/OS flags |
| **ISO9660 / UDF Inspector** | Pure Go header & El Torito parser | Pure Go header & El Torito parser | Pure Go header & El Torito parser |
| **Windows Installer Mode** | Dual-partition UEFI (FAT32 + NTFS) | Dual-partition UEFI (FAT32 + NTFS) | Native Windows PE preparation |
| **Windows 11 Bypasses** | `autounattend.xml` LabConfig injection | `autounattend.xml` LabConfig injection | `autounattend.xml` LabConfig injection |
| **Raw Disk Image Streaming** | 4 MiB chunk aligned streaming | 4 MiB chunk aligned streaming | 4 MiB chunk aligned streaming |
| **Readback SHA-256 Verify** | Full streaming block digest check | Full streaming block digest check | Full streaming block digest check |
| **Dry-Run Simulation** | Complete preflight & journal log | Complete preflight & journal log | Complete preflight & journal log |
| **Automated Sudo Elevation** | Built-in CLI re-execution with sudo | Built-in CLI re-execution with sudo | Windows UAC prompt / Administrator |

---

## 2. Filesystem Creation Support

Ventox strictly validates host formatting tool availability before initiating any operation. It does not promise filesystems that cannot be created on the host platform.

| Filesystem | Linux Support | macOS Support | Windows Support | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **exFAT** | Supported (`mkfs.exfat`) | Supported (`diskutil eraseVolume ExFAT`) | Supported (`format /FS:exFAT`) | Default cross-platform storage FS |
| **FAT32** | Supported (`mkfs.vfat` / `mkfs.fat`) | Supported (`diskutil eraseVolume FAT32`) | Supported (`format /FS:FAT32`) | Used for UEFI boot partition (max 4GB file) |
| **NTFS** | Supported (`mkfs.ntfs` / ntfs-3g) | Read-only without third-party drivers | Supported (`format /FS:NTFS`) | Used for Windows install.wim data partition |
| **ext4** | Supported (`mkfs.ext4`) | Not supported natively | Not supported natively | Native Linux filesystem |
| **APFS** | Not supported on Linux | Supported (`diskutil eraseVolume APFS`) | Not supported natively | Native Apple modern filesystem |

---

## 3. Creation Modes

### Mode A: Raw Image Streaming (`RawImageStream`)
- **Use Case:** Linux hybrid ISOs (Ubuntu, Debian, Fedora, Arch, Alpine, Proxmox) and raw `.img` / `.raw` disk images (Raspberry Pi OS).
- **Execution:** Writes sector-by-sector using 4 MiB aligned streaming buffers, context cancellation on each iteration, short-write detection, and hardware write buffer synchronization (`sync`).
- **Post-Write Verification:** Supported via SHA-256 block readback.

### Mode B: Windows UEFI Dual-Partition Mode (`DualPartitionUEFI`)
- **Use Case:** Official Windows 10 & 11 ISOs containing `install.wim` files exceeding the 4 GiB FAT32 boundary.
- **Execution:** 
  1. Creates a GPT partition table.
  2. Partition 1 (~1 GiB, FAT32): Holds UEFI boot files (`/efi/boot/bootx64.efi`) ensuring universal UEFI firmware readability without custom drivers.
  3. Partition 2 (Remaining capacity, NTFS): Holds the large `sources/install.wim` and installation image assets.
  4. Injects `autounattend.xml` when Windows 11 bypasses are requested.

### Mode C: Standard USB Formatting (`FormatOnly`)
- **Use Case:** Restoring damaged bootable drives, wiping old partition tables, or converting flash drives into high-performance exFAT/FAT32/NTFS/ext4 storage devices.
- **Execution:** Wipes partition signatures, re-creates GPT or MBR partition table, creates selected filesystem with user label, and forces kernel buffer synchronization.
