# Security and Destructive-Operation Policy

Writing bootable images or repartitioning storage devices is inherently destructive. Ventox implements a rigorous multi-tier safety architecture to guarantee host system protection and eliminate data loss risks.

---

## 1. System Disk Protection Engine

Ventox never permits destructive operations on devices containing the host operating system, active system mounts, swap spaces, or internal startup volumes.

### Critical System Mounts Enforced
The preflight engine (`internal/safety/system_disk.go`) actively verifies all target block devices and all subordinate partitions against protected paths:
- Linux: `/`, `/boot`, `/boot/efi`, `/home`, `/usr`, `/var`, `/etc`, and active entries in `/proc/swaps`.
- macOS: `/`, `/System/Volumes/Data`, and the active startup disk reported by `diskutil info /`.
- Windows: `C:`, `C:\`, and devices flagged as `IsSystem` or `IsBoot`.

If any target device or its child partitions host any of these mount points or active swap spaces, Ventox immediately rejects the operation with a typed `domain.ErrSystemDisk` error and terminates.

---

## 2. Device Identity Revalidation (Anti-Race Invariant)

In terminal and scripted environments, a user may select a target device (`/dev/sdb`), but before the confirmation dialog is accepted, the drive could be unplugged or replaced by a different disk.

To prevent write operations to an unexpected device:
1. When a device is initially selected, Ventox records its baseline identity snapshot:
   - Device Path
   - Capacity in bytes
   - Sector size
   - Serial number
   - Vendor / Model identifier
2. Immediately before opening the target device handle for writing, Ventox queries the storage provider for the current physical device state (`safety.RevalidateDevice`).
3. If the capacity, serial number, or vendor deviates from the baseline, the operation is aborted with `domain.ErrIdentityMismatch`.

---

## 3. Capacity Preflight Verification

Before initiating partitioning or writing:
- The required payload capacity is calculated (image file size + partition table overhead).
- The target storage device's advertised capacity is verified (`safety.ValidateCapacity`).
- If `target.SizeBytes < requiredBytes`, the operation is aborted with `domain.ErrInsufficientCapacity`.

---

## 4. Dry-Run Simulation Mode

Ventox provides `--dry-run` to allow testing, automated CI validation, and execution planning without writing a single byte to the destination device:
- The entire preflight safety suite is executed (system disk check, capacity calculation, device identity validation).
- An immutable `domain.WritePlan` is generated and presented.
- The executor logs each planned step into the `domain.OperationJournal` with simulated timings.
- The target device is never opened with write flags (`os.O_WRONLY`, `os.O_RDWR`).

---

## 5. Explicit User Confirmation

Unless explicitly overridden by the automated script flag (`--yes` / `-y`), Ventox displays an unmistakable, high-visibility red warning banner displaying:
- Target device node
- Target capacity and model
- Loaded image name and classification
- Planned creation mode and filesystem
- Every step that will be executed

The user must explicitly type `YES` to authorize execution. Any other input immediately cancels the operation without modifying the target media.
