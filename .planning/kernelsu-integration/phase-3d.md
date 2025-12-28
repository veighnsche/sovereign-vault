# Phase 3D — Forgejo: Git Hosting

**Feature:** Sovereign Vault
**Phase:** 3D of 4
**Team:** TBD
**Status:** Preview
**Depends On:** Phase 3C (Vaultwarden) - Must be complete

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I have not started this phase yet. But I know my future self will be tempted to:
> - Use SQLite instead of PostgreSQL because "Forgejo supports both"
> - Set HTTP_ADDR to `127.0.0.1` which breaks external access
> - Skip Tailscale serve and claim "it works locally"
> - Not test git push/pull operations
>
> **The rule:** Forgejo MUST use PostgreSQL (DB_TYPE=postgres). HTTP_ADDR MUST be `0.0.0.0`. HTTPS MUST work via Tailscale serve.
>
> **TEAM_030's sin:** They switched Forgejo to SQLite and set HTTP_ADDR to 127.0.0.1. Both were wrong.

---

## 1. Phase Goal

**Run Forgejo in a pKVM VM, using PostgreSQL, accessible via Tailscale HTTPS.**

---

## 2. Success Criteria

- [ ] Forgejo VM builds successfully
- [ ] Forgejo connects to sql-vm PostgreSQL
- [ ] Forgejo accessible via `https://forge.<tailnet>/`
- [ ] Can create repository
- [ ] Can git clone, push, pull via HTTPS

---

## 3. Steps (Preview)

| Step | Description | Status |
|------|-------------|--------|
| 1 | Forgejo VM Setup (Alpine + Forgejo binary) | [ ] |
| 2 | Deploy & Configure (DB_TYPE=postgres, HTTP_ADDR=0.0.0.0) | [ ] |
| 3 | Tailscale serve + HTTPS + Verify | [ ] |

---

## 4. sovereign CLI Commands (Phase 3D)

```bash
# Build Forgejo VM
go run sovereign.go build --forge

# Deploy to device
go run sovereign.go deploy --forge

# Start the VM
go run sovereign.go start --forge

# Test connectivity
go run sovereign.go test --forge

# Stop the VM
go run sovereign.go stop --forge
```

---

## 5. VM Specifications (Planned)

| Property | Value |
|----------|-------|
| Role | forge |
| CID (VSOCK) | 12 |
| IP Address | 192.168.12.2 |
| TAP Interface | sovereign_forge |
| Gateway | 192.168.12.1 |
| Memory | 1536 MB |
| vCPUs | 2 |
| Disk Image | forgejo.img (8GB) |
| Tailscale hostname | forge-vm |
| Exposed port | 443 (HTTPS via Tailscale serve) |

---

## 6. Critical Configuration

**app.ini must have:**
```ini
[database]
DB_TYPE = postgres
HOST = sql-vm:5432
NAME = forgejo
USER = postgres
PASSWD = sovereign

[server]
HTTP_ADDR = 0.0.0.0
HTTP_PORT = 3000
ROOT_URL = https://forge.<tailnet>/
```

**NOT SQLite. NOT DB_TYPE=sqlite3.**
**NOT HTTP_ADDR=127.0.0.1 (this blocks external access).**

---

## 7. Phase 3D Complete When

| Criterion | Test | Expected |
|-----------|------|----------|
| VM running | `pgrep crosvm.*forge` | Process exists |
| On Tailscale | `tailscale status \| grep forge-vm` | forge-vm appears |
| HTTPS works | `curl https://forge.<tailnet>/` | Forgejo page |
| DB connected | Check forgejo logs | "ORM engine initialized" |
| Git clone | `git clone https://forge.<tailnet>/user/repo.git` | Clones successfully |
| Git push | Push a commit | Push succeeds |

---

## 8. Detailed Steps (To Be Written)

Step files will be created when Phase 3C is complete:
- `phase-3d-step-1.md` — Forgejo VM Setup
- `phase-3d-step-2.md` — Deploy & Configure
- `phase-3d-step-3.md` — Tailscale HTTPS + Verify

---

## After Phase 3D

When all phases complete, Sovereign Vault is operational:

```
┌─────────────────────────────────────────────────────┐
│                  SOVEREIGN VAULT                     │
├─────────────────────────────────────────────────────┤
│  Phase 3A: KernelSU        ✓ Root access            │
│  Phase 3B: PostgreSQL      ✓ Database via Tailscale │
│  Phase 3C: Vaultwarden     ✓ Passwords via HTTPS    │
│  Phase 3D: Forgejo         ✓ Git via HTTPS          │
└─────────────────────────────────────────────────────┘
```

Proceed to **Phase 4 — Integration Testing** and **Phase 5 — Documentation & Handoff**.
