# Guest Kernel Build Guide

**TEAM_011** — Documented after making mistakes and learning from them.

---

## Overview

The Sovereign Vault runs **Alpine Linux** userspace inside VMs, but the **kernel** is built from the AOSP tree. This is confusing at first — this document clarifies it.

```
┌─────────────────────────────────────┐
│           Guest VM                  │
│  ┌───────────────────────────────┐  │
│  │  Alpine Linux Userspace       │  │
│  │  - PostgreSQL                 │  │
│  │  - Tailscale                  │  │
│  │  - gvforwarder                │  │
│  └───────────────────────────────┘  │
│  ┌───────────────────────────────┐  │
│  │  AOSP-derived Kernel          │  │
│  │  (built from aosp/ tree)      │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

---

## Build Command

```bash
sovereign build --guest-kernel
```

This is the ONLY way to build the guest kernel. Do NOT run manual `make` commands.

---

## What The CLI Does

1. Sets up environment with AOSP prebuilt OpenSSL
2. Runs `defconfig` then `olddefconfig` (non-interactive)
3. Merges `sovereign_guest.fragment` 
4. Builds the kernel Image
5. Copies Image to `vm/sql/Image`

---

## OpenSSL Build Fix

### The Problem

System OpenSSL 3.x removed `engine.h`, causing kernel build failures:

```
fatal error: 'openssl/engine.h' file not found
```

### The WRONG Fix (Do Not Do This)

```c
// DO NOT modify aosp/certs/extract-cert.c!
#if OPENSSL_VERSION_NUMBER < 0x30000000L
#include <openssl/engine.h>  // WRONG - hacking source
#endif
```

### The CORRECT Fix

Use AOSP prebuilt OpenSSL/BoringSSL:

```bash
export HOSTCFLAGS="-I$KERNEL_ROOT/prebuilts/kernel-build-tools/linux-x86/include"
export HOSTLDFLAGS="-L$KERNEL_ROOT/prebuilts/kernel-build-tools/linux-x86/lib64 -Wl,-rpath,$KERNEL_ROOT/prebuilts/kernel-build-tools/linux-x86/lib64"
```

The CLI sets these automatically in `internal/kernel/kernel.go`.

---

## AOSP Prebuilt Tools

Location: `prebuilts/kernel-build-tools/linux-x86/`

```
prebuilts/kernel-build-tools/linux-x86/
├── include/
│   └── openssl/
│       ├── engine.h      ← This exists! Use it!
│       ├── bio.h
│       ├── pem.h
│       └── ...
└── lib64/
    ├── libcrypto.so
    └── ...
```

---

## Kernel Configuration

### sovereign_guest.fragment

Location: `private/devices/google/raviole/sovereign_guest.fragment`

Critical configs:

```
# Required for gvforwarder
CONFIG_TUN=y
CONFIG_VSOCKETS=y
CONFIG_VIRTIO_VSOCKETS=y

# Required for PostgreSQL
CONFIG_SYSVIPC=y

# Required for Tailscale
CONFIG_NETFILTER=y
CONFIG_WIREGUARD=y

# Disable for guest (doesn't run VMs itself)
# CONFIG_KVM is not set

# KEEP ENABLED - security features
CONFIG_KEYS=y
CONFIG_INTEGRITY=y
CONFIG_SYSTEM_TRUSTED_KEYRING=y
```

### What NOT To Disable

Never disable these to "fix" build issues:

- `CONFIG_KEYS` — kernel keyring
- `CONFIG_INTEGRITY` — file integrity verification
- `CONFIG_SYSTEM_TRUSTED_KEYRING` — trusted certificates

If you're tempted to disable security features to make a build pass, **you're solving the wrong problem**.

---

## Common Errors

### `engine.h not found`

**Cause:** Using system OpenSSL 3.x instead of AOSP prebuilts.

**Fix:** Use CLI which sets correct HOSTCFLAGS/HOSTLDFLAGS.

### `pkvm_load_early_modules undefined`

**Cause:** CONFIG_KVM=y but pKVM module code is incomplete.

**Fix:** Add `# CONFIG_KVM is not set` to fragment. Guest doesn't need KVM.

### `virtio_vsock not binding`

**Status:** Unresolved blocker as of TEAM_011.

**Symptoms:** PCI device 1af4:1053 detected but driver doesn't bind.

**Investigation needed:** Check kernel driver initialization order.

---

## Output Locations

| File | Description |
|------|-------------|
| `out/guest-kernel/.config` | Final kernel config |
| `out/guest-kernel/arch/arm64/boot/Image` | Built kernel |
| `sovereign/vm/sql/Image` | Deployed kernel (copied by CLI) |

---

## Verification

After building, verify security configs:

```bash
grep -E 'CONFIG_KEYS=|CONFIG_INTEGRITY=|CONFIG_SYSTEM_TRUSTED_KEYRING=' out/guest-kernel/.config
```

Expected output:

```
CONFIG_KEYS=y
CONFIG_INTEGRITY=y
CONFIG_SYSTEM_TRUSTED_KEYRING=y
```

If any show `=n` or `is not set`, the build is WRONG.
