# Phase 2: Design

**Feature:** Forgejo VM Implementation  
**Status:** ✅ COMPLETE (design done by TEAM_023, reviewed by TEAM_024)  
**Parent:** `.plans/forgejo-implementation/`  
**Depends On:** [Phase 1: Discovery](phase-1.md)

---

## 1. Proposed Solution

Mirror the PostgreSQL VM architecture exactly:

1. **TAP Networking** with NAT for internet access
2. **Custom init.sh** that bypasses OpenRC
3. **Persistent data disk** for Tailscale state and Forgejo data
4. **Console via ttyS0** for debugging
5. **Tailscale serve** for port exposure

---

## 2. Network Subnet Allocation

| Service | TAP Interface | Host IP | Guest IP | Tailscale Hostname |
|---------|---------------|---------|----------|-------------------|
| PostgreSQL | vm_sql | 192.168.100.1 | 192.168.100.2 | sovereign-sql |
| **Forgejo** | **vm_forge** | **192.168.101.1** | **192.168.101.2** | **sovereign-forge** |
| Vaultwarden | vm_vault | 192.168.102.1 | 192.168.102.2 | sovereign-vault |

---

## 3. Port Exposure Design

Forgejo exposes via Tailscale:

| Port | Service | Tailscale Command |
|------|---------|-------------------|
| 3000 | Web UI | `tailscale serve --bg --tcp 3000 3000` |
| 22 | SSH (git) | `tailscale serve --bg --tcp 22 22` |

---

## 4. Database Connectivity

**Decision:** Use Tailscale for Forgejo → PostgreSQL connection.

**Rationale:**
- Tailscale handles reconnection automatically
- Works even if VMs restart in different order
- No dependency on TAP network being up

**app.ini configuration:**
```ini
[database]
HOST = sovereign-sql:5432
```

---

## 5. Init Script Design

The init script must:

1. Mount essential filesystems (`/proc`, `/sys`, `/dev`)
2. Mount data disk (`/dev/vdb` → `/data`)
3. Set system time for TLS
4. Configure TAP networking (192.168.101.2)
5. Start Tailscale with persistent state
6. Wait for PostgreSQL database
7. Start Forgejo
8. Supervision loop (restart services if they die)

See [phase-3-step-1.md](phase-3-step-1.md) for implementation details.

---

## 6. Behavioral Decisions

### Q1: What happens if PostgreSQL is not available?

**Decision:** Wait up to 60 seconds, then start Forgejo anyway.
- Forgejo will show database error but won't crash
- User can fix database and Forgejo will reconnect

### Q2: What happens if Tailscale state exists but authkey is also provided?

**Decision:** Use existing state (ignore authkey).
- Preserves machine identity
- Only use authkey on first boot

### Q3: What order should services start?

**Decision:** 
1. Networking first (TAP)
2. Tailscale second (needs network)
3. Wait for PostgreSQL (needs Tailscale)
4. Forgejo last (needs database)

---

## 7. Design Alternatives Considered

### Alternative 1: Use vsock/gvproxy
**Rejected:** Doesn't work in AVF environment. Proven broken after weeks of debugging.

### Alternative 2: Use OpenRC
**Rejected:** Hangs during sysinit in crosvm. Custom init.sh is the only working approach.

### Alternative 3: Use Alpine tailscale package
**Rejected:** Static binary is more reliable and consistent with SQL VM.

---

## 8. Questions (All Answered)

See `.questions/TEAM_024_forgejo_decisions.md` for full tracking.

| Question | Decision |
|----------|----------|
| DB connection method | Tailscale (`sovereign-sql`) |
| Ports to expose | 3000 (web) + 22 (SSH) |
| Shared kernel | Defer (use SQL VM kernel) |

---

## 9. Future Optimization (NOT for initial implementation)

> **TEAM_024 Note:** This is deferred. Get Forgejo working first.

Common patterns that COULD be extracted after Forgejo works:
- `internal/vm/common/networking.go` - TAP setup/teardown
- `internal/vm/common/tailscale.go` - Registration cleanup
- `vm/common/init.sh.tmpl` - Shared init template

---

## Next Phase

→ [Phase 3: Implementation](phase-3.md)
