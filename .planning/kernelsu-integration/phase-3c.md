# Phase 3C — Vaultwarden: Password Manager

**Feature:** Sovereign Vault
**Phase:** 3C of 4
**Team:** TBD
**Status:** Preview
**Depends On:** Phase 3B (PostgreSQL) - Must be complete

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I have not started this phase yet. But I know my future self will be tempted to:
> - Use SQLite instead of PostgreSQL because "it's easier"
> - Skip the `postgresql` feature flag when building Vaultwarden
> - Hardcode `127.0.0.1` instead of making it Tailscale-accessible
> - Claim success when Vaultwarden starts but can't reach the database
>
> **The rule:** Vaultwarden MUST use PostgreSQL. DATABASE_URL MUST point to the sql-vm. HTTPS MUST work via Tailscale serve.
>
> **TEAM_030's sin:** They switched Vaultwarden to SQLite. This destroyed the architecture. Do not repeat.

---

## 1. Phase Goal

**Run Vaultwarden in a pKVM VM, using PostgreSQL, accessible via Tailscale HTTPS.**

---

## 2. Success Criteria

- [ ] Vaultwarden VM builds with `postgresql` feature enabled
- [ ] Vaultwarden connects to sql-vm PostgreSQL
- [ ] Vaultwarden accessible via `https://vault.<tailnet>/`
- [ ] Can create account and store password
- [ ] Bitwarden clients can connect

---

## 3. Steps (Preview)

| Step | Description | Status |
|------|-------------|--------|
| 1 | Vaultwarden VM Setup (Alpine + Vaultwarden with postgresql feature) | [ ] |
| 2 | Deploy & Configure (DATABASE_URL=postgresql://sql-vm:5432) | [ ] |
| 3 | Tailscale serve + HTTPS + Verify | [ ] |

---

## 4. sovereign CLI Commands (Phase 3C)

```bash
# Build Vaultwarden VM
go run sovereign.go build --vault

# Deploy to device
go run sovereign.go deploy --vault

# Start the VM
go run sovereign.go start --vault

# Test connectivity
go run sovereign.go test --vault

# Stop the VM
go run sovereign.go stop --vault
```

---

## 5. VM Specifications (Planned)

| Property | Value |
|----------|-------|
| Role | vault |
| CID (VSOCK) | 11 |
| IP Address | 192.168.11.2 |
| TAP Interface | sovereign_vault |
| Gateway | 192.168.11.1 |
| Memory | 1024 MB |
| vCPUs | 1 |
| Disk Image | vault.img (1GB) |
| Tailscale hostname | vault-vm |
| Exposed port | 443 (HTTPS via Tailscale serve) |

---

## 6. Critical Configuration

**DATABASE_URL must be:**
```
postgresql://postgres:sovereign@sql-vm:5432/vaultwarden
```

**NOT SQLite. NOT localhost. MUST be sql-vm via Tailscale DNS.**

---

## 7. Phase 3C Complete When

| Criterion | Test | Expected |
|-----------|------|----------|
| VM running | `pgrep crosvm.*vault` | Process exists |
| On Tailscale | `tailscale status \| grep vault-vm` | vault-vm appears |
| HTTPS works | `curl https://vault.<tailnet>/` | Vaultwarden page |
| DB connected | Check vault logs | "Connected to PostgreSQL" |
| Client works | Bitwarden app connects | Can sync |

---

## 8. Detailed Steps (To Be Written)

Step files will be created when Phase 3B is complete:
- `phase-3c-step-1.md` — Vaultwarden VM Setup
- `phase-3c-step-2.md` — Deploy & Configure
- `phase-3c-step-3.md` — Tailscale HTTPS + Verify

---

## Next Phase

After Phase 3C passes all tests → **Phase 3D: Forgejo**
