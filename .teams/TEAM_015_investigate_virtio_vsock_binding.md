# TEAM_015: Investigation - virtio_vsock Driver Binding Failure (Continued)

## Status: **RESOLVED** ✓

## 1. Pre-Investigation Checklist

### 1.1 Team Registration
- **Team ID:** TEAM_015
- **Previous Team:** TEAM_014
- **Summary:** Continued investigation into why `virtio_vsock` fails to bind to PCI device `1af4:1053`.

### 1.2 Bug Report (from TEAM_014)
- **Symptom:** `virtio_vsock` driver does not probe the enumerated PCI device `1af4:1053`.
- **Impact:** No vsock device in guest -> no gvforwarder -> no networking -> no Tailscale/PostgreSQL.
- **Environment:** Android AVF (crosvm), Alpine Linux guest, Custom Kernel.

## 2. Root Cause Confirmed

**TEAM_014 was CORRECT.** The upstream AOSP 6.1 kernel's `virtio_pci_id_table` only has:
```c
{ PCI_DEVICE(PCI_VENDOR_ID_REDHAT_QUMRANET, PCI_ANY_ID) }
```

The modern PCI IDs (0x1040-0x105a) were added as **LOCAL UNCOMMITTED CHANGES** but the patched kernel had NOT been deployed when TEAM_014 investigated.

## 3. Investigation Findings

### Phase 1: Verified Local Patch Exists
The patch in `aosp/drivers/virtio/virtio_pci_common.c` adds modern PCI IDs:
```c
/* Modern devices (0x1040 - 0x107f) */
{ PCI_DEVICE(PCI_VENDOR_ID_REDHAT_QUMRANET, 0x1040), },
...
{ PCI_DEVICE(PCI_VENDOR_ID_REDHAT_QUMRANET, 0x1053), },  // vsock
...
{ PCI_DEVICE(PCI_VENDOR_ID_REDHAT_QUMRANET, 0x105a), },
```

### Phase 2: Added Debug Logging
Added debug printk to `virtio_pci_probe` and `virtio_dev_match` to trace:
1. Modern probe success/failure
2. Device registration
3. Bus matching

### Phase 3: Rebuilt Kernel
- Fixed OpenSSL 3.x compatibility issue in `certs/extract-cert.c`
- Rebuilt with debug logging

### Phase 4: Deployed and Verified
Console log shows:
```
virtio_pci: probing device 1af4:1053
virtio_pci: modern_probe returned 0 for 1af4:1053
virtio_pci: registering virtio device id=19 vendor=6900 for 1af4:1053
virtio_pci: successfully registered 1af4:1053
virtio: matched dev=19 vendor=6900 with driver vmw_vsock_virtio_transport
NET: Registered PF_VSOCK protocol family
```

**All 4 virtio devices now register and bind correctly:**
- id=2 (block) → virtio_blk ✓
- id=4 (rng) → virtio_rng ✓  
- id=5 (balloon) → virtio_balloon ✓
- id=19 (vsock) → vmw_vsock_virtio_transport ✓

## 4. Files Modified

### Kernel Source (Debug - should be cleaned up):
- `aosp/drivers/virtio/virtio_pci_common.c` - Debug printk + modern PCI IDs
- `aosp/drivers/virtio/virtio.c` - Debug printk for bus matching
- `aosp/certs/extract-cert.c` - OpenSSL 3.x compatibility fix

## 5. Handoff Checklist

- [x] Root cause identified and confirmed
- [x] Fix verified (virtio_vsock now binds)
- [x] Kernel builds successfully
- [x] VM boots with vsock driver matching
- [ ] Clean up debug breadcrumbs (optional, helpful for future debugging)
- [ ] Test gvforwarder connectivity (next step for networking)
- [ ] Commit the virtio_pci patch properly

## 6. Next Steps for Future Teams

1. **Test networking:** Verify gvforwarder can connect via vsock
2. **Commit patch:** The virtio_pci modern IDs patch should be committed properly
3. **Consider upstreaming:** This is a legitimate fix that benefits anyone using modern virtio devices with AOSP kernels

