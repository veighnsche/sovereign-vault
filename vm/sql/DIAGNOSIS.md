# SQL VM Diagnosis & Verification Guide

**TEAM_017: Complete diagnosis guide for future teams**

---

## Quick Status Check

Run these commands to diagnose issues:

```bash
# 1. Is crosvm running?
adb shell "su -c 'pgrep -f crosvm.*sql'"

# 2. Is TAP interface up?
adb shell "su -c 'ip link show vm_sql'"

# 3. Is policy routing fix applied?
adb shell "su -c 'ip rule list'" | grep "pref 1"
# Should show: "1: from all lookup main"

# 4. Can host ping VM?
adb shell "su -c 'ping -c 1 -I vm_sql 192.168.100.2'"

# 5. Check VM init.log
adb shell "su -c 'cat /data/sovereign/vm/sql/console.log'" | tail -50

# 6. Check Tailscale from your machine
tailscale status | grep sovereign-sql
```

---

## Layer-by-Layer Diagnosis

### Layer 1: VM Process

**Check**: Is crosvm running?
```bash
adb shell "su -c 'pgrep -f crosvm.*sql'"
```

**If NOT running**:
- Check console.log for errors
- Verify Image and rootfs.img exist
- Check start.sh syntax

**If running but problems**:
- Continue to Layer 2

---

### Layer 2: TAP Interface (Host Side)

**Check**: TAP interface exists and is UP?
```bash
adb shell "su -c 'ip link show vm_sql'"
```

**Expected output**:
```
vm_sql: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
    inet 192.168.100.1/24 scope global vm_sql
```

**If missing or DOWN**:
- start.sh didn't complete
- Check if `ip tuntap add` succeeded

---

### Layer 3: Policy Routing Fix (CRITICAL)

**Check**: Is the Android policy routing bypass in place?
```bash
adb shell "su -c 'ip rule list'" | head -5
```

**MUST show**:
```
0:      from all lookup local
1:      from all lookup main    <-- THIS IS THE KEY FIX
```

**If priority 1 rule is missing**:
```bash
adb shell "su -c 'ip rule add from all lookup main pref 1'"
```

**WHY THIS MATTERS**: Android's netd uses complex policy routing with fwmarks. Without this rule, NAT return traffic never reaches the VM.

---

### Layer 4: NAT and Forwarding

**Check**: NAT rule exists?
```bash
adb shell "su -c 'iptables -t nat -L POSTROUTING -n'" | grep 192.168.100
```

**Expected**:
```
MASQUERADE  all  --  192.168.100.0/24  0.0.0.0/0
```

**Check**: FORWARD rules?
```bash
adb shell "su -c 'iptables -L FORWARD -n -v'" | head -5
```

**Expected**: Rules for vm_sql <-> wlan0

---

### Layer 5: Guest Network

**Check**: Pull and examine init.log
```bash
adb pull /data/sovereign/vm/sql/rootfs.img /tmp/check.img
sudo mount -o loop /tmp/check.img /tmp/vm_mnt
cat /tmp/vm_mnt/init.log
sudo umount /tmp/vm_mnt
```

**Look for**:
- "Found interface: eth0" (or similar)
- "PING ... 3 packets transmitted, 3 received"
- Tailscale output

**Common failures**:
- "Ping failed" → Layer 3 issue (policy routing)
- "No network interface found" → Kernel missing virtio-net
- TLS certificate errors → System time not set

---

### Layer 6: System Time (TLS)

**CRITICAL**: VM clock starts at 1970. TLS fails without correct time.

**Check in init.log**:
```
date -s "2025-12-28 22:00:00"
```

**If you see**:
```
x509: certificate has expired or is not yet valid
```
→ Time was not set properly

---

### Layer 7: Tailscale

**Check**: Is Tailscale daemon running in VM?
```
# In init.log, look for:
Starting Tailscale...
tailscaled started
tailscale up --authkey=...
```

**Check**: Is it connected?
```bash
tailscale status | grep sovereign-sql
```

**If not connected**:
- Check TAILSCALE_AUTHKEY in /data/sovereign/.env
- Check init.log for Tailscale errors
- Verify Layer 5 (ping 8.8.8.8 works)

---

### Layer 8: PostgreSQL

**Check in init.log**:
```
Starting PostgreSQL...
waiting for server to start.... done
server started
```

**Common failures**:

