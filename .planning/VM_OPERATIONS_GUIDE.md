# Sovereign Vault VM Operations Guide

> **⚠️ IMPORTANT:** See also [SOVEREIGN_VAULT_OPERATIONS.md](./SOVEREIGN_VAULT_OPERATIONS.md) for detailed troubleshooting, gotchas, and issue resolutions discovered by TEAM_041.

## Overview

This guide documents how to build, deploy, start, stop, test, and clean up the three VMs:
- **SQL** (PostgreSQL) - Database server
- **Vault** (Vaultwarden) - Password manager  
- **Forge** (Forgejo) - Git server

## Prerequisites

1. **Host machine**: Linux with Docker installed
2. **Android device**: Connected via ADB with KernelSU root
3. **Environment**: `.env` file with Tailscale keys and database passwords

## CLI Commands

All operations use the `sovereign` CLI tool:

```bash
# Build the CLI first
go build -o sovereign ./cmd/sovereign

# Available commands
./sovereign build    --sql|--vault|--forge   # Build VM rootfs
./sovereign deploy   --sql|--vault|--forge   # Push to device
./sovereign start    --sql|--vault|--forge   # Start VM
./sovereign stop     --sql|--vault|--forge   # Stop VM
./sovereign test     --sql|--vault|--forge   # Run connectivity tests
./sovereign diagnose --sql|--vault|--forge   # Comprehensive debugging
./sovereign fix      --sql|--vault|--forge   # TEAM_041: Auto-detect and fix issues
./sovereign remove   --sql|--vault|--forge   # Remove from device
./sovereign clean    --sql|--vault|--forge   # Clean Tailscale registrations
./sovereign status   --sql|--vault|--forge   # Show status
```

## Build Process

### SQL VM
```bash
sudo ./sovereign build --sql
```
- Creates `vm/sql/rootfs.img` (Alpine + PostgreSQL)
- Creates `vm/sql/data.img` (4GB persistent storage)
- Prompts for database password if `.secrets` doesn't exist

### Vault VM
```bash
sudo ./sovereign build --vault
```
- Creates `vm/vault/rootfs.img` (Alpine + Vaultwarden + Tailscale)
- Uses shared kernel from SQL VM
- Requires SQL VM to be built first

### Forge VM
```bash
sudo ./sovereign build --forge
```
- Creates `vm/forge/rootfs.img` (Alpine + Forgejo + Tailscale)
- Uses shared kernel from SQL VM
- Requires SQL VM to be built first

## Deploy Process

```bash
./sovereign deploy --sql
./sovereign deploy --vault
./sovereign deploy --forge
```

Deploy pushes files to `/data/sovereign/vm/<name>/` on the device:
- `rootfs.img` - VM filesystem
- `data.img` - Persistent storage (preserved on redeploy)
- `Image` - Kernel
- `start.sh` - Legacy start script

Also deploys the daemon script to `/data/sovereign/sovereign_start.sh`.

## Start/Stop Process

### Start Order (IMPORTANT)
VMs must be started in order due to dependencies:

```bash
# 1. Start SQL first (database)
./sovereign start --sql
# Wait 15-20 seconds for PostgreSQL to initialize

# 2. Start Vault (depends on SQL for database)
./sovereign start --vault

# 3. Start Forge (depends on SQL for database)  
./sovereign start --forge
```

### Stop
```bash
./sovereign stop --vault
./sovereign stop --forge
./sovereign stop --sql
```

## Testing

```bash
./sovereign test --sql    # Tests PostgreSQL on 192.168.100.2:5432
./sovereign test --vault  # Tests Vaultwarden on 192.168.100.4:443
./sovereign test --forge  # Tests Forgejo on 192.168.100.3:3000
```

## Tailscale Cleanup

If Tailscale registrations become duplicated (sovereign-vault-1, -2, -3...):

```bash
./sovereign clean --vault  # Removes all sovereign-vault-* machines
./sovereign clean --sql    # Removes all sovereign-sql-* machines
./sovereign clean --forge  # Removes all sovereign-forge-* machines
```

Requires `TAILSCALE_API_KEY` in `.env`.

## Network Architecture

All VMs share a bridge network:
- Bridge: `vm_bridge` at 192.168.100.1
- SQL: 192.168.100.2 (port 5432)
- Forge: 192.168.100.3 (port 3000, 22)
- Vault: 192.168.100.4 (port 443)

## Troubleshooting

### VM Dies During Boot
Check console log:
```bash
adb shell "su -c 'cat /data/sovereign/vm/sql/console.log'"
```

### VMs Killed by Android
The daemon script prevents this, but verify it's running:
```bash
adb shell "su -c 'ps -ef | grep sovereign_start'"
```

### Tailscale State Not Persisting
TEAM_039 fixed this - state is now checked BEFORE tailscaled starts.
If still having issues, check `/data/tailscale/tailscaled.state` exists.

### Password Authentication Failures
TEAM_038 added ALTER USER to update passwords on each boot.
Check SQL console.log for "Created/updated vaultwarden user".

## Key Files

| File | Purpose |
|------|---------|
| `.env` | Environment variables (Tailscale keys, passwords) |
| `.secrets` | PostgreSQL master password |
| `vm/*/init.sh` | VM init scripts (copied to rootfs) |
| `host/sovereign_start.sh` | Daemon script for VM lifecycle |
| `internal/vm/*/` | Go code for each VM type |

## Common Operations

### Full Rebuild and Deploy
```bash
sudo ./sovereign build --sql --skip-preflight
sudo ./sovereign build --vault --skip-preflight
./sovereign deploy --sql
./sovereign deploy --vault
./sovereign start --sql
sleep 20
./sovereign start --vault
```

### Check All VMs Running
```bash
adb shell "su -c 'ps -ef | grep crosvm | grep -v grep'"
```

### Check Service Connectivity
```bash
adb shell "su -c 'nc -z -w 2 192.168.100.2 5432 && echo SQL_OK'"
adb shell "su -c 'nc -z -w 2 192.168.100.4 443 && echo VAULT_OK'"
```
