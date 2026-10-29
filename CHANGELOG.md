# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.3.0] - 2026-10-30

### Added
- **Native Go Architecture:** Completely decoupled from external scripting dependencies with layered architecture (`domain`, `device`, `iso`, `safety`, `engine`, `writer`, `ui`).
- **Safety Preflight Engine (`internal/safety`):**
  - Host root, boot, `/home`, swap space, and macOS startup disk protection (`IsSystemDisk`).
  - Target drive capacity preflight check (`ValidateCapacity`).
  - Device identity revalidation snapshot preventing device-swap race conditions (`RevalidateDevice`).
- **ISO9660 & UDF Header Inspector (`internal/iso`):**
  - Sector 16 Primary Volume Descriptor parser for labels and root extents.
  - El Torito boot catalog parser identifying EFI system type `0xEF` and Microsoft Corporation vendor ID.
  - Detection of oversized `install.wim` files exceeding the 4 GiB FAT32 file limit.
  - Streaming SHA-256 checksum calculation with context cancellation.
- **Cross-Platform Device Management Engine (`internal/device`):**
  - Linux block device query via `lsblk` with `/sys/block` fallback.
  - macOS physical disk query via `diskutil -plist` with `/dev/rdisk` character device optimization.
  - Windows physical drive query via PowerShell `Get-Disk` JSON / WMI.
  - In-memory sparse block simulated disk backend (`internal/device/fake`) for unit testing without physical USB hardware.
- **Media Creation Engine (`internal/engine` & `internal/writer`):**
  - `ModeDualPartitionUEFI`: Rufus-style dual-partition scheme (FAT32 BOOT + NTFS INSTALL) for Windows 10/11.
  - `ModeRawWrite`: 4 MiB chunk aligned byte-for-byte disk image streaming with throughput and ETA calculation.
  - `ModeFormatOnly`: Multi-filesystem formatting (exFAT, FAT32, NTFS, ext4, APFS).
  - Post-write readback SHA-256 verification engine.
  - Windows 11 `autounattend.xml` answer file generator for LabConfig bypasses (TPM 2.0, Secure Boot, RAM, CPU, Storage, NRO).
- **CLI Commands & Flags:**
  - `--dry-run`: Non-destructive simulation executing preflight validation without writing.
  - `--verify`: Post-write SHA-256 readback data verification.
  - `--bypass-win11`: Configurable Windows 11 hardware requirement bypass.
  - `--device <path>`: Direct target selection for scripting.
  - `--iso <path>`: Direct image selection.
  - `--fs <type>`: Selectable filesystem.
  - `--json`: Structured JSON output for CI pipelines and scripts.
  - `-y` / `--yes`: Automated execution without interactive prompt.
- **Documentation & CI:**
  - Architecture Decision Record (`docs/adr/0001-native-boot-engine.md`).
  - Platform Compatibility Matrix (`docs/compatibility.md`).
  - Security and Destructive-Operation Policy (`docs/security.md`).
  - GitHub Actions multi-OS CI workflow across Linux, macOS, and Windows.

---

## [0.2.0] - 2026-10-30
### Added
- Rufus-style dual-partition prototype for Windows installation media.
- macOS diskutil external drive detection.
- Multi-filesystem format commands (`ventox format`).
- Auto-elevation support with sudo.

---

## [0.1.0] - 2026-10-29
### Added
- Initial project release with USB and ISO detection.