1. **"could not open shared memory segment"**
   - /dev/shm not mounted as tmpfs
   - Fix: Add `mount -t tmpfs -o mode=1777 tmpfs /dev/shm`

2. **"dynamic_shared_memory_type" errors**
   - Need `dynamic_shared_memory_type = mmap` in postgresql.conf
   - POSIX shared memory doesn't work in AVF

3. **Permission denied on log file**
   - /var/log not writable
   - Fix: `mkdir -p /var/log; touch /var/log/postgresql.log; chown postgres:postgres ...`

---

## Complete Test Sequence

```bash
# From your development machine:

# 1. Rebuild everything
cd sovereign
./sovereign build --sql
./sovereign deploy --sql  
./sovereign start --sql

# 2. Wait 90 seconds for boot + Tailscale
sleep 90

# 3. Check Tailscale
tailscale status | grep sovereign-sql
# Expected: 100.x.x.x    sovereign-sql  ...

# 4. Test PostgreSQL
PGPASSWORD=<your-password> psql -h sovereign-sql -U postgres -c "SELECT version();"
# Expected: PostgreSQL 15.x on aarch64-alpine-linux-musl
```

---

## Credentials

| Setting | Location |
|---------|----------|
| Username | `postgres` (always) |
| Password | `sovereign/.secrets` file, `DB_PASSWORD=` line |

**To check**:
```bash
cat sovereign/.secrets | grep DB_PASSWORD
```

---

## Known Gotchas (TEAM_017)

### 1. Android Policy Routing
- **Problem**: Android's netd blocks return traffic
- **Fix**: `ip rule add from all lookup main pref 1`
- **Source**: https://github.com/bvucode/crosvm-on-android

### 2. VM Clock at 1970
- **Problem**: TLS certificate validation fails
- **Fix**: Set time in simple_init before Tailscale starts

### 3. PostgreSQL Shared Memory
- **Problem**: POSIX shm_open fails in AVF
- **Fix**: Use `dynamic_shared_memory_type = mmap`

### 4. Network Interface Name
- **Problem**: May be eth0, enp0s1, or other
- **Fix**: Detect dynamically: `ls /sys/class/net/ | grep -v lo | head -1`

### 5. /dev/shm Not Mounted
- **Problem**: PostgreSQL needs shared memory
- **Fix**: `mount -t tmpfs -o mode=1777 tmpfs /dev/shm`

---

## Idempotency Requirements

For `sovereign` CLI to be idempotent, each command must:

### `sovereign build --sql`
- [ ] Skip Docker build if image exists and Dockerfile unchanged
- [ ] Skip rootfs export if rootfs.img exists and image unchanged
- [ ] Always re-run rootfs.PrepareForAVF (it's already idempotent)

### `sovereign deploy --sql`
- [ ] Delete old files before pushing new ones
- [ ] Or use checksums to skip unchanged files

### `sovereign start --sql`
- [ ] Kill existing VM before starting new one
- [ ] Clean up old TAP interface
- [ ] Clean up old ip rules
- [ ] Clean up old iptables rules

### `sovereign stop --sql`
- [ ] Kill crosvm process
- [ ] Remove TAP interface
- [ ] (Optionally) Remove ip rules and iptables rules

---

## Files Reference

| File | Purpose |
|------|---------|
| `vm/sql/Image` | ARM64 kernel (RAW format) |
| `vm/sql/rootfs.img` | Alpine Linux rootfs |
| `vm/sql/start.sh` | VM launch script (pushed to device) |
| `vm/sql/Dockerfile` | Docker build for rootfs |
| `internal/rootfs/rootfs.go` | Generates simple_init script |
| `internal/vm/sql/sql.go` | CLI commands implementation |
| `.secrets` | DB credentials (not in git) |
| `.env` | TAILSCALE_AUTHKEY (not in git) |

---

## Emergency Recovery

If everything is broken:

```bash
# 1. Stop everything
adb shell "su -c 'pkill -9 crosvm; ip link del vm_sql 2>/dev/null'"

# 2. Remove from device
./sovereign remove --sql

# 3. Clean local build artifacts
rm -f vm/sql/rootfs.img vm/sql/data.img

# 4. Rebuild from scratch
./sovereign build --sql
./sovereign deploy --sql
./sovereign start --sql
```

---

*Last updated: 2024-12-28 by TEAM_017*
