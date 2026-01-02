# Sovereign Vault Setup Guide

**TEAM_036** - January 2, 2026

## Overview

This guide documents the complete setup process for deploying PostgreSQL, Forgejo, and Vaultwarden VMs on Android 16 with AVF (Android Virtualization Framework).

## Prerequisites

Run preflight checks before starting:

```bash
./sovereign preflight --sql --forge --vault
```

### Required Components

| Component | Purpose | Check |
|-----------|---------|-------|
| Docker | Build VM rootfs images | `docker info` |
| adb | Device communication | `adb devices` |
| qemu-user-static | Cross-arch builds (if not on ARM64) | `/usr/bin/qemu-aarch64-static` |
| .env file | Secrets configuration | Copy from `.env.example` |

### Device Requirements

- Android 16+ device with AVF support (e.g., Pixel 6+)
- Root access via Magisk or similar
- USB debugging enabled

## Build Process

### 1. Configure Secrets

```bash
cp .env.example .env
# Edit .env with your Tailscale auth key and passwords
```

Required variables:
- `TAILSCALE_AUTHKEY` - Get from Tailscale admin console
- `POSTGRES_FORGEJO_PASSWORD` - Database password for Forgejo
- `POSTGRES_VAULTWARDEN_PASSWORD` - Database password for Vaultwarden
- `FORGEJO_SECRET_KEY` - Generate with `openssl rand -hex 32`
- `FORGEJO_INTERNAL_TOKEN` - Generate with `openssl rand -hex 32`
- `VAULTWARDEN_ADMIN_TOKEN` - Generate with `openssl rand -base64 48`

### 2. Build VMs

```bash
# Build requires Docker with root (for cross-arch builds)
sudo ./sovereign build --sql --skip-preflight
sudo ./sovereign build --forge --skip-preflight
sudo ./sovereign build --vault --skip-preflight
```

### 3. Deploy to Device

```bash
./sovereign deploy --sql
./sovereign deploy --forge
./sovereign deploy --vault
```

## Starting VMs

**Important**: VMs must be started in order due to dependencies:
1. PostgreSQL (SQL) - base dependency
2. Forgejo - depends on PostgreSQL
3. Vaultwarden - depends on PostgreSQL

### Start Sequence

```bash
# Start SQL first and wait for it to be ready
adb shell "su -c 'cd /data/sovereign/vm/sql && sh start.sh'"
# Wait 60 seconds for PostgreSQL to initialize

# Check SQL is ready
adb shell "su -c 'nc -z 192.168.100.2 5432 && echo SQL_READY'"

# Start Forgejo
adb shell "su -c 'cd /data/sovereign/vm/forgejo && sh start.sh'"

# Start Vaultwarden
adb shell "su -c 'cd /data/sovereign/vm/vault && sh start.sh'"
```

### Verify All Running

```bash
adb shell "su -c 'ps -ef | grep crosvm | grep -v grep'"
# Should show 3 crosvm processes
```

## Network Configuration

| VM | TAP Interface | IP Address | Ports |
|----|---------------|------------|-------|
| PostgreSQL | vm_sql | 192.168.100.2 | 5432 |
| Forgejo | vm_forge | 192.168.100.3 | 443, 22 |
| Vaultwarden | vm_vault | 192.168.100.4 | 8080, 3012 |

All VMs connect to bridge `vm_bridge` at 192.168.100.1.

## Accessing Services

### Via Tailscale (Recommended)

Each VM registers with Tailscale:
- PostgreSQL: `sovereign-sql.tailXXXXX.ts.net`
- Forgejo: `https://sovereign-forge.tailXXXXX.ts.net`
- Vaultwarden: `https://sovereign-vault.tailXXXXX.ts.net`

### Via TAP Network (Local)

From the Android device:
```bash
# PostgreSQL
nc -z 192.168.100.2 5432

# Forgejo
curl -k https://192.168.100.3

# Vaultwarden
curl -k https://192.168.100.4:8080
```

## Known Issues

### 1. Android Init Killing VMs

**Symptom**: VMs start but die after ~60-90 seconds with message:
```
init: Untracked process (pid: XXXX name: (crosvm) ...) received SIGKILL
```

**Cause**: Android 12+ init kills orphaned processes not associated with a service.

**Workarounds**:
1. Start VMs using `nohup` from a persistent shell
2. Restart VMs when they die
3. Use `setsid` to create a new session (partial fix)

**Permanent Solution**: Run crosvm as an Android service (requires system modification)

### 2. Kernel Missing CONFIG_SYSVIPC

**Symptom**: PostgreSQL fails with:
```
FATAL: could not create shared memory segment: Function not implemented
```

**Cause**: Guest kernel missing System V IPC support.

**Fix**: Use kernel from `vm/sql/Image` which has SYSVIPC enabled.

See `docs/POSTGRESQL_FIX.md` for full details.

### 3. Docker Permission Denied

**Symptom**: Build fails with permission denied on Docker socket.

**Fix**: Run build with sudo:
```bash
sudo ./sovereign build --sql
```

## Troubleshooting

### View VM Logs

```bash
# Console output
adb shell "su -c 'cat /data/sovereign/vm/sql/console.log'"

# Init script log (inside VM)
# Pull rootfs and mount to view /var/log/init.log
```

### Check VM Process

```bash
adb shell "su -c 'ps -ef | grep crosvm'"
```

### Test Port Connectivity

```bash
adb shell "su -c 'nc -z 192.168.100.2 5432'"  # SQL
adb shell "su -c 'nc -z 192.168.100.3 443'"   # Forgejo
adb shell "su -c 'nc -z 192.168.100.4 8080'"  # Vaultwarden
```

### Restart a VM

```bash
# Stop
adb shell "su -c 'pkill -f vm_sql'"

# Start
adb shell "su -c 'cd /data/sovereign/vm/sql && sh start.sh'"
```

## File Locations

### On Build Machine
- `vm/sql/` - PostgreSQL VM files
- `vm/forgejo/` - Forgejo VM files
- `vm/vault/` - Vaultwarden VM files
- `.env` - Secrets configuration

### On Device
- `/data/sovereign/vm/sql/` - PostgreSQL VM
- `/data/sovereign/vm/forgejo/` - Forgejo VM
- `/data/sovereign/vm/vault/` - Vaultwarden VM
- `/data/sovereign/.env` - Secrets (pushed during deploy)

## Summary

| Step | Command |
|------|---------|
| Preflight | `./sovereign preflight --sql --forge --vault` |
| Build | `sudo ./sovereign build --sql --forge --vault --skip-preflight` |
| Deploy | `./sovereign deploy --sql --forge --vault` |
| Start | See start sequence above |
| Test | `./sovereign test --sql --forge --vault` |
