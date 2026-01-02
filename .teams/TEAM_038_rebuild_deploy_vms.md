# TEAM_038 — Rebuild and Deploy VMs

## Mission
Execute the rebuild, deploy, and start sequence for SQL and Vault VMs, then verify they survive > 90 seconds.

## Status: COMPLETED

## Bugs Found and Fixed

### Bug 1: Daemon trap kills ALL VMs instead of just the watched one
**File**: `host/sovereign_start.sh`

**Problem**: The global `trap stop_all TERM INT` was called even in `start_single()` mode, so stopping SQL would also kill Vault.

**Fix**: Override trap inside `start_single()` to only kill the specific VM being watched:
```bash
trap "log 'Stopping ${VM} VM (PID: ${VM_PID})'; kill ${VM_PID} 2>/dev/null; rm -f ${VM_DIR}/vm.pid; exit 0" TERM INT
```

### Bug 2: Vault VM not receiving database password
**File**: `host/sovereign_start.sh`

**Problem**: In `start_single()`, the vault case passed empty extra params, so Vaultwarden couldn't authenticate to PostgreSQL.

**Fix**: Pass `vaultwarden.db_password` to the Vault VM:
```bash
vault)
    VM_DIR="$VAULT_DIR"
    local VAULT_EXTRA=""
    [ -n "$POSTGRES_VAULTWARDEN_PASSWORD" ] && VAULT_EXTRA="$VAULT_EXTRA vaultwarden.db_password=$POSTGRES_VAULTWARDEN_PASSWORD"
    VM_PID=$(start_vm "$VAULT_DIR" "vm_vault" "$VAULT_EXTRA")
    ;;
```

## Verification Results

### VMs Survived > 90 Seconds ✓
- SQL VM: PID 27184 (started 01:46:00, verified running at 01:48:12)
- Vault VM: PID 27253 (started 01:46:09, verified running at 01:48:12)
- PostgreSQL reachable on 192.168.100.2:5432 ✓
- TAP interfaces UP ✓

## Files Modified

| File | Change |
|------|--------|
| `host/sovereign_start.sh` | Fixed trap in `start_single()` to only kill watched VM, added password for vault |

## Handoff Checklist
- [x] VMs built successfully
- [x] VMs deployed successfully  
- [x] Boot script deployed to device
- [x] VMs survive > 90 seconds
- [x] PostgreSQL reachable
- [x] Bugs documented and fixed

## Additional Bugs Found and Fixed (Session 2)

### Bug 3: Vault init.sh reads /proc/cmdline BEFORE mounting /proc
**File**: `vm/vault/init.sh`

**Problem**: The password parsing happened at lines 31-36, but `/proc` wasn't mounted until line 43. Result: `VAULTWARDEN_DB_PASS` was always empty, falling back to default "vaultwarden" password.

**Fix**: Moved `mount -t proc` BEFORE the cmdline parsing block.

### Bug 4: Tailscale state detection used wrong format check  
**File**: `vm/vault/init.sh`

**Problem**: The state check used `grep 'PrivateNodeKey'` but Tailscale state files are binary/protobuf format, not plaintext. This caused every restart to re-register with Tailscale, incrementing the hostname (sovereign-vault-1, -2, -3...).

**Fix**: Changed to simply check if state file exists and is non-empty. Let tailscaled validate the state itself.

### Bug 5: SQL init.sh had same /proc mount ordering issue
**File**: `vm/sql/init.sh` (already fixed by TEAM_035)

The SQL init.sh correctly mounts /proc before reading cmdline. The Vault init.sh was copied but the order was wrong.

### Bug 6: Tailscale up command blocks indefinitely
**File**: `vm/vault/init.sh`

**Problem**: `tailscale up` command would hang forever if Tailscale couldn't connect, blocking the entire boot process.

**Fix**: Added `timeout 30` to tailscale up commands so boot continues even if Tailscale is slow.

## Current Status

- **SQL VM**: Working, PostgreSQL accessible on 192.168.100.2:5432 ✓
- **Vault VM**: Running on HTTPS:443, Rocket launched ✓
- **Tailscale FQDN**: May be empty if Tailscale times out, but service still works via direct IP
- **Note**: VMs must be started in order: SQL first, wait 15-20s, then Vault

## Commands for Future Teams

```bash
# Kill all and restart clean
adb shell "su -c 'pkill -9 sovereign; pkill -9 crosvm'"

# Start SQL (must be first - others depend on it)
adb shell "su -c 'nohup /data/sovereign/sovereign_start.sh start sql > /data/sovereign/daemon_sql.log 2>&1 &'"

# Wait for SQL to be ready
sleep 15

# Start Vault
adb shell "su -c 'nohup /data/sovereign/sovereign_start.sh start vault > /data/sovereign/daemon_vault.log 2>&1 &'"

# Verify running
adb shell "su -c 'ps -ef | grep crosvm | grep -v grep'"

# Check services
adb shell "su -c 'nc -z -w 2 192.168.100.2 5432 && echo SQL_OK || echo SQL_FAIL'"
adb shell "su -c 'nc -z -w 2 192.168.100.4 443 && echo VAULT_OK || echo VAULT_FAIL'"
```

## Files Modified This Session

| File | Change |
|------|--------|
| `host/sovereign_start.sh` | Fixed trap in start_single(), added vault password parameter |
| `vm/sql/init.sh` | Added ALTER USER to fix password if role exists |
| `vm/vault/init.sh` | Moved /proc mount before cmdline parsing, fixed Tailscale state detection, added timeout to tailscale up |
