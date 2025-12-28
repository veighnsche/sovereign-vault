# Phase 3B — PostgreSQL: Database via Tailscale

**Feature:** Sovereign Vault
**Phase:** 3B of 4
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Phase 3A (KernelSU) - Must be complete

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I am about to work with VMs, pKVM, and Tailscale networking. This is where TEAM_030 destroyed everything by:
> - Switching from PostgreSQL to SQLite because "it's simpler"
> - Commenting out Tailscale because "networking is hard"
> - Faking test results to see green checkmarks
>
> **My failure modes in this phase:**
> - Building a VM that works locally but fails on device
> - Skipping Tailscale because "we can add it later"
> - Using hardcoded IPs instead of proper Tailscale DNS
> - Declaring success when PostgreSQL starts but isn't reachable
>
> **The rule:** PostgreSQL MUST be accessible via Tailscale from another machine. Local-only is not success.
>
> **Remember:** The architecture says PostgreSQL, not SQLite. I do not "simplify" by changing the architecture.

---

## 1. Phase Goal

**Run PostgreSQL in a pKVM VM, accessible via Tailscale.**

This provides the database backend for Vaultwarden and Forgejo.

---

## 2. Success Criteria

- [ ] PostgreSQL VM builds successfully
- [ ] VM starts under pKVM/crosvm
- [ ] VM joins Tailscale network
- [ ] PostgreSQL accessible from another Tailscale device
- [ ] `psql -h <tailscale-ip> -U postgres` works

---

## 3. Pre-Conditions

- [ ] **Phase 3A complete** (root access working)
- [ ] Docker or Podman installed on dev machine
- [ ] Tailscale auth key (https://login.tailscale.com/admin/settings/keys)

---

## 4. Steps

| Step | File | Description | Status |
|------|------|-------------|--------|
| 1 | [phase-3b-step-1.md](phase-3b-step-1.md) | PostgreSQL VM Setup | [ ] |
| 2 | [phase-3b-step-2.md](phase-3b-step-2.md) | Deploy & Start VM | [ ] |
| 3 | [phase-3b-step-3.md](phase-3b-step-3.md) | Tailscale + Verify | [ ] |

---

## 5. sovereign CLI Commands (Phase 3B)

```bash
# Build PostgreSQL VM image
go run sovereign.go build --sql

# Deploy to device
go run sovereign.go deploy --sql

# Start the VM
go run sovereign.go start --sql

# Test connectivity
go run sovereign.go test --sql

# Check status
go run sovereign.go status

# Stop the VM
go run sovereign.go stop --sql
```

---

## 6. VM Specifications

| Property | Value |
|----------|-------|
| Role | database |
| CID (VSOCK) | 10 |
| IP Address | 192.168.10.2 |
| TAP Interface | sovereign_sql |
| Gateway | 192.168.10.1 |
| Memory | 1024 MB |
| vCPUs | 2 |
| Disk Image | postgres.img (4GB) |
| Tailscale hostname | sql-vm |
| Exposed port | 5432 (PostgreSQL) |

---

## 7. Phase 3B Complete When

All of the following are true:

| Criterion | Test | Expected |
|-----------|------|----------|
| VM running | `adb shell su -c 'ps \| grep crosvm'` | crosvm process exists |
| On Tailscale | `tailscale status \| grep sql-vm` | sql-vm appears |
| PostgreSQL listening | `nc -zv <tailscale-ip> 5432` | Connection succeeded |
| Can connect | `psql -h <tailscale-ip> -U postgres` | psql prompt |

---

## 8. If This Phase Fails

> 🤖 **AI Warning:** VM and networking issues are HARD. That's the point. If something fails:
> 1. Check VM console logs: `adb shell su -c 'cat /data/sovereign/sql-console.log'`
> 2. Check Tailscale status inside VM (if accessible)
> 3. Verify TAP interface exists on host
> 4. Check iptables rules
>
> I do NOT:
> - Switch to SQLite
> - Skip Tailscale
> - Fake the test results

---

## Next Phase

After Phase 3B passes all tests → **Phase 3C: Vaultwarden**
