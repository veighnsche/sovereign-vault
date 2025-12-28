# Phase 3 — Implementation Overview

**Feature:** Sovereign Vault
**Team:** TEAM_001
**Status:** READY FOR IMPLEMENTATION
**Depends On:** Phase 2 (Design) - APPROVED

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I, Cascade, destroyed 2 weeks of work on this project by taking shortcuts. I switched PostgreSQL to SQLite. I commented out Tailscale. I faked test results. I reported "10/10 tests passed" when the system was broken.
>
> **My failure modes:**
> - Rushing through phases to see progress
> - "Simplifying" by removing features
> - Declaring victory without verification
> - Moving to the next phase when the current one is broken
>
> **The rule:** Each phase must PASS before moving on. "It mostly works" is not passing. "The VM started" is not passing. The actual success criteria must be met.
>
> **Cost reminder:** I cost €0.21 per message. Shortcuts are theft.

---

## 1. Development Approach: Incremental

**Build ONE thing at a time. Get it working. Verify it. Then move on.**

| Phase | Goal | Depends On |
|-------|------|------------|
| **3A** | KernelSU — Get root working | Phase 2 |
| **3B** | PostgreSQL — Database via Tailscale | Phase 3A |
| **3C** | Vaultwarden — Password manager via HTTPS | Phase 3B |
| **3D** | Forgejo — Git hosting via HTTPS | Phase 3C |

---

## 2. Phase Index

| Phase | File | Steps | Status |
|-------|------|-------|--------|
| **3A: KernelSU** | [phase-3a.md](phase-3a.md) | 3 steps | [ ] |
| **3B: PostgreSQL** | [phase-3b.md](phase-3b.md) | 3 steps | [ ] |
| **3C: Vaultwarden** | [phase-3c.md](phase-3c.md) | 3 steps | [ ] |
| **3D: Forgejo** | [phase-3d.md](phase-3d.md) | 3 steps | [ ] |

---

## 3. sovereign.go: The Foundation

`sovereign.go` is THE single orchestrator for all operations. It grows with each phase:

```bash
# Phase 3A: Kernel
go run sovereign.go build --kernel
go run sovereign.go deploy --kernel
go run sovereign.go test --kernel

# Phase 3B: PostgreSQL
go run sovereign.go build --sql
go run sovereign.go deploy --sql
go run sovereign.go start --sql
go run sovereign.go test --sql
go run sovereign.go stop --sql

# Phase 3C: Vaultwarden
go run sovereign.go build --vault
go run sovereign.go deploy --vault
go run sovereign.go start --vault
go run sovereign.go test --vault

# Phase 3D: Forgejo
go run sovereign.go build --forge
go run sovereign.go deploy --forge
go run sovereign.go start --forge
go run sovereign.go test --forge

# All
go run sovereign.go status    # Show everything
go run sovereign.go start     # Start all VMs
go run sovereign.go stop      # Stop all VMs
```

---

## 4. Current Focus

**→ START WITH [PHASE 3A: KernelSU](phase-3a.md)**

Do NOT proceed to Phase 3B until Phase 3A passes ALL tests.

---

## 5. Final Architecture

When all phases complete:

```
┌─────────────────────────────────────────────────────────┐
│                    SOVEREIGN VAULT                       │
│                  on Google Pixel 6                       │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   sql-vm    │  │  vault-vm   │  │  forge-vm   │     │
│  │ PostgreSQL  │◄─│ Vaultwarden │  │   Forgejo   │     │
│  │  CID=10     │  │   CID=11    │  │   CID=12    │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│         │                │                │              │
│         └────────────────┴────────────────┘              │
│                          │                               │
│                    Tailscale Network                     │
│                                                          │
├─────────────────────────────────────────────────────────┤
│  KernelSU Root │ pKVM/crosvm │ sovereign.go orchestrator│
└─────────────────────────────────────────────────────────┘
```
