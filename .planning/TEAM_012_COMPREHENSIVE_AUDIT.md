# TEAM_012 Comprehensive Audit

**Date:** 2025-12-28
**Status:** ANALYSIS COMPLETE

---

## 1. KERNEL CONFIG AUDIT

### 🚨 CRITICAL: sovereign_guest.fragment is MISSING key configs!

The fragment at `/private/devices/google/raviole/sovereign_guest.fragment` is **MISSING**:

| Config | Needed For | Status in Fragment |
|--------|------------|-------------------|
| `CONFIG_SYSVIPC=y` | PostgreSQL shmget() | **❌ MISSING** |
| `CONFIG_NETFILTER=y` | Tailscale iptables | **❌ MISSING** |
| `CONFIG_IP_NF_IPTABLES=y` | iptables userspace | **❌ MISSING** |
| `CONFIG_NF_CONNTRACK=y` | Connection tracking | **❌ MISSING** |

### Required Additions to `sovereign_guest.fragment`:

```kconfig
# -----------------------------------------------------------------------------
# 13. System V IPC (REQUIRED for PostgreSQL shared memory)
# -----------------------------------------------------------------------------
CONFIG_SYSVIPC=y
CONFIG_SYSVIPC_SYSCTL=y

# -----------------------------------------------------------------------------
# 14. Netfilter/iptables (REQUIRED for Tailscale)
# -----------------------------------------------------------------------------
CONFIG_NETFILTER=y
CONFIG_NETFILTER_ADVANCED=y
CONFIG_NF_CONNTRACK=y
CONFIG_NF_TABLES=y
CONFIG_NF_TABLES_INET=y
CONFIG_NFT_CT=y
CONFIG_NFT_NAT=y
CONFIG_NFT_MASQ=y

# IPv4 netfilter
CONFIG_IP_NF_IPTABLES=y
CONFIG_IP_NF_FILTER=y
CONFIG_IP_NF_NAT=y
CONFIG_IP_NF_TARGET_MASQUERADE=y

# IPv6 netfilter
CONFIG_IP6_NF_IPTABLES=y
CONFIG_IP6_NF_FILTER=y
CONFIG_IP6_NF_NAT=y
CONFIG_IP6_NF_TARGET_MASQUERADE=y
```

### Also in `vm/sql/kernel-config`:
This file has `CONFIG_NETFILTER=y` but is missing `CONFIG_SYSVIPC=y` and all the iptables submenu items.

---

## 2. POSTGRESQL CREDENTIALS

### Current State:
Found in `vm/sql/scripts/init.sh` line 74:
```sh
su postgres -c "psql -c \"ALTER USER postgres PASSWORD 'sovereign';\""
```

Found in `sql.go` line 317 (test command):
```go
cmd.Env = append(os.Environ(), "PGPASSWORD=sovereign")
```

### Summary:
| Component | User | Password | Location |
|-----------|------|----------|----------|
| PostgreSQL | `postgres` | `sovereign` | `init.sh:74`, `sql.go:317` |

### 🚨 Problem: HARDCODED CREDENTIALS
- Password is hardcoded in shell script AND Go code
- No way to configure via environment variable or .env file
- Future teams should:
  1. Generate random password at build time
  2. Store in /data/postgres/.pgpass or similar
  3. Pass to CLI via .env file

---

## 3. WORKAROUNDS AUDIT

### All Workarounds Implemented by TEAM_012:

| # | Workaround | Why Needed | Can Be Eliminated? |
|---|------------|------------|-------------------|
| 1 | **Bypass OpenRC** with `/sbin/simple_init` | OpenRC hangs during sysinit | Yes, if OpenRC bug is fixed |
| 2 | **dhclient wrapper** in `/usr/bin/dhclient` | gvforwarder calls dhclient but Alpine uses udhcpc | Yes, if gvforwarder adds -no-dhcp flag or uses udhcpc |
| 3 | **Set PATH in init** | PATH not set when running as PID 1 | No, this is correct behavior |
| 4 | **console=hvc0** instead of ttyS0 | crosvm uses virtio console | No, this is the correct setting |
| 5 | **Socket cleanup** before restart | Stale sockets cause bind errors | Partially - could add to start.sh properly |
| 6 | **Tailscale userspace mode** | Kernel lacks netfilter | Yes, once kernel has CONFIG_NETFILTER |

