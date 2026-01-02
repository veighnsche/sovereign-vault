# PostgreSQL VM Fix Documentation

**TEAM_036** - January 1, 2026

## Problem Summary

PostgreSQL VM was booting but PostgreSQL service failed to start with the error:

```
FATAL: could not create shared memory segment: Function not implemented
DETAIL: Failed system call was shmget(key=2, size=56, 03600)
```

## Root Cause

The guest kernel was missing `CONFIG_SYSVIPC` (System V IPC support).

PostgreSQL's `initdb` command requires System V shared memory (`shmget`, `semget`, etc.) 
during the database initialization bootstrap phase. Without kernel support for SYSVIPC, 
the `shmget()` syscall returns `ENOSYS` (Function not implemented).

### Additional Issue Found

Tailscale also failed to start because iptables wasn't working:
```
wgengine.NewUserspaceEngine error: creating router: could not get iptables version: exit status 1
```

This is related to missing netfilter kernel options.

## Investigation Process

### Phase 1: Symptom Analysis
- VM boots successfully (kernel starts, init.sh runs)
- PostgreSQL port 5432 not responding
- Kernel log showed `pg_isready` was called

### Phase 2: Hypothesis Formation
1. H1: init.sh fails early due to missing DB_PASSWORD
2. H2: /data mount fails silently
3. H3: exec redirection breaks script execution

### Phase 3: Evidence Gathering
- Retrieved `/var/log/init.log` from the rootfs after VM shutdown
- Found the actual error message from PostgreSQL initdb
- Confirmed SYSVIPC syscall failure

### Phase 4: Root Cause Confirmation
The kernel Image on the device (13MB) was an older build without SYSVIPC.
The local kernel Image (35MB) from `vm/sql/Image` was built with the 
`build-guest-kernel.sh` script which includes `--enable SYSVIPC`.

## Solution

### Quick Fix (Applied)
Push the local kernel with SYSVIPC to the device:

```bash
adb push vm/sql/Image /data/sovereign/vm/sql/Image
./sovereign start --sql
```

### Permanent Fix
The `vm/build-guest-kernel.sh` script already includes the correct kernel options:

```bash
./scripts/config --file "${BUILD_DIR}/.config" \
    --enable SYSVIPC \
    --enable VIRTIO \
    --enable VIRTIO_PCI \
    --enable VIRTIO_NET \
    --enable VIRTIO_BLK \
    # ... (netfilter options for Tailscale)
```

To rebuild the kernel with all required features:

```bash
cd sovereign-vault
./vm/build-guest-kernel.sh
```

**Note**: The build script requires:
- AOSP kernel source at `../aosp`
- Clang toolchain at `../prebuilts/clang/host/linux-x86/clang-r487747c`
- OpenSSL headers/libs at `../prebuilts/kernel-build-tools/linux-x86`

**Known Issue**: The kernel build may fail with linker errors for OpenSSL:
```
ld.lld: error: undefined symbol: OpenSSL_add_all_algorithms
ld.lld: error: undefined symbol: ERR_load_crypto_strings
```

Workaround: Use the pre-built kernel at `vm/sql/Image` which already has all required options.
The OpenSSL linker issue requires either:
1. Installing system OpenSSL dev packages (`libssl-dev`)
2. Or modifying the kernel config to disable `CONFIG_MODULE_SIG` and `CONFIG_SYSTEM_TRUSTED_KEYRING`

## Required Kernel Options

For PostgreSQL and Tailscale to work in AVF VMs:

| Option | Purpose |
|--------|---------|
| `CONFIG_SYSVIPC=y` | PostgreSQL shared memory (shmget, semget) |
| `CONFIG_VIRTIO=y` | crosvm device support |
| `CONFIG_VIRTIO_NET=y` | Network connectivity |
| `CONFIG_VIRTIO_BLK=y` | Block device (rootfs, data disk) |
| `CONFIG_TUN=y` | Tailscale tunnel device |
| `CONFIG_NETFILTER=y` | Tailscale routing |
| `CONFIG_IP_NF_IPTABLES=y` | iptables for Tailscale |
| `CONFIG_NF_TABLES=y` | nftables support |

## Verification

After applying the fix:

```bash
./sovereign test --sql
```

Expected output:
```
=== Testing PostgreSQL VM ===
1. VM process running: ✓ PASS
2. TAP interface (vm_sql): ✓ PASS
3. Tailscale connected: ✓ PASS (100.x.x.x as sovereign-sql)
4. PostgreSQL responding (via TAP): ✓ PASS
5. Can execute query (via TAP): ✓ PASS

=== ALL TESTS PASSED ===
```

## Debugging Tips

### How to Get VM Logs

The init.sh script logs to `/var/log/init.log` on the rootfs. To retrieve:

```bash
# Stop VM first
./sovereign stop --sql

# Pull rootfs from device
adb pull /data/sovereign/vm/sql/rootfs.img /tmp/rootfs.img

# Mount and read logs
e2fsck -f -y /tmp/rootfs.img
mkdir -p /tmp/rootfs_mount
fuse2fs -o ro /tmp/rootfs.img /tmp/rootfs_mount
cat /tmp/rootfs_mount/var/log/init.log

# Cleanup
fusermount -u /tmp/rootfs_mount
```

### Check Kernel Config

To verify kernel has required options:

```bash
# Inside VM (if you have shell access)
zcat /proc/config.gz | grep SYSVIPC

# Or check the built kernel config
cat out/guest-kernel/.config | grep SYSVIPC
```

## Files Modified

- `vm/build-guest-kernel.sh` - Added kernel-build-tools to PATH for OpenSSL

## Lessons Learned

1. **Always check kernel config** when syscalls return ENOSYS
2. **Retrieve VM logs** from rootfs for debugging init script failures
3. **The console warning** ("unable to open an initial console") is cosmetic - 
   the real errors are in the init.log
4. **PostgreSQL requires SYSVIPC** even when using `dynamic_shared_memory_type = mmap` 
   because initdb's bootstrap phase uses shmget regardless
