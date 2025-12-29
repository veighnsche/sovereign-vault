# TEAM_030: Tailscale Native Tun Investigation

## Status: BLOCKED - Architectural Limitation

## Task
Fix Forgejo connectivity via Tailscale by enabling native tun mode in guest kernel.

## Investigation Timeline

### Phase 1: Userspace Networking Analysis
- Identified that `--tun=userspace-networking` creates isolated network namespace
- `tailscale serve` cannot reach localhost OR TAP IP from isolated namespace
- Error: `dial tcp 127.0.0.1:3000: i/o timeout`

### Phase 2: Kernel Enhancement
Added nftables support to `build-guest-kernel.sh`:
- `NF_TABLES`, `NF_TABLES_INET`, `NF_TABLES_IPV4`
- `NFT_CT`, `NFT_NAT`, `NFT_MASQ`, `NFT_CHAIN_NAT`
- `IP_MULTIPLE_TABLES`, `IP_ROUTE_FWMARK`, `FIB_RULES`

Result: Tailscale runs in native nftables mode ✓

### Phase 3: Port Conflict Fix
Discovered `tailscale serve` binds port before PostgreSQL starts.
Fix: Moved `tailscale serve` to after PostgreSQL startup.

Result: PostgreSQL starts successfully ✓

### Phase 4: Native Tun Testing
Native tun mode uses fwmark-based policy routing.
All connections from tailscaled (including to local IPs) get marked and routed through Tailscale routing tables.

Result: Same timeout - `dial tcp 192.168.100.3:3000: i/o timeout`

## Root Cause

**Tailscale's fwmark routing marks ALL outbound connections from tailscaled, including those to local interfaces. These marked packets follow Tailscale's routing tables which cannot reach local interfaces.**

This is a fundamental architectural limitation, not a configuration issue.

## Files Modified
- `vm/sql/build-guest-kernel.sh` - Added 20+ kernel configs
- `vm/sql/init.sh` - Native tun, tailscale serve ordering
- `vm/forgejo/init.sh` - Native tun, TAP IP proxy
- `docs/TAILSCALE_AVF_LIMITATIONS.md` - Detailed documentation

## What Works
- VM-to-VM communication via bridge (192.168.100.0/24) ✓
- Forgejo connecting to PostgreSQL ✓
- Tailscale connecting from inside VMs ✓
- VMs appearing in Tailscale admin panel ✓

## What Doesn't Work
- External access to VM services via Tailscale ✗

## Recommendations for Future Teams
1. Consider running Tailscale only on Android host, not in VMs
2. Use Android host as reverse proxy to VMs
3. Explore Tailscale subnet routing from Android host
4. Consider alternative overlay networks (WireGuard direct, Nebula)

## Handoff Checklist
- [x] Project builds
- [x] PostgreSQL running
- [x] Forgejo running
- [x] VM-to-VM networking working
- [x] Documentation created
- [ ] External Tailscale access (BLOCKED)
