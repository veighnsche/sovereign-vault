# Tailscale Port Exposure in AVF VMs

## Status: ✅ IMPLEMENTED (TEAM_034)

**TEAM_033 Discovery (2025-12-29)**: `tailscale serve` is NOT needed. Services just need to listen on any routable interface.

**TEAM_034 Implementation (2025-12-30)**: Removed `tailscale serve` from all init scripts. Direct port binding now in use.

### The Working Solution

```bash
# From external Tailscale device (verified working):
nc -zv sovereign-sql.tail5bea38.ts.net 5432
# Connection to sovereign-sql.tail5bea38.ts.net (100.119.99.33) 5432 port [tcp/postgres] succeeded!
```

**Why it works**: PostgreSQL listens on `192.168.100.2:5432`. Tailscale routes incoming traffic to the VM's `tailscale0` interface. The service responds directly - no proxy involved, no fwmark issue.

This document captures extensive investigation into exposing VM services via Tailscale in Android Virtualization Framework (AVF) VMs.

---

## The Problem

Services running inside crosvm guest VMs (PostgreSQL, Forgejo) cannot be reached via Tailscale, despite:
- Tailscale connecting successfully inside the VM
- Services running and listening on correct ports
- `tailscale serve` configured correctly

**Symptom**: `ERR_CONNECTION_RESET` or `i/o timeout` when accessing `http://<tailscale-ip>:<port>`

---

## Investigation Summary

### Attempt 1: Userspace Networking (Default)

```bash
tailscaled --tun=userspace-networking &
tailscale serve --bg --tcp 3000 tcp://127.0.0.1:3000
```

**Result**: `failed to TCP proxy port 3000 to 127.0.0.1:3000: dial tcp 127.0.0.1:3000: i/o timeout`

**Root Cause**: Userspace networking creates an isolated network namespace. Tailscale's network stack cannot reach the VM's localhost or any local interface.

### Attempt 2: Userspace + TAP IP

```bash
tailscaled --tun=userspace-networking &
tailscale serve --bg --tcp 3000 tcp://192.168.100.3:3000
```

**Result**: Same timeout - userspace networking isolation is complete.

### Attempt 3: Native Tun Mode (Kernel Fix)

Added full nftables support to guest kernel:
```bash
# build-guest-kernel.sh additions:
--enable NF_TABLES
--enable NF_TABLES_INET
--enable NF_TABLES_IPV4
--enable NFT_CT
--enable NFT_NAT
--enable NFT_MASQ
--enable NFT_CHAIN_NAT
--enable NFT_CHAIN_ROUTE
--enable IP_ADVANCED_ROUTER
--enable IP_MULTIPLE_TABLES
--enable IP_ROUTE_FWMARK
--enable FIB_RULES
```

Then:
```bash
tailscaled &  # No --tun flag = native tun mode
tailscale serve --bg --tcp 3000 tcp://127.0.0.1:3000
```

**Result**: Same timeout.

**Root Cause**: Native tun mode uses fwmark-based policy routing. All connections from tailscaled (including to localhost) get marked and routed through Tailscale's routing rules, which can't reach local interfaces.

### Attempt 4: Native Tun + TAP IP

```bash
tailscaled &
tailscale serve --bg --tcp 3000 tcp://192.168.100.3:3000
```

**Result**: Same timeout - fwmark routing affects TAP interface connections too.

---

## Why This Happens

Tailscale's architecture:

1. **Userspace Networking**: Creates isolated netstack - cannot reach any VM interfaces
2. **Native Tun Mode**: Uses `fwmark` to mark packets and policy routing (`ip rule`) to route them
   - All outbound connections from tailscaled get marked
   - Marked packets follow Tailscale's routing tables
   - These tables route to the Tailscale interface, not local interfaces

The `tailscale serve` proxy runs inside tailscaled, so its connections inherit the fwmark routing.

---

## What Doesn't Work

| Approach | Why It Fails |
|----------|--------------|
| `tailscale serve` to localhost | Userspace: isolated namespace. Native: fwmark routing |
| `tailscale serve` to TAP IP | Same as above |
| Android host iptables DNAT | Android Tailscale is VPN app, not in iptables path |

---

## Solution Implemented (TEAM_034)

### Direct Port Binding (Primary)

Services listen on `0.0.0.0` which binds to all interfaces including tailscale0. Inbound Tailscale traffic arrives directly - no proxy needed.

**TEAM_034 removed `tailscale serve` from:**
- `vm/sql/init.sh` (lines 264, 293)
- `vm/forgejo/init.sh` (lines 197-198, 296-297)

