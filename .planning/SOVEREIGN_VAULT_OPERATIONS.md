# Sovereign Vault Operations Guide

**Purpose:** This document captures ALL the issues, gotchas, and solutions discovered while making the Sovereign Vault CLI and VMs work. Follow this guide exactly to avoid rediscovering these problems.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Critical Issues and Solutions](#2-critical-issues-and-solutions)
3. [VM Startup Sequence](#3-vm-startup-sequence)
4. [CLI Commands Reference](#4-cli-commands-reference)
5. [Troubleshooting Guide](#5-troubleshooting-guide)
6. [Common Gotchas](#6-common-gotchas)

---

## 1. Architecture Overview

### Components

| Component | Location | Purpose |
|-----------|----------|---------|
| **SQL VM** | `/data/sovereign/vm/sql/` | PostgreSQL database |
| **Vault VM** | `/data/sovereign/vm/vault/` | Vaultwarden password manager |
| **Forge VM** | `/data/sovereign/vm/forgejo/` | Forgejo git server |
| **Daemon Script** | `/data/sovereign/sovereign_start.sh` | Keeps VMs alive |
| **Bridge** | `vm_bridge` (192.168.100.1/24) | VM networking |

### Network Layout

```
Host (Android)
├── vm_bridge: 192.168.100.1/24
│   ├── vm_sql (TAP): 192.168.100.2
│   ├── vm_forge (TAP): 192.168.100.3
│   └── vm_vault (TAP): 192.168.100.4
└── wlan0 → Internet (NAT via iptables)
```

### Tailscale FQDNs

- SQL: No Tailscale (internal only)
- Vault: `sovereign-vault.tail5bea38.ts.net`
- Forge: `sovereign-forge.tail5bea38.ts.net` (or `-2`, `-3` if re-registered)

---

## 2. Critical Issues and Solutions

### Issue #1: VMs Die After ~90 Seconds

**Symptom:** VMs start but die within 90 seconds with no error.

**Root Cause:** Android 12+ Phantom Process Killer kills processes with `ppid=1` that aren't tracked by init.

**Solution:** Use `sovereign_start.sh` daemon script that:
1. Disables phantom process killer before starting VMs
2. Stays alive as parent process so VMs aren't orphaned
3. Sets OOM score to -1000 for VM processes

**Critical Code:**
```bash
# In sovereign_start.sh
disable_process_killers() {
    /system/bin/device_config set_sync_disabled_for_tests persistent
    /system/bin/device_config put activity_manager max_phantom_processes 2147483647
    settings put global settings_enable_monitor_phantom_procs false
}
```

---

### Issue #2: CLI `sovereign start` Command Hangs or VM Dies

**Symptom:** Running `./sovereign start --sql` causes the command to hang forever, or VM dies immediately after CLI exits.

**Root Cause:** `adb shell` doesn't properly background processes. Tried:
- `nohup ... &` → adb waits for all children
- `setsid ... &` → Android shell doesn't properly detach
- `(... &)` subshell → Syntax error on Android's mksh/ash shell

**Solution:** Use Go's `syscall.SysProcAttr` to detach the process:

```go
// In internal/vm/common/lifecycle.go
cmd := exec.Command("adb", "shell", "su", "-c", startCmd)
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,  // Create new process group
    Pgid:    0,     // Use the new process's PID as PGID
}
cmd.Start()
cmd.Process.Release()  // Don't wait or kill on exit
```

**NEVER DO:**
```bash
# These DON'T work via adb shell:
adb shell "nohup /data/sovereign/sovereign_start.sh start sql &"
adb shell "setsid /data/sovereign/sovereign_start.sh start sql &"
adb shell "(... &)"  # Syntax error!
```

---

### Issue #3: TAP Interface Shows NO-CARRIER

**Symptom:** `ip link show vm_sql` shows `NO-CARRIER` even though VM is running.

**Root Cause:** TAP interface must be created immediately before starting crosvm, not separately.

**Solution:** Create TAP and start crosvm in same command sequence:
```bash
ip tuntap add dev vm_sql mode tap && \
ip link set vm_sql master vm_bridge && \
ip link set vm_sql up && \
crosvm run ... &
```

---

### Issue #4: PostgreSQL Password Authentication Failed

**Symptom:** Vault VM crashes with "FATAL: password authentication failed for user 'vaultwarden'"

**Root Cause:** Password mismatch between:
1. Password set when SQL VM created the user (in `init.sh`)
2. Password Vault VM tries to use (from kernel cmdline)

**Solution:** Ensure `.env` file has correct password and is deployed before VMs start:
```bash
# .env must contain:
POSTGRES_VAULTWARDEN_PASSWORD=<same-as-secrets>
POSTGRES_FORGEJO_PASSWORD=<same-as-secrets>
```

**If already broken:** Restart SQL VM first, then restart Vault VM:
```bash
./sovereign stop --vault
./sovereign stop --sql
./sovereign start --sql
# Wait for PostgreSQL ready
./sovereign start --vault
```

---

### Issue #5: TLS Certificate Validation Fails

**Symptom:** Services fail with TLS/certificate errors during boot.

**Root Cause:** VMs have hardcoded dates in `init.sh` that are in the past. Let's Encrypt certs are "not yet valid" if system date is before cert issuance.

**Solution:** Update date in each VM's `init.sh`:
```bash
# In vm/vault/init.sh and vm/forgejo/init.sh
date -s "2026-01-02 12:00:00"  # Update to current date!
```

**GOTCHA:** This must be updated periodically or VMs will fail after rebuild!

---

### Issue #6: Tailscale Creates Multiple Machines

**Symptom:** Tailscale shows `sovereign-vault-1`, `sovereign-vault-2`, etc.

**Root Cause:** Each time VM boots with fresh `data.img`, Tailscale sees it as new machine.

**Solution:** 
1. Use `--fresh-data` flag only when intentionally resetting
2. Use `./sovereign clean --vault` to remove old Tailscale entries
3. Check `/var/lib/tailscale/` state inside VM's `data.img`

---

### Issue #7: Tests Pass But Browser Can't Connect

**Symptom:** `./sovereign test --vault` passes, but browser shows "ERR_CONNECTION_REFUSED"

**Possible Causes:**
1. **Browser DNS cache** → Clear cache or use incognito
2. **Host DNS cache** → `sudo systemd-resolve --flush-caches`
3. **Tailscale not connected on host** → `tailscale status`
4. **Test using wrong FQDN** → Fixed in GetTailscaleFQDN()

**Verify with:**
```bash
curl -sk https://sovereign-vault.tail5bea38.ts.net/
# Should return HTML
```

---

## 3. VM Startup Sequence

### Correct Order (Dependencies)

```
1. SQL VM (no dependencies)
   ↓ Wait for PostgreSQL port 5432
2. Vault VM (depends on SQL)
   ↓ Wait for INIT COMPLETE
3. Forge VM (depends on SQL)
```

### Full Startup Commands

```bash
# Build (only needed once or after code changes)
./sovereign build --sql
./sovereign build --vault
./sovereign build --forge

# Deploy to device
./sovereign deploy --sql
./sovereign deploy --vault
./sovereign deploy --forge

# Start in order
./sovereign start --sql
# Wait for "PostgreSQL is ready"
./sovereign start --vault
./sovereign start --forge

# Verify
./sovereign test --sql
./sovereign test --vault
./sovereign test --forge
```

### Quick Restart (If Already Deployed)

```bash
./sovereign stop --vault --forge --sql
./sovereign start --sql
# Wait 30 seconds
./sovereign start --vault
./sovereign start --forge
```

---

## 4. CLI Commands Reference

| Command | Description |
|---------|-------------|
| `sovereign preflight --<vm>` | Check prerequisites |
| `sovereign build --<vm>` | Build VM rootfs and data disk |
| `sovereign deploy --<vm>` | Push files to device |
| `sovereign start --<vm>` | Start VM via daemon |
| `sovereign stop --<vm>` | Stop running VM |
| `sovereign test --<vm>` | Run connectivity tests |
| `sovereign diagnose --<vm>` | Comprehensive debugging |
| `sovereign fix --<vm>` | **NEW:** Auto-detect and fix issues |
| `sovereign status --<vm>` | Quick status check |
| `sovereign clean --<vm>` | Remove Tailscale registrations |
| `sovereign remove --<vm>` | Remove VM from device |

### Diagnose Command Output

```bash
./sovereign diagnose --vault
```

Shows:
1. Process status with PID and uptime
2. TAP interface state (UP/DOWN/NO-CARRIER)
3. Bridge network status
4. Port connectivity tests (TAP)
5. Tailscale status (online/offline)
6. HTTPS connectivity with timing
7. Recent console output (last 10 lines)
8. Error detection (greps for FATAL/ERROR)
9. Recommendations

### Fix Command - Auto-Repair

```bash
./sovereign fix --vault
```

Automatically detects and fixes:
1. **Bridge network** - Creates vm_bridge if missing, adds IP, brings UP
2. **Process killers** - Disables Android phantom process killer
3. **VM process** - Restarts if dead, cleans stale state first
4. **TAP interface** - Brings UP, attaches to bridge
5. **Stale state** - Removes old vm.sock/vm.pid files
6. **Dependencies** - Checks if SQL is running before starting Vault/Forge
7. **Tailscale** - Verifies registration status
8. **Verification** - Runs HTTPS test after fixes

---

## 5. Troubleshooting Guide

### VM Won't Start

```bash
# 1. Check if daemon is running
adb shell "ps -ef | grep sovereign_start"

# 2. Check daemon log
adb shell "tail -50 /data/sovereign/daemon.log"

# 3. Check if crosvm exists
adb shell "ls -la /apex/com.android.virt/bin/crosvm"

# 4. Try manual start (for debugging only)
adb shell "su -c '/data/sovereign/sovereign_start.sh start sql'"
```

### VM Starts But Service Not Accessible

```bash
# 1. Check console.log
adb shell "tail -100 /data/sovereign/vm/vault/console.log"

# 2. Check for errors
adb shell "grep -i error /data/sovereign/vm/vault/console.log"

# 3. Check TAP interface
adb shell "ip link show vm_vault"

# 4. Check bridge
adb shell "ip addr show vm_bridge"

# 5. Test port locally
adb shell "nc -zv 192.168.100.4 443"
```

### Tailscale Not Working

```bash
# 1. Check Tailscale on host
tailscale status | grep sovereign

# 2. Check inside VM (via console.log)
adb shell "grep tailscale /data/sovereign/vm/vault/console.log"

# 3. Look for auth key issues
adb shell "grep -i 'auth\|key\|expired' /data/sovereign/vm/vault/console.log"
```

---

## 6. Common Gotchas

### GOTCHA #1: Process Detachment
**NEVER** use manual `adb shell` commands with `&` to start VMs. Always use the CLI which properly detaches processes.

### GOTCHA #2: Startup Grace Period
VMs take 10-15 seconds to set up networking before crosvm starts. The CLI waits for this. Don't assume VM is dead if `console.log` is empty initially.

### GOTCHA #3: Date Synchronization
VMs have hardcoded dates. Update `init.sh` files when rebuilding after long periods.

### GOTCHA #4: Tailscale Auth Key
Auth key in `.env` (`TAILSCALE_AUTHKEY`) expires. Check Tailscale admin console if VMs can't register.

### GOTCHA #5: Clean State Before Restart
Always clean old state before starting:
```bash
rm -f /data/sovereign/vm/<name>/vm.sock
rm -f /data/sovereign/vm/<name>/vm.pid
rm -f /data/sovereign/vm/<name>/console.log
```
The CLI does this automatically in `StartVM()`.

### GOTCHA #6: FQDN vs Hostname
Tailscale status shows short hostname (`sovereign-vault`). Tests and browsers need full FQDN (`sovereign-vault.tail5bea38.ts.net`). The CLI's `GetTailscaleFQDN()` handles this.

### GOTCHA #7: Multiple VMs Can Share Kernel
Vault and Forge VMs can use SQL's kernel (`SharedKernel: true` in config). Only SQL needs `vm/sql/Image`.

### GOTCHA #8: crosvm Needs LD_LIBRARY_PATH
```bash
export LD_LIBRARY_PATH=/apex/com.android.virt/lib64:/system/lib64
```
Without this, crosvm fails silently.

### GOTCHA #9: Bridge IP Must Be .1
The bridge IP must be `192.168.100.1`. VMs expect this as their gateway.

### GOTCHA #10: Secrets Must Be Deployed
The `.env` file must be deployed to `/data/sovereign/.env` BEFORE starting VMs. The daemon script sources it.

---

## Quick Reference Card

```
┌─────────────────────────────────────────────────────────────────┐
│                    SOVEREIGN VAULT QUICK REF                     │
├─────────────────────────────────────────────────────────────────┤
│ START ORDER:  sql → (wait 30s) → vault → forge                  │
│                                                                  │
│ KEY PATHS:                                                       │
│   /data/sovereign/vm/<name>/     VM files                       │
│   /data/sovereign/daemon.log     Daemon output                  │
│   /data/sovereign/.env           Secrets/passwords              │
│   /data/sovereign/sovereign_start.sh  Daemon script             │
│                                                                  │
│ NETWORK:                                                         │
│   Bridge: 192.168.100.1/24                                      │
│   SQL:    192.168.100.2:5432                                    │
│   Forge:  192.168.100.3:443                                     │
│   Vault:  192.168.100.4:443                                     │
│                                                                  │
│ DEBUG:                                                           │
│   ./sovereign diagnose --<vm>    Comprehensive debug            │
│   adb shell "tail -f /data/sovereign/vm/<name>/console.log"     │
│                                                                  │
│ IF STUCK:                                                        │
│   1. Stop all VMs                                                │
│   2. Kill any orphan crosvm: adb shell "pkill crosvm"           │
│   3. Start in order: sql → vault → forge                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## File Checksums (for verification)

After successful deployment, verify key files:
```bash
adb shell "md5sum /data/sovereign/sovereign_start.sh"
adb shell "md5sum /data/sovereign/.env"
adb shell "ls -la /data/sovereign/vm/*/rootfs.img"
```

---

*Last Updated: 2026-01-02 by TEAM_041*
*Issues Documented: 7 major, 10 gotchas*
