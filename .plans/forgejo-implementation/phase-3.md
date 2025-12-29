# Phase 3: Implementation

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** `.plans/forgejo-implementation/`  
**Depends On:** [Phase 2: Design](phase-2.md)

---

## Overview

This phase implements the Forgejo VM following the design from Phase 2. It is split into 4 steps:

| Step | Focus | Files |
|------|-------|-------|
| [Step 1](phase-3-step-1.md) | Init Script | `vm/forgejo/init.sh` |
| [Step 2](phase-3-step-2.md) | Start Script | `vm/forgejo/start.sh` |
| [Step 3](phase-3-step-3.md) | Dockerfile | `vm/forgejo/Dockerfile` |
| [Step 4](phase-3-step-4.md) | Go Code | `internal/vm/forge/*.go` |

---

## Prerequisites

Before starting implementation:

- [ ] Phase 1 and 2 are complete
- [ ] All questions in `.questions/TEAM_024_forgejo_decisions.md` are answered
- [ ] SQL VM is working (`sovereign test --sql` passes)
- [ ] BDD tests written (Phase 4, Step 1 should run first if using TDD)

---

## Step Summary

### Step 1: Init Script
Create `vm/forgejo/init.sh` modeled after `vm/sql/init.sh`:
- Mount essential filesystems
- Mount persistent data disk
- Configure TAP networking (192.168.101.2)
- Start Tailscale with persistent identity
- Wait for PostgreSQL
- Start Forgejo
- Supervision loop

### Step 2: Start Script  
Rewrite `vm/forgejo/start.sh` to use TAP networking:
- Remove gvproxy/vsock code
- Set up TAP interface `vm_forge`
- Configure NAT and Android routing bypass
- Use `console=ttyS0`
- Pass data.img as second block device

### Step 3: Dockerfile
Update `vm/forgejo/Dockerfile`:
- Use static Tailscale binary (not Alpine package)
- Remove OpenRC configuration
- Install init.sh to `/sbin/init.sh`

### Step 4: Go Code
Refactor `internal/vm/forge/`:
- **CRITICAL:** Fix device path to `/data/sovereign/vm/forgejo/`
- Split into `forge.go`, `lifecycle.go`, `verify.go`
- Remove gvproxy references
- Add TAP networking in Start()

---

## Next Phase

→ [Phase 4: Integration & Testing](phase-4.md)
