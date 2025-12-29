# Forgejo Implementation Plan

**Status:** 🔲 READY FOR IMPLEMENTATION  
**Created:** 2025-12-29 by TEAM_023  
**Reviewed:** 2025-12-29 by TEAM_024  
**Split:** 2025-12-29 by TEAM_024

---

## Quick Start

1. Read [Phase 1](phase-1.md) for context
2. Check [Phase 2](phase-2.md) for design decisions
3. Start implementation at [Phase 3](phase-3.md)

---

## Phase Overview

| Phase | Purpose | Status |
|-------|---------|--------|
| [Phase 1](phase-1.md) | Discovery | ✅ Complete |
| [Phase 2](phase-2.md) | Design | ✅ Complete |
| [Phase 3](phase-3.md) | Implementation | 🔲 Not Started |
| [Phase 4](phase-4.md) | Integration & Testing | 🔲 Not Started |
| [Phase 5](phase-5.md) | Polish & Cleanup | 🔲 Not Started |

---

## Phase 3 Steps (Implementation)

| Step | Focus | Files |
|------|-------|-------|
| [Step 1](phase-3-step-1.md) | Init Script | `vm/forgejo/init.sh` |
| [Step 2](phase-3-step-2.md) | Start Script | `vm/forgejo/start.sh` |
| [Step 3](phase-3-step-3.md) | Dockerfile | `vm/forgejo/Dockerfile` |
| [Step 4](phase-3-step-4.md) | Go Code | `internal/vm/forge/*.go` |

---

## Key Files

| File | Purpose |
|------|---------|
| `vm/sql/init.sh` | **GOLD STANDARD** - copy patterns |
| `vm/sql/start.sh` | TAP networking reference |
| `.questions/TEAM_024_forgejo_decisions.md` | Design decisions |

---

## Critical Reminders

1. **Device path:** `/data/sovereign/vm/forgejo/` (NOT `/data/sovereign/forgejo/`)
2. **TAP interface:** `vm_forge` at 192.168.101.x
3. **Tailscale hostname:** `sovereign-forge`
4. **Console:** `ttyS0` (NOT `hvc0`)
5. **Init:** Custom `init.sh` (NOT OpenRC)

---

## Document History

| Date | Team | Change |
|------|------|--------|
| 2025-12-29 | TEAM_023 | Created monolithic plan |
| 2025-12-29 | TEAM_024 | Reviewed, corrected issues |
| 2025-12-29 | TEAM_024 | Split into phase files |