### Subnet Router in SQL VM (Backup)

The SQL VM also advertises the VM subnet (192.168.100.0/24) as a Tailscale subnet router:

```bash
# In vm/sql/init.sh
tailscale up --hostname=sovereign-sql --advertise-routes=192.168.100.0/24 --accept-routes
```

**How it works:**
1. SQL VM connects to Tailscale and advertises the 192.168.100.0/24 subnet
2. Tailscale admin must approve the subnet routes in the admin panel
3. External Tailscale devices can then access VMs directly at their TAP IPs:
   - PostgreSQL: `192.168.100.2:5432`
   - Forgejo: `192.168.100.3:3000`

**Requirements:**
1. Approve subnet routes at https://login.tailscale.com/admin/machines
2. Enable "Subnet routes" for the sovereign-sql machine

**Why this works:**
- Subnet routing happens at the Tailscale network level, not inside tailscaled
- Traffic from external Tailscale devices is routed to the VM's TAP interface
- No fwmark isolation issues because traffic arrives from the Tailscale interface, not from tailscaled itself

---

## Previous Attempts (For Reference)

### 1. Reverse Proxy on Android Host

Run a TCP proxy (socat, nginx) on the Android host that:
- Listens on a port accessible to Android's Tailscale
- Forwards to VM's TAP IP (192.168.100.x)

Challenge: Android doesn't have socat/nginx by default.

### 2. Tailscale Subnet Router on Android Host

Advertise VM subnet (192.168.100.0/24) from Android host.

Challenge: Requires Android Tailscale CLI which isn't available by default.

### 3. Direct Port Exposure via Bridge

Use Linux bridge to connect Android's Tailscale interface directly to VM network.

Challenge: Android's Tailscale is a VPN app, not a network interface.

### 4. Different Tailscale Architecture

Run Tailscale on Android host only (not in VM). Use Android host as reverse proxy.

---

## Kernel Configs Reference

For future attempts requiring native tun, these kernel configs are needed:

```bash
# Netfilter core
CONFIG_NETFILTER=y
CONFIG_NF_CONNTRACK=y
CONFIG_NETFILTER_XTABLES=y

# For fwmark support
CONFIG_NETFILTER_XT_MARK=y
CONFIG_NETFILTER_XT_CONNMARK=y

# nftables (preferred by modern Tailscale)
CONFIG_NF_TABLES=y
CONFIG_NF_TABLES_INET=y
CONFIG_NF_TABLES_IPV4=y
CONFIG_NFT_CT=y
CONFIG_NFT_NAT=y
CONFIG_NFT_MASQ=y
CONFIG_NFT_CHAIN_NAT=y
CONFIG_NFT_CHAIN_ROUTE=y

# iptables fallback
CONFIG_IP_NF_IPTABLES=y
CONFIG_IP_NF_FILTER=y
CONFIG_IP_NF_NAT=y
CONFIG_IP_NF_TARGET_MASQUERADE=y

# Policy routing
CONFIG_IP_ADVANCED_ROUTER=y
CONFIG_IP_MULTIPLE_TABLES=y
CONFIG_IP_ROUTE_FWMARK=y
CONFIG_FIB_RULES=y
```

---

## Current Status (WORKING)

VMs are accessible via Tailscale DNS names:
- **PostgreSQL**: `sovereign-sql.tail5bea38.ts.net:5432` ✅ VERIFIED
- **Forgejo**: `sovereign-forge.tail5bea38.ts.net:3000` (pending setup)
- **Vaultwarden**: `sovereign-vault.tail5bea38.ts.net:8080` (pending setup)

**Key insight**: `tailscale serve` was a red herring. Direct port binding works fine.

---

## Files Modified

- `vm/build-guest-kernel.sh` - Shared kernel with nftables configs
- `vm/sql/init.sh` - Tailscale startup, tailscale serve placement
- `vm/forgejo/init.sh` - Same

---

## Team Notes

- TEAM_030 spent significant effort investigating `tailscale serve` - it doesn't work
- TEAM_033 discovered that direct port binding works - no `tailscale serve` needed
- **TEAM_034 implemented the fix** - removed `tailscale serve` from all init scripts
- The subnet router approach also works for accessing VMs via TAP IPs
- **Don't use `tailscale serve`** - just have services listen on a routable interface

### What You Need For Forgejo/Vaultwarden

Just configure the service to listen on `0.0.0.0` or a specific IP:

```ini
# Forgejo app.ini
[server]
HTTP_ADDR = 0.0.0.0
HTTP_PORT = 3000
```

Then access via `sovereign-forge.tail5bea38.ts.net:3000`
