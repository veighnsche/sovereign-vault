---
description: Patch guest kernel virtio_pci to support modern devices (vsock)
---
## Phase 1: Preparation & Scoping
1. **Goal:** Enable `virtio_vsock` by patching `virtio_pci` to recognize modern PCI IDs.
2. **Current State:** `virtio_pci` only matches legacy IDs (`0x1af4:0x1000-0x103f`).
3. **Required Change:** Add modern PCI ID range (`0x1040-0x107f`) to `virtio_pci_id_table`.

## Phase 2: Implementation
1. **Modify `aosp/include/linux/pci_ids.h`**:
   - Add `#define PCI_DEVICE_ID_VIRTIO_1040_107F 0x1040` (or similar macro if missing).
2. **Modify `aosp/drivers/virtio/virtio_pci_common.c`**:
   - Update `virtio_pci_id_table` to include:
     ```c
     { PCI_DEVICE(PCI_VENDOR_ID_REDHAT_QUMRANET, PCI_ANY_ID) },
     /* Modern devices */
     { PCI_VDEVICE(REDHAT_QUMRANET, PCI_ANY_ID) }, 
     ```
     *Note: Since the table already has `PCI_ANY_ID` for `REDHAT_QUMRANET`, investigate why `1053` is not matching. It might be because `pci_device_id` flags or class mismatch.*
3. **Debug Probe Logic**:
   - Add `printk` in `virtio_pci_probe` to see if it even triggers for `0x1053`.
4. **Rebuild Guest Kernel**:
   - Use `sovereign build --guest-kernel` (custom command via `internal/kernel/kernel.go`).
5. **Redeploy and Test**:
   - `sovereign deploy --sql`
   - `sovereign start --sql`

## Phase 3: Verification
1. Check `dmesg` in guest for `virtio_vsock` binding.
2. Verify `/dev/vsock` exists.
3. Test `gvforwarder` connectivity.
