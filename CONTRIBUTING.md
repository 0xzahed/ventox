# Contributing to Ventox ⚡

First off, thank you for considering contributing to **Ventox**! It's people like you that make open source such an amazing community to learn, create, and inspire.

Please take a few moments to review these guidelines before getting started.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Features](#suggesting-features)
  - [Pull Requests](#pull-requests)
- [Development Setup](#development-setup)
- [Project Architecture](#project-architecture)
- [Coding & Git Conventions](#coding--git-conventions)
- [Submitting a Pull Request](#submitting-a-pull-request)

---

## Code of Conduct

This project and everyone participating in it is governed by the [Ventox Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

---

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing [Issues](https://github.com/0xzahed/ventox/issues) to avoid duplicates.

When opening a bug report, please include:
* Operating System and version (`cat /etc/os-release`).
* Output of `lsblk` and the file size of your `.iso`.
* Clear, reproducible steps.
* Terminal output or logs with errors.

### Suggesting Features

Feature requests are always welcome! If you have an idea:
1. Open an issue with the **Feature Request** template.
2. Explain the use case and why it benefits users.
3. Suggest an API or CLI command interface if applicable.

---

## Development Setup

### Prerequisites
* **Go 1.22+**
* Linux operating system (with `lsblk`, `mount`, `umount`)
* `git` and `make`

### Quickstart

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/0xzahed/ventox.git
   cd ventox
   ```

2. Build the binary:
   ```bash
   make build
   ```

3. Run tests:
   ```bash
   make test
   ```

4. Run code checks (linter):
   ```bash
   make lint
   ```

---

## Project Architecture

Ventox follows standard Go package layout principles with strict separation of concerns:

```
ventox/
├── cmd/
│   └── ventox/
│       └── main.go            # CLI entrypoint, flag parsing, flow orchestrator
├── internal/
│   ├── domain/                # Clean domain models, typed errors, write plans, journals
│   │   ├── device.go          # Storage device representations & formatting
│   │   ├── errors.go          # Standard typed errors (ErrSystemDisk, ErrDeviceRemoved, etc.)
│   │   ├── image.go           # Disk image metadata & classifications
│   │   ├── operation.go       # Step journal records & verification reports
│   │   └── write_plan.go      # Deterministic, immutable write plans
│   ├── device/                # Cross-platform device discovery (Linux, macOS, Windows)
│   │   ├── provider.go        # DeviceProvider interface contract
│   │   ├── provider_linux.go  # Linux block device detection (lsblk / sysfs)
│   │   ├── provider_darwin.go # macOS physical disk discovery (diskutil -plist)
│   │   ├── provider_windows.go# Windows storage discovery (PowerShell / WMI)
│   │   └── fake/              # In-memory sparse block simulated disk for tests
│   ├── iso/                   # Image discovery & inspection
│   │   ├── checksum.go        # Context-cancellable streaming SHA-256 calculator
│   │   ├── discover.go        # Standard user directory image scanner (sudo-aware)
│   │   └── inspect.go         # Pure Go ISO9660 & UDF inspector & El Torito boot parser
│   ├── safety/                # Preflight validation & system disk protection
│   │   ├── capacity.go        # Payload vs target size calculation
│   │   ├── identity.go        # Anti-race condition device identity revalidation
│   │   ├── preflight.go       # Composite write plan validator
│   │   └── system_disk.go     # System disk guard (root, /boot, /home, swap, C:)
│   ├── engine/                # Native execution engine
│   │   ├── planner.go         # Autonomous WritePlan generator
│   │   ├── executor.go        # Step journal executor with cancellation & dry-run
│   │   └── progress.go        # Live progress event dispatcher
│   ├── writer/                # Low-level media writers & formatters
│   │   ├── format.go          # Multi-OS filesystem formatter (exFAT, FAT32, NTFS, ext4, APFS)
│   │   ├── raw.go             # 4 MiB chunk aligned streaming raw disk writer
│   │   ├── unmount.go         # Cross-platform device & partition unmounting
│   │   ├── verify.go          # Readback SHA-256 target verification engine
│   │   └── windows_bypass.go  # Windows 11 autounattend.xml answer file generator
│   └── ui/                    # User interface & terminal rendering
│       ├── colors.go          # ANSI colors, step banners, and badges
│       ├── help.go            # CLI documentation & version displays
│       ├── menu.go            # Interactive prompts & selection menus
│       └── progress.go        # Inline terminal progress bars & plan confirmation
├── docs/                      # Architectural Decision Records (ADRs) and policies
├── Makefile                   # Multiplatform build targets (Linux, macOS, Windows)
├── LICENSE                    # MIT License
└── README.md                  # Comprehensive user and architectural documentation
```

---

## Coding & Git Conventions

### Go Code Style
* Run `go fmt ./...` and `go vet ./...` before committing.
* Write unit tests for new logic whenever possible (`*_test.go`).
* Keep functions focused, modular, and error messages actionable.

### Commit Messages
We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

* `feat: add support for NVMe drive auto-detection`
* `fix: handle target busy when unmounting partitions`
* `docs: update troubleshooting guide for Secure Boot`
* `refactor: simplify ISO file tree traversal`
* `test: add unit tests for progress writer`

---

## Submitting a Pull Request

1. Create a feature branch:
   ```bash
   git checkout -b feat/my-awesome-feature
   ```
2. Commit your changes with meaningful commit messages.
3. Push to your fork:
   ```bash
   git push origin feat/my-awesome-feature
   ```
4. Open a Pull Request against the `main` branch of `0xzahed/ventox`.
5. Fill out the PR template with relevant details and test outcomes.

Thank you for building a better open-source tool for everyone! 🚀
