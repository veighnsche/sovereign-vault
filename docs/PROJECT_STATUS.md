# Sovereign Project Status

**TEAM_034** | Last Updated: 2025-12-30

---

## 🎯 Executive Summary

**The project is ~80% complete.** All core infrastructure is working. The remaining 20% is:

1. Simplify Tailscale architecture (remove from VMs, use host subnet router)
2. Complete Forgejo integration
3. Add Vaultwarden VM

---

## ✅ What's DONE and WORKING

| Component | Status | Notes |
|-----------|--------|-------|
| **Host Kernel** | ✅ DONE | KernelSU + pKVM enabled |
| **Guest Kernel** | ✅ DONE | Custom Alpine kernel with required configs |
| **PostgreSQL VM** | ✅ DONE | Boots, runs PostgreSQL 15.15, persists data |
| **Bridge Networking** | ✅ DONE | VMs can talk to each other on 192.168.100.0/24 |
| **Internet from VMs** | ✅ DONE | NAT via iptables works |
| **Tailscale Connection** | ✅ DONE | VMs connect and appear in admin panel |
| **Sovereign CLI** | ✅ DONE | `sovereign build/deploy/start/stop/test` |
| **Forgejo VM** | 🟡 Partial | Boots, needs service integration |

---

## ✅ TAILSCALE EXTERNAL ACCESS - SOLVED!

### TEAM_033 Discovery (2025-12-29)

**`tailscale serve` is NOT needed!** Direct port binding works.

```bash
# Verified working from external Tailscale device:
nc -zv sovereign-sql.tail5bea38.ts.net 5432
# Connection succeeded!
```

**Why it works**: Services listen on any interface (0.0.0.0 or TAP IP). Tailscale routes incoming traffic directly to the VM. No proxy, no fwmark issue.

### Access Pattern

```
External Device → Tailscale Network → VM's tailscale0 → Service

sovereign-sql.tail5bea38.ts.net:5432   → PostgreSQL ✅ VERIFIED
sovereign-forge.tail5bea38.ts.net:3000 → Forgejo (pending)
sovereign-vault.tail5bea38.ts.net:8080 → Vaultwarden (pending)
```

### What This Means

- **Keep Tailscale in VMs** - each VM gets its own DNS name
- **Remove `tailscale serve`** - it's not needed and causes confusion
- **Configure services to listen on 0.0.0.0** - standard practice

---

## 📋 Remaining Tasks

### Task 1: Remove `tailscale serve` from Init Scripts (Priority: HIGH)

Keep Tailscale in VMs (for DNS names), but remove the unnecessary `tailscale serve` commands.

**Files to modify:**
- `vm/sql/init.sh` - Remove `tailscale serve` lines
- `vm/forgejo/init.sh` - Remove `tailscale serve` lines

**Result:** Simpler init scripts. Services accessed directly via Tailscale DNS names.

### Task 2: Verify Forgejo Access (Priority: HIGH)

1. Ensure Forgejo listens on `0.0.0.0:3000`
2. Test: `nc -zv sovereign-forge.tail5bea38.ts.net 3000`
3. Access web UI in browser

### Task 3: Complete Forgejo VM (Priority: MEDIUM)

- Verify Forgejo service starts and is accessible at 192.168.100.3:3000
- Configure Forgejo to use PostgreSQL at 192.168.100.2:5432
- Test git clone/push via HTTP

### Task 4: Add Vaultwarden VM (Priority: MEDIUM)

- Create `vm/vaultwarden/` following sql/forgejo patterns
- Use IP 192.168.100.4
- Configure to use PostgreSQL backend
- Test web UI access

---

## 🔧 Architecture Decisions Made

### Decision 1: Keep Tailscale in VMs, Remove `tailscale serve`

**Why:** Each VM gets its own Tailscale DNS name (e.g., `sovereign-sql.tail5bea38.ts.net`). Direct port binding works - services listen on 0.0.0.0 and Tailscale routes inbound traffic directly. `tailscale serve` was a red herring.

**Implementation (TEAM_034):** Removed `tailscale serve` from init.sh. Services accessible via Tailscale DNS names directly.

### Decision 2: Keep Current Approach (Not Android App)

**Why:** We're 80% done. Android app approach would require weeks of new development for the same end result.

**When to reconsider:** If Android's netd keeps clearing our routing rules, causing stability issues.

### Decision 3: Shared Bridge, Not Separate Subnets

**Why:** All VMs on 192.168.100.0/24 simplifies routing and allows direct VM-to-VM communication.

---

## 🧪 Verification Commands

```bash
# Build everything
./sovereign build --sql --forge

# Deploy to device
./sovereign deploy --sql --forge

# Start VMs
./sovereign start --sql
./sovereign start --forge

# Run tests
./sovereign test --sql

# Check VM is reachable from host
adb shell ping -c 3 192.168.100.2

# Check PostgreSQL from host
adb shell nc -zv 192.168.100.2 5432

# Check from another Tailscale device (after subnet router configured)
psql -h 192.168.100.2 -U postgres -c "SELECT 1"
```

---

## 📁 Key Files

| File | Purpose |
|------|---------|
| `sovereign_vault.md` | Master architecture document |
| `vm/sql/start.sh` | Host-side VM startup script |
| `vm/sql/init.sh` | Guest-side initialization |
| `vm/sql/Dockerfile` | Rootfs build definition |
| `docs/AVF_VM_NETWORKING.md` | Networking architecture |
| `docs/TAILSCALE_AVF_LIMITATIONS.md` | Why Tailscale in VMs doesn't work |

---

## ⚠️ Gotchas for Future Teams

### TEAM_033: Networking Rule Persistence

Android's netd may clear custom `ip rule` entries. If VMs lose connectivity after a while:
```bash
ip rule add from all lookup main pref 1
```
This bypasses Android's fwmark routing for return traffic.

### TEAM_033: TAP_HOST_IP Undefined

There was a bug in start.sh using undefined `TAP_HOST_IP`. Fixed to use `BRIDGE_NAME` instead.

### TEAM_030: Tailscale fwmark

Don't waste time trying to make `tailscale serve` work from inside VMs. It's a fundamental design limitation. Use subnet routing instead.

### TEAM_023: Duplicate Tailscale Registrations

Fixed by using persistent state on data.img. Don't regenerate Tailscale state on every boot.

---

## 🚀 Path to Completion

```
Week 1:
├── Day 1-2: Remove Tailscale from VM init scripts
├── Day 3: Configure Android Tailscale subnet router
└── Day 4-5: Test end-to-end access

Week 2:
├── Day 1-2: Complete Forgejo integration
├── Day 3-4: Add Vaultwarden VM
└── Day 5: Full system test
```

**Estimated time to fully working system: 1-2 weeks**