### Contradiction Analysis:

**Question: Do any workarounds contradict each other?**

| Workaround A | Workaround B | Contradiction? |
|--------------|--------------|----------------|
| OpenRC bypass | dhclient wrapper | **NO** - independent issues |
| Simple init | PATH setting | **NO** - complementary (simple_init needs PATH) |
| Tailscale userspace | Missing netfilter | **YES** - userspace mode is a workaround for missing netfilter. Once kernel has netfilter, can use kernel mode |

**Key insight:** The Tailscale userspace networking workaround becomes UNNECESSARY once the kernel is rebuilt with `CONFIG_NETFILTER=y`. This is NOT a contradiction but a temporary fix.

### Workarounds That Can Be Eliminated:

1. **Once kernel is rebuilt with proper configs:**
   - Tailscale userspace mode → can use kernel mode
   - PostgreSQL will work (SYSVIPC present)

2. **If gvforwarder is fixed/updated:**
   - dhclient wrapper → not needed if gvforwarder uses udhcpc or has -no-dhcp

3. **If OpenRC issue is debugged:**
   - simple_init bypass → can use proper OpenRC

---

## 4. SOVEREIGN CLI INTEGRATION STATUS

### What's IN the Go module:

| Function | File | Status |
|----------|------|--------|
| Docker build | `sql.go:Build()` | ✅ Works |
| rootfs export | `docker.ExportImage()` | ✅ Works |
| rootfs AVF prep | `rootfs.PrepareForAVF()` | ⚠️ Incomplete |
| Deploy to device | `sql.go:Deploy()` | ✅ Works |
| Start VM | `sql.go:Start()` | ✅ Works (calls start.sh) |
| Stop VM | `sql.go:Stop()` | ✅ Works |
| Test VM | `sql.go:Test()` | ✅ Works |

### What's NOT in the Go module (manual workarounds):

| Missing | Current Workaround | Should Be In CLI |
|---------|-------------------|------------------|
| Kernel build | Manual cross-compile | Yes - `sovereign build --kernel` |
| simple_init creation | Manual mount + copy | Yes - in rootfs.PrepareForAVF() |
| dhclient wrapper | Manual mount + copy | Yes - in rootfs.PrepareForAVF() |
| PATH in init | In simple_init script | N/A (part of simple_init) |
| Socket cleanup | In start.sh | Already there |

### Missing from `rootfs.PrepareForAVF()`:

Looking at what I implemented manually that should be in the CLI:

1. **Create `/sbin/simple_init`** - bypasses OpenRC
2. **Create `/usr/bin/dhclient`** - wrapper for udhcpc
3. **Set execute permissions correctly**

---

## 5. RECOMMENDED ACTIONS

### Immediate (before next kernel build):

1. **Add to `sovereign_guest.fragment`:**
   ```kconfig
   CONFIG_SYSVIPC=y
   CONFIG_NETFILTER=y
   CONFIG_IP_NF_IPTABLES=y
   CONFIG_NF_CONNTRACK=y
   # ... (full list above)
   ```

2. **Update `rootfs.PrepareForAVF()` in Go:**
   - Add simple_init creation
   - Add dhclient wrapper creation
   - Ensure PATH is set in init script

### Medium-term:

3. **Add `sovereign build --kernel` command:**
   - Uses sovereign_guest.fragment
   - Cross-compiles for ARM64
   - Outputs RAW Image format

4. **Fix PostgreSQL credentials:**
   - Generate random password
   - Store in secure location
   - Pass via .env file

### Long-term:

5. **Debug OpenRC hang** - eliminate simple_init workaround
6. **Request gvforwarder -no-dhcp flag** - eliminate dhclient wrapper

---

## 6. FILES THAT NEED UPDATES

| File | Change Needed |
|------|---------------|
| `private/devices/google/raviole/sovereign_guest.fragment` | Add SYSVIPC, NETFILTER configs |
| `vm/sql/kernel-config` | Add SYSVIPC, expand NETFILTER |
| `internal/rootfs/rootfs.go` | Add simple_init, dhclient wrapper creation |
| `vm/sql/scripts/init.sh` | Consider replacing with simple_init permanently |
| `.env.example` | Add POSTGRES_PASSWORD variable |

