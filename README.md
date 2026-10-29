<div align="center">

# ⚡ Ventox

### Next-Generation UEFI/GPT Bootable USB Creator & Drive Formatter in Go
#### The Rufus Alternative for Linux & macOS (Apple Silicon M1/M2/M3/M4 & Intel)

[![Go Report Card](https://goreportcard.com/badge/github.com/0xzahed/ventox)](https://goreportcard.com/report/github.com/0xzahed/ventox)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/0xzahed/ventox)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Release](https://img.shields.io/github/v/release/0xzahed/ventox?include_prereleases)](https://github.com/0xzahed/ventox/releases)

<p align="center">
  <b>Ventox</b> is a fast, interactive, zero-configuration CLI tool written in Go that brings the proven architecture of <b>Rufus</b> to <b>Linux & macOS</b>. It automatically detects USB flash drives and ISO images, crafts 100% compliant UEFI/GPT bootable media for both <b>Windows 11</b> and <b>Linux</b> without the 4 GB FAT32 file size limit, and provides a standalone <b>Multi-Filesystem USB Formatter</b> (NTFS, exFAT, FAT32, ext4, APFS).
</p>

[Quick Start](#-quick-start--commands) •
[Rufus Architecture](#-the-ventox-native-engine-rufus-inspired) •
[Features](#-features) •
[Comparison](#-feature-comparison) •
[Installation](#-installation) •
[Contributing](#-contributing) •
[License](#-license)

</div>

---

## 🎯 The Problem

Creating a bootable Windows 11 or modern Linux USB drive on non-Windows platforms has always been frustrating:
1. **Raw `dd` fails on Windows ISOs:** Direct byte copy (`dd if=win11.iso of=/dev/sdX`) does **not** boot Windows installers because Windows ISOs lack hybrid MBR/GPT partition headers.
2. **FAT32 4 GB Limit:** Standard UEFI boot partitions require FAT32, but Windows 11 `install.wim` is typically **5 GB to 8+ GB**, causing copying to fail unless manually split with specialized tools.
3. **Hardware Restrictions in Windows 11:** Official Windows 11 installers block installation on computers lacking TPM 2.0, Secure Boot, or 4GB RAM, and force an active Microsoft Online Account during setup.
4. **Heavy GUI Tools:** Electron-based flashers consume hundreds of megabytes of RAM and lack native Windows UEFI partition bootstrapping on Linux and macOS.

---

## ⚡ The Ventox Native Engine (Rufus-Inspired)

Ventox features an independent, native boot engine inspired by Pete Batard's architectural work on **Rufus**:

### 1. Dual-Partition UEFI Standard for Windows
* **Partition 1 (`BOOT`, FAT32, ~1024 MiB):** Holds standard UEFI boot components (`/efi/boot/bootx64.efi`, `/boot`, `/bootmgr`, `/bootmgr.efi`), the WinPE kernel (`sources/boot.wim`), and the automated hardware bypass answer file (`autounattend.xml`). Guaranteed 100% UEFI boot compatibility on every motherboard firmware without requiring custom third-party drivers.
* **Partition 2 (`INSTALL`, NTFS/ExFAT, Remaining Disk):** Holds the full Windows installation media including oversized `sources/install.wim` files (> 8 GB). WinPE natively mounts NTFS/ExFAT, locates `sources\install.wim`, and runs setup without FAT32 restrictions or slow WIM file splitting.

### 2. Windows 11 Automated Hardware Bypass
Ventox automatically generates and injects an `autounattend.xml` answer file into both partitions to bypass Microsoft hardware restrictions during `windowsPE` execution:
* `BypassTPMCheck = 1` (Bypasses TPM 2.0 requirement)
* `BypassSecureBootCheck = 1` (Bypasses Secure Boot requirement)
* `BypassRAMCheck = 1` (Bypasses 4GB minimum RAM requirement)
* `BypassCPUCheck = 1` (Bypasses unsupported CPU model checks)
* `BypassStorageCheck = 1` (Bypasses 64GB minimum disk requirement)
* `BypassNRO = 1` (Disables mandatory Microsoft Online Account login during OOBE, allowing offline local accounts)

### 3. Direct Streaming Block Copy for Linux
For Linux hybrid ISO images (Ubuntu, Debian, Fedora, Arch, Kali), Ventox operates in **Raw Block Streaming (DD Mode)** with a high-throughput 4MB buffer, real-time progress calculations (MB/s and ETA), and kernel cache synchronization (`sync`).

---

## 📊 Feature Comparison

| Feature | Rufus | balenaEtcher | Ventoy | **Ventox** |
| :--- | :---: | :---: | :---: | :---: |
| **Linux Support** | ❌ (Windows only) | ✅ | ✅ | **✅ (Native)** |
| **macOS Support (Apple Silicon & Intel)** | ❌ | ✅ | ❌ | **✅ (Native)** |
| **Windows 11 UEFI without 4GB Limit** | ✅ | ❌ | ✅ | **✅ (Native Dual-Partition)** |
| **Automated Win 11 TPM & Account Bypass** | ✅ | ❌ | Manual Plugin | **✅ (Built-in autounattend.xml)** |
| **Standalone USB Formatter (NTFS/FAT/exFAT)**| ✅ | ❌ | ❌ | **✅ (`ventox format`)** |
| **Resource Usage** | Lightweight (<5MB) | Heavy Electron (~250MB+) | Medium | **Ultra-Lightweight (<10MB)** |
| **External Dependencies** | None (Win32) | Node/Electron | Bash/Tarball | **Zero External Dependencies** |

---

## 🚀 Terminal Preview

```text
╔═══════════════════════════════════════════════════════════════╗
║                      ⚡ V E N T O X ⚡                        ║
║     Next-Gen UEFI/GPT Bootable USB Creator for Linux & Win    ║
╚═══════════════════════════════════════════════════════════════╝

[1/3] Detecting USB Drives
✔ Auto-detected USB Drive:
   Device: /dev/sdb (28.9G) - USB DISK 3.2
Use this USB drive? [Y/n/custom] [Y]: 
✔ Target USB Selected: /dev/sdb

[2/3] Scanning for ISO Files
✔ Auto-detected ISO File:
   Name: Windows11_Client_x64_en-gb_26300_9457.iso
   Size: 8.44 GB
   Path: /home/panda/Downloads/Windows11_Client_x64_en-gb_26300_9457.iso
Use this ISO file? [Y/n/custom] [Y]: 
✔ ISO File Selected: Windows11_Client_x64_en-gb_26300_9457.iso

===============================================================
 ⚠️  CRITICAL WARNING: ALL DATA ON THE USB WILL BE ERASED!   ⚠️ 
===============================================================
 Target Device: /dev/sdb
 Source ISO:    /home/panda/Downloads/Windows11_Client_x64_en-gb_26300_9457.iso
===============================================================
Type YES to proceed and write bootable image [NO]: YES

[3/3] Deploying Bootable Media (Native Rufus-Style Engine)
ℹ Analyzing ISO image architecture (native inspector)...
✔ Detected: Windows OS Installer (Install WIM: sources/install.wim)
ℹ Applying Rufus-Style Dual-Partition Architecture (FAT32 BOOT + NTFS INSTALL)...
ℹ Injecting Windows 11 Hardware Bypass (TPM/SecureBoot/OOBE BypassNRO)...
Copying: [████████████████████░░░░░]  78.4% (6.62 GB / 8.44 GB) | 31.2 MB/s | ETA: 00m 58s
ℹ Flushing filesystem cache (sync) to ensure USB integrity...
✔ All data flushed to USB successfully.

═══════════════════════════════════════════════════════════════
  🎉 CONGRATULATIONS! BOOTABLE USB IS READY! 🎉               
═══════════════════════════════════════════════════════════════
 • Target USB:   /dev/sdb
 • Boot Engine:  Ventox Native (Rufus-Style Dual Partition UEFI / DD)
 • Loaded ISO:   Windows11_Client_x64_en-gb_26300_9457.iso
 • Ready to boot on any modern PC (Press F12, F11, or Esc at boot)
═══════════════════════════════════════════════════════════════
```

---

## ✨ Features

- 🔍 **Hardware Auto-Detection:** Automatically discovers removable USB drives using `lsblk` (Linux) or `diskutil` (macOS), ignoring internal system disks.
- 📁 **Automated ISO Discovery:** Crawls user directories (`Downloads`, `Desktop`, `Documents`) across user accounts even when running under `sudo`.
- 🛡️ **Defensive Unmounting:** Automatically detects and force/lazy unmounts busy partitions locked by desktop file managers (Nautilus, Dolphin, Finder).
- 🪟 **True Windows 11 Compatibility:** Dual-partition GPT scheme with automated `autounattend.xml` hardware restriction bypass.
- 🍏 **Full macOS Support:** Native builds for both Apple Silicon (M1/M2/M3/M4) and Intel Macs with unbuffered `/dev/rdisk` block acceleration.
- 📊 **Real-time Transfer Analytics:** Live speed (MB/s), elapsed time, ETA countdown, and byte-accurate progress bar.
- 🧹 **Standalone USB Formatter:** Reset any bootable, locked, or corrupted flash drive back to a clean universal pen drive with your choice of filesystem.
- 📦 **Zero External Go Dependencies:** Pure standard library Go implementation for blazing-fast builds and small binary footprint (<10 MB).

---

## 🎮 Quick Start & Commands

### CLI Command Overview

| Command | Description | Needs Root? |
| :--- | :--- | :---: |
| `ventox` | Interactive bootable USB creator wizard | Yes (Auto-elevates) |
| `ventox format` (or `wipe`) | Format or restore a USB drive (NTFS, exFAT, FAT32, ext4, APFS) | Yes (Auto-elevates) |
| `ventox list` (or `ls`) | List connected USB storage drives and scanned ISO files | ❌ No |
| `ventox version` (or `-v`) | Display version, build, runtime, and license information | ❌ No |
| `ventox help` (or `-h`) | Display comprehensive help screen and usage examples | ❌ No |

---

### 1. Creating a Bootable USB (Default Wizard)

```bash
ventox
```
1. Insert your USB drive.
2. Select your USB drive (or press **Enter** to accept the auto-detected one).
3. Select your `.iso` image (or press **Enter** to accept the auto-detected one).
4. Review the drive path and confirm by typing **`YES`**.
5. Wait for the copy and cache flush to complete!

---

### 2. Formatting or Restoring a USB Drive (No ISO Required)

When you want to wipe a bootable USB, fix a corrupted drive, or restore it back to standard universal storage:

```bash
ventox format
```

Ventox provides 6 distinct format profiles:
* **[1] Ventoy Bootable (GPT / UEFI):** Modern UEFI systems & Windows 11.
* **[2] Ventoy Bootable (MBR / Legacy BIOS):** For older motherboards and BIOS compatibility.
* **[3] Clean Standard USB (exFAT):** Universal cross-platform storage (Windows/Mac/Linux) with files > 4GB.
* **[4] Clean Standard USB (FAT32):** Maximum compatibility with embedded systems, stereos, and older consoles.
* **[5] Clean Standard USB (NTFS):** Native Windows filesystem with permissions & journaling.
* **[6] Clean Standard USB (ext4 / APFS):** High-performance Unix/Linux/macOS native filesystem.

---

### 3. Inspecting Drives and ISOs

```bash
ventox list
```
Scans and prints all connected removable USB drives and discoverable ISO disk images without touching or modifying any disks.

---

## 📦 Installation

### Option 1: Pre-compiled Binaries (Recommended)

Download the release archive matching your operating system and CPU architecture from the [Releases](https://github.com/0xzahed/ventox/releases) page:

```bash
# macOS Apple Silicon (M1/M2/M3/M4)
curl -sSL https://github.com/0xzahed/ventox/releases/latest/download/ventox-darwin-arm64.tar.gz | tar -xz
sudo mv ventox-darwin-arm64 /usr/local/bin/ventox

# macOS Intel
curl -sSL https://github.com/0xzahed/ventox/releases/latest/download/ventox-darwin-amd64.tar.gz | tar -xz
sudo mv ventox-darwin-amd64 /usr/local/bin/ventox

# Linux x86_64
curl -sSL https://github.com/0xzahed/ventox/releases/latest/download/ventox-linux-amd64.tar.gz | tar -xz
sudo mv ventox-linux-amd64 /usr/local/bin/ventox
```

### Option 2: Via Go Install

```bash
go install github.com/0xzahed/ventox/cmd/ventox@latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/0xzahed/ventox.git
cd ventox
make build
sudo make install
```

---

## 🛠️ Project Structure

```
ventox/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                 # Automated CI tests for Go 1.22 & 1.23
│   │   └── release.yml            # Multiplatform build & release (Linux & macOS)
│   ├── ISSUE_TEMPLATE/            # Standardized bug & feature request templates
│   └── PULL_REQUEST_TEMPLATE.md   # PR review checklist
├── bin/                           # Local build outputs
├── cmd/
│   └── ventox/
│       └── main.go                # Application entrypoint & CLI command router
├── internal/
│   ├── copier/
│   │   └── progress.go            # Buffered file streamer with ETA & rate calculations
│   ├── detector/
│   │   ├── disk.go                # Shared USB disk definitions
│   │   ├── disk_linux.go          # Linux lsblk JSON block device detector
│   │   ├── disk_darwin.go         # macOS diskutil external USB detector
│   │   ├── iso.go                 # Recursive file crawler for ISO disk images
│   │   └── detector_test.go       # Detector unit tests
│   ├── engine/
│   │   ├── format.go              # Shared format profiles & modes
│   │   ├── format_linux.go        # Linux filesystem formatter (mkfs.ntfs, vfat, exfat, ext4)
│   │   ├── format_darwin.go       # macOS filesystem formatter (diskutil eraseDisk)
│   │   ├── unmount_linux.go       # Linux lazy unmount (/proc/mounts)
│   │   ├── unmount_darwin.go      # macOS force unmount (diskutil unmountDisk)
│   │   └── native/
│   │       ├── analyzer.go        # Read-only ISO inspector (Windows vs Linux)
│   │       ├── bypass.go          # Windows 11 autounattend.xml bypass generator
│   │       ├── windows.go         # Dual-partition UEFI/GPT Windows deployer
│   │       ├── linux.go           # High-speed raw block streamer (DD Mode)
│   │       ├── partition_linux.go # Linux parted & mkfs partition setup
│   │       ├── partition_darwin.go# macOS diskutil partition setup
│   │       ├── mount_linux.go     # Linux loopback ISO mount
│   │       ├── mount_darwin.go    # macOS hdiutil ISO mount
│   │       ├── raw_linux.go       # Linux block device path provider
│   │       ├── raw_darwin.go      # macOS /dev/rdisk unbuffered path provider
│   │       └── engine.go          # Native boot engine coordinator
│   └── ui/
│       ├── colors.go              # ANSI terminal styling and ASCII banners
│       ├── help.go                # Help screens and version outputs
│       └── menu.go                # Interactive terminal menus and confirmation gates
├── CONTRIBUTING.md                # Guidelines for open-source contributors
├── CODE_OF_CONDUCT.md             # Contributor Covenant v2.1 code of conduct
├── LICENSE                        # MIT Open Source License
├── Makefile                       # Multiplatform build targets (Linux & macOS)
└── README.md                      # Comprehensive project documentation
```

---

## 🤝 Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**!

Please read our [Contributing Guide](CONTRIBUTING.md) to learn how to:
- Set up your local development environment
- Run unit tests and lint checks (`make test`, `make lint`)
- Submit bug reports and feature requests
- Open pull requests following Conventional Commits

---

## 📄 License

Distributed under the **MIT License**. See [`LICENSE`](LICENSE) for more information.

---

<div align="center">
  <sub>Crafted with ❤️ by <a href="https://github.com/0xzahed">Abu Zahed</a> and open-source contributors.</sub>
</div>
