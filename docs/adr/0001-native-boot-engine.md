# ADR 0001: Ventox Native Boot Media Engine Architecture

**Status:** Accepted  
**Date:** October 30, 2026  
**Author:** Abu Zahed (0xzahed) & Antigravity Systems Team  

## Context
Ventox was originally conceived to provide a zero-configuration CLI bootable media creator on Linux and macOS. The initial proof of concept relied on third-party scripts and rudimentary shell-outs. To achieve production reliability comparable to Rufus, Ventox requires an independent, robust, modular systems architecture written natively in Go without external runtime dependencies.

## Decision

We replace all ad-hoc script and command executions with a layered, interface-driven Native Boot Media Engine following the workflow:
```
Discover → Inspect → Plan → Validate → Confirm → Execute → Verify → Report
```

### 1. Architectural Layers & Boundaries
```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                      │
│        (CLI Commands, Interactive TUI, Signal Handling)     │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│                    Safety & Planning Engine                 │
│      (WritePlanner, SystemDiskDetector, IdentityValidator)   │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│                     Media Creation Engine                   │
│         (RawWriter, WindowsInstallerWriter, Verifier)       │
└──────────────────────────────┬──────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────┐
│                      Platform Adapters                      │
│     (Linux Block/Sysfs, macOS DiskArbitration, Windows)     │
└─────────────────────────────────────────────────────────────┘
```

### 2. Invariants & Safety Mandates
1. **Never Touch System Disks:** The system disk (`/`, `/boot`, `/home`, active swap) is unconditionally rejected by `SystemDiskDetector`.
2. **Planning Separated from Execution:** Every operation must first construct an immutable `WritePlan`. A `--dry-run` flag executes planning and validation without modifying storage.
3. **Capacity Invariant:** `TargetDevice.SizeBytes >= Image.SizeBytes` is enforced before any destructive action.
4. **Context & Cancellation:** All disk I/O, sector streaming, and filesystem operations take `context.Context` to guarantee instantaneous and clean cleanup upon interruption.
5. **No Swallowed Errors:** All error values from partitioning, formatting, copying, syncing, and verification are wrapped and propagated with actionable diagnostic messages.
6. **Cross-Platform Compilation:** Shared packages must compile cleanly on Linux, macOS (Darwin), and Windows. OS-specific operations are abstracted behind clean interfaces.

## Consequences
- **Positive:** Maximum safety against host data loss; zero dependency on Ventoy; native macOS and Windows compatibility; testable via a fake storage backend.
- **Negative:** Increased codebase surface area; requires maintaining OS-specific adapters for storage discovery and partition management.
