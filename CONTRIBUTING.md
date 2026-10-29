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

Ventox follows standard Go package layout principles:

```
ventox/
├── bin/                       # Local build outputs
├── cmd/
│   └── ventox/
│       └── main.go            # CLI entrypoint, argument parsing, flow orchestrator
├── internal/
│   ├── copier/
│   │   └── progress.go        # Buffered file copy with real-time ETA & speed
│   ├── detector/
│   │   ├── disk.go            # Block device querying via lsblk JSON
│   │   └── iso.go             # File tree crawler for ISO disk images
│   ├── engine/
│   │   ├── unmount.go         # Safe namespace/lazy unmounting
│   │   └── ventoy.go          # Ventoy runtime bootstrap and disk formatting
│   └── ui/
│       ├── colors.go          # ANSI terminal styling
│       └── menu.go            # Interactive prompts and confirmation gates
├── Makefile
├── LICENSE
└── README.md
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
