<div align="center">

# ⚡ Ventox

### Next-Generation UEFI/GPT Bootable USB Creator in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/0xzahed/ventox)](https://goreportcard.com/report/github.com/0xzahed/ventox)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/0xzahed/ventox)](https://golang.org)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![Release](https://img.shields.io/github/v/release/0xzahed/ventox?include_prereleases)](https://github.com/0xzahed/ventox/releases)

<p align="center">
  <b>Ventox</b> is a fast, interactive, zero-configuration CLI tool written in Go that automatically detects your USB flash drives and ISO disk images, then crafts 100% compliant UEFI/GPT bootable media for both <b>Windows 11</b> and <b>Linux</b> without the FAT32 file size limit.
</p>

[Quick Start](#-quick-start) •
[Features](#-features) •
[Installation](#-installation) •
[How It Works](#-how-it-works) •
[Contributing](#-contributing) •
[License](#-license)

</div>

---

## 🎯 The Problem

Creating a bootable Windows 11 or modern Linux USB drive on Linux is notoriously frustrating:
1. **Raw `dd` fails:** Direct byte copy (`dd if=win11.iso of=/dev/sdX`) does **not** boot Windows installers because Windows ISOs lack hybrid MBR/GPT partition headers.
2. **FAT32 4 GB Limit:** Standard UEFI boot partitions require FAT32, but Windows 11 `install.wim` is typically **5 GB to 7 GB+** (and modern ISOs exceed 8.5 GB), causing copying to fail unless manually split with specialized tools.
3. **Heavy GUI Tools:** Electron-based flashers consume hundreds of megabytes of RAM and often lack native Windows UEFI partition bootstrapping on Linux.

## ⚡ The Ventox Solution

**Ventox** solves this elegantly through a lightweight, standalone Go binary:
- **Auto-Detects Everything:** Discovers inserted USB drives and scans common directories (`Downloads`, `Desktop`, `Documents`) for `.iso` files.
- **Dual Partition GPT Scheme:** Sets up a robust exFAT data partition (no 4 GB file size limit) combined with a UEFI boot partition powered by the Ventoy runtime.
- **Native Go Streamer:** Direct buffered file streaming with live transfer rates (MB/s), visual ASCII progress bar, ETA calculation, and filesystem cache sync.
- **Auto-Elevation:** Seamlessly re-executes with `sudo` if run as an unprivileged user.

---

## 🚀 Terminal Preview

```text
╔═══════════════════════════════════════════════════════════════╗
║                      ⚡ V E N T O X ⚡                        ║
║     Next-Gen UEFI/GPT Bootable USB Creator for Linux & Win    ║
╚═══════════════════════════════════════════════════════════════╝

[1/4] Detecting USB Drives
✔ Auto-detected USB Drive:
   Device: /dev/sda
   Size:   15G
   Model:  SanDisk Cruzer Blade
Use this USB drive? [Y/n/custom] [Y]: 

[2/4] Scanning for ISO Files
✔ Auto-detected ISO File:
   Name: Windows11_Client_x64_en-gb_26300_9457.iso
   Size: 8.44 GB
   Path: /home/panda/Downloads/Windows11_Client_x64_en-gb_26300_9457.iso
Use this ISO file? [Y/n/custom] [Y]: 

===============================================================
 ⚠️  CRITICAL WARNING: ALL DATA ON THE USB WILL BE ERASED!   ⚠️ 
===============================================================
 Target Device: /dev/sda
 Source ISO:    /home/panda/Downloads/Windows11_Client_x64_en-gb_26300_9457.iso
===============================================================
Type YES to proceed and write bootable image [NO]: YES

[3/4] Installing UEFI/GPT Bootloader (Ventoy)
✔ Ventoy bootloader installed successfully.

[4/4] Copying ISO Image to Bootable USB
Copying: [████████████████████░░░░░]  78.4% (6.62 GB / 8.44 GB) | 31.2 MB/s | ETA: 00m 58s
ℹ Flushing filesystem cache (sync) to ensure USB integrity...
✔ All data flushed to USB successfully.

═══════════════════════════════════════════════════════════════
  🎉 CONGRATULATIONS! BOOTABLE USB IS READY! 🎉               
═══════════════════════════════════════════════════════════════
 • Target USB:   /dev/sda
 • Boot Engine:  Ventoy (GPT / UEFI & Legacy BIOS)
 • Loaded ISO:   Windows11_Client_x64_en-gb_26300_9457.iso
 • Ready to boot on any modern PC (Press F12, F11, or Esc at boot)
═══════════════════════════════════════════════════════════════
```

---

## ✨ Features

- 🔍 **Smart Hardware Detection:** Filters out system hard drives and only surfaces removable USB storage devices via `lsblk` JSON parsing.
- 📁 **Automated ISO Discovery:** Crawls user directories for `.iso` disk images and displays them sorted by latest modification time.
- 🛡️ **Defensive Unmounting:** Automatically identifies and lazy-unmounts (`umount -l`) busy partitions locked by desktop file managers (e.g. Nautilus, Dolphin).
- 🪟 **True Windows 11 & Linux Compatibility:** Full GPT partition scheme with Secure Boot and UEFI support.
- 📊 **Real-time Transfer Analytics:** Live speed (MB/s), elapsed time, ETA countdown, and byte-accurate progress bar.
- 📦 **Zero External Go Dependencies:** Pure standard library Go implementation for blazing-fast builds and small binary footprint (<10 MB).

---

## 📦 Installation

### Option 1: Via Go Install (Recommended for Go developers)

```bash
go install github.com/0xzahed/ventox/cmd/ventox@latest
```

### Option 2: Pre-compiled Binaries

Download the latest release tarball for your architecture from the [Releases](https://github.com/0xzahed/ventox/releases) page:

```bash
# Example for Linux x86_64
curl -sSL https://github.com/0xzahed/ventox/releases/latest/download/ventox-linux-amd64.tar.gz | tar -xz
sudo mv ventox-linux-amd64 /usr/local/bin/ventox
```

### Option 3: Build from Source

```bash
git clone https://github.com/0xzahed/ventox.git
cd ventox
make build
sudo make install
```

---

## 🎮 Quick Start & Commands

### CLI Command Overview

| Command | Description | Needs Root? |
| :--- | :--- | :--- |
| `ventox` | Interactive bootable USB creator wizard | Yes (Auto-elevates) |
| `ventox format` | Interactive USB formatter & partition resetter | Yes (Auto-elevates) |
| `ventox list` | List connected USB storage drives and scanned ISO files | No |
| `ventox version` | Display version, build, runtime, and license information | No |
| `ventox help` | Display comprehensive help screen and usage examples | No |

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

### 2. Formatting or Restoring a USB Drive

When you want to wipe or reformat a USB drive, change its partition style, or restore it back to standard universal storage:

```bash
ventox format
```

Ventox provides 6 distinct format profiles:
* **[1] Ventoy Bootable (GPT / UEFI):** Modern UEFI systems & Windows 11.
* **[2] Ventoy Bootable (MBR / Legacy BIOS):** For older motherboards and BIOS compatibility.
* **[3] Clean Standard USB (exFAT):** Universal cross-platform storage (Windows/Mac/Linux) with files > 4GB.
* **[4] Clean Standard USB (FAT32):** Maximum compatibility with embedded systems, stereos, and older consoles.
* **[5] Clean Standard USB (NTFS):** Native Windows filesystem with permissions & journaling.
* **[6] Clean Standard USB (ext4):** High-performance Linux native filesystem.

---

### 3. Inspecting Drives and ISOs

```bash
ventox list
```
Scans and prints all connected removable USB drives and discoverable ISO disk images without touching or modifying any disks.

---

## 🏗️ Architecture & How It Works

```
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Detect USB    │ ───► │   Scan ISOs     │ ───► │  Confirm Target │
│ (lsblk tran=usb)│      │ (~/Downloads/..)│      │ (Prevent Erase) │
└─────────────────┘      └─────────────────┘      └─────────────────┘
                                                           │
                                                           ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│   Safe Unmount  │ ◄─── │ Buffered Copy   │ ◄─── │ Install Ventoy  │
│  (sync + umount)│      │  (with Live UI) │      │   (GPT / UEFI)  │
└─────────────────┘      └─────────────────┘      └─────────────────┘
```

1. **Detection:** Queries `lsblk -J` to find devices where `type == "disk"` and `tran == "usb"`.
2. **Safety Check:** Prompts for explicit verification to ensure no internal hard drives or critical disks are ever overwritten.
3. **Partition Isolation:** Issues lazy unmount calls to clear desktop environment file locks without freezing the UI.
4. **Bootloader Setup:** Boots the Ventoy runtime to format the drive with GPT layout, creating an exFAT partition for storage and a VTOYEFI partition for the bootloader.
5. **Streaming Copy:** Copies the ISO image with buffered I/O, feeds bytes through a terminal progress meter, and executes a full kernel cache `sync` before safely unmounting.

---

## 🛠️ Project Structure

```
ventox/
├── .github/
│   ├── workflows/             # Automated CI & GitHub Release pipelines
│   └── ISSUE_TEMPLATE/        # Standardized bug & feature templates
├── bin/                       # Local build outputs
├── cmd/
│   └── ventox/
│       └── main.go            # Application entrypoint & wizard controller
├── internal/
│   ├── copier/
│   │   └── progress.go        # Buffered file streamer with ETA & rate calculations
│   ├── detector/
│   │   ├── disk.go            # Block device auto-detection logic
│   │   ├── iso.go             # Recursive file discovery for disk images
│   │   └── detector_test.go   # Detector unit tests
│   ├── engine/
│   │   ├── unmount.go         # Safe mount detachment routines
│   │   └── ventoy.go          # Portable Ventoy installer and runner
│   └── ui/
│       ├── colors.go          # ANSI color abstractions and ASCII banners
│       └── menu.go            # Interactive terminal prompts & validation
├── CONTRIBUTING.md            # Guidelines for open source contributors
├── CODE_OF_CONDUCT.md         # Community code of conduct
├── LICENSE                    # MIT Open Source License
├── Makefile                   # Build, test, lint, and install targets
└── README.md                  # Comprehensive project documentation
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

## 🗺️ Roadmap

- [x] Auto-detection for USB devices and ISO images
- [x] GPT/UEFI Ventoy bootloader integration
- [x] Real-time terminal progress bar with ETA & speed
- [x] Defensive auto-unmount for busy desktop file locks
- [ ] Multi-ISO support (copy multiple ISOs onto a single USB)
- [ ] Windows 11 TPM & Secure Boot bypass configuration generator (`ventoy.json`)
- [ ] Automated Homebrew (`brew`) and Arch User Repository (`AUR`) packages
- [ ] Native Bubbletea / TUI full-screen dashboard option

---

## 📄 License

Distributed under the **MIT License**. See [`LICENSE`](LICENSE) for more information.

---

<div align="center">
  <sub>Crafted with ❤️ by <a href="https://github.com/0xzahed">Abu Zahed</a> and open-source contributors.</sub>
</div>
