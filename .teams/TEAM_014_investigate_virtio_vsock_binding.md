# TEAM_014: Investigation - virtio_vsock Driver Binding Failure

## Status: **RESOLVED by TEAM_015** ✓
See `TEAM_015_investigate_virtio_vsock_binding.md` for full resolution.

## 1. Pre-Investigation Checklist

### 1.1 Team Registration
- **Team ID:** TEAM_014
- **Previous Team:** TEAM_013
- **Summary:** Investigating why `virtio_vsock` fails to bind to PCI device in Alpine Linux guest.

### 1.2 Bug Report
- **Symptom:** `virtio_vsock` driver does not probe the enumerated PCI device `1af4:1053`.
- **Impact:** No vsock device in guest -> no gvforwarder connection -> no networking -> no Tailscale/PostgreSQL.
- **Environment:** Android AVF (crosvm), Alpine Linux guest, Custom Kernel.

### 1.3 Context
- TEAM_013 verified `CONFIG_VIRTIO_VSOCKETS=y` and `CONFIG_SYSVIPC=y`.
- Only `virtio_blk` binds (as `virtio0`).
- PCI device is seen by kernel: `[0.795404] pci 0000:00:04.0: [1af4:1053] type 00 class 0x028000`.

### 1.4 Reproducibility
- Consistently fails on VM start.

## 2. Investigation Structure

### Phase 1: Understand the Symptom
- **Actual Behavior:** PCI device `0000:00:04.0` [1af4:1053] exists but is not claimed by any driver.
- **Expected Behavior:** `virtio_vsock` should claim it and create `/dev/vsock`.
- **Delta:** Missing driver binding for vsock PCI device.

### Phase 2: Hypotheses
1. **H1: Missing CONFIG_VIRTIO_PCI** - While virtio_blk works, it might be using a different transport or virtio_pci is missing a specific sub-config.
2. **H2: Kernel config mismatch** - `CONFIG_VIRTIO_VSOCKETS` is `y` but maybe `CONFIG_VIRTIO_VSOCKETS_COMMON` or similar is missing.
3. **H3: PCI ID mismatch** - The kernel version might expect a different PCI ID or layout for virtio-vsock.
4. **H4: Deployment Failure** - The kernel running on the device is NOT the one with the config changes.

### Phase 3: Test Hypotheses
- [x] Check `.config` for `CONFIG_VIRTIO_PCI` and all `VIRTIO` related flags.
  - **Result:** Confirmed `CONFIG_VIRTIO_PCI=y`, `CONFIG_VIRTIO_VSOCKETS=y`, `CONFIG_VIRTIO_VSOCKETS_COMMON=y`.
- [x] Verify symbols in System.map.
  - **Result:** `virtio_vsock_probe` and `virtio_vsock_driver` symbols exist.
- [x] Verify `Image` timestamp/md5 on device against local build.
  - **Result:** MD5 match (`382290828799abe630583fb64e2ba7d0`). The correct kernel is deployed.
- [x] Check `dmesg` for any "virtio_pci" or "pci" errors.
  - **Result:** `virtio_blk` binds to `virtio0` (using legacy ID `0x1001`), but `1af4:1053` (vsock) is ignored. No errors, just no match.

## 3. Phase 4 — Narrow Down to Root Cause

### Root Cause Found
The `virtio_pci` driver in the AOSP 6.1 guest kernel is missing the PCI IDs for modern (v1.0+) devices.

**Evidence:**
1.  **PCI ID Table:** In `drivers/virtio/virtio_pci_common.c`, the `virtio_pci_id_table` only contains:
    ```c
    { PCI_DEVICE(PCI_VENDOR_ID_REDHAT_QUMRANET, PCI_ANY_ID) }
    ```
    This vendor ID (`0x1af4`) with "any ID" usually covers legacy devices (`0x1000-0x103f`), but modern devices use a specific range starting at `0x1040`.
2.  **Crosvm vs. Kernel:** Crosvm exposes vsock as `1af4:1053`. The kernel's `virtio_pci` driver probe function is never called for this ID because it's not in the table.
3.  **Contrast with virtio-blk:** `virtio_blk` works because crosvm (or AVF) likely exposes it as a legacy-compatible device (`0x1001`), which matches the broad `0x1af4` rule in the old table. Vsock has no legacy equivalent.

### Causal Chain
1.  Crosvm/AVF creates a Virtio-Vsock PCI device with ID `1af4:1053`.
2.  Guest kernel scans PCI bus and finds the device.
3.  `virtio_pci` driver is registered but its `id_table` does not include `0x1053` (or the modern range).
4.  Linux PCI core does not call `virtio_pci_probe` for the vsock device.
5.  No `virtio_device` is registered for vsock.
6.  The `virtio_vsock` transport driver sits idle with no hardware to bind to.

## 4. Phase 5 — Decision: Fix or Plan

**Decision:** Create a bugfix plan.
**Reasoning:** Fixing this requires patching the guest kernel's `virtio_pci` driver to include modern PCI IDs. This is a critical kernel change and should be planned carefully.
