# TEAM_003 — Plan Review: Coherence and Alignment

**Created:** 2024-12-27
**Status:** Active
**Task:** Review plan coherence, ensure sovereign.go is fundamental, verify priority ordering

---

## Mission

1. Ensure `sovereign.go` is fundamental to all plans
2. Verify priority: Kernel → PostgreSQL → Vault → Forge
3. Check vault/forge documentation
4. Review overall plan coherence

---

## Progress Log

| Date       | Action                                      |
|------------|---------------------------------------------|
| 2024-12-27 | Team registered, beginning coherence review |
| 2024-12-27 | Confirmed vault/forge documented in sovereign_vault.md |
| 2024-12-27 | Reviewed all 11 plan files |
| 2024-12-27 | Identified coherence issues |
| 2024-12-27 | Applied fixes to phase-3.md and phase-3-step-1.md |

---

## Findings

### Vault and Forge Documentation
**Status:** ✓ Documented in `sovereign_vault.md`
- **vault** = Vaultwarden (password manager, Bitwarden-compatible)
- **forge** = Forgejo (Git hosting and CI/CD)

### Plan Coherence Issues Found

| Issue | Severity | Resolution |
|-------|----------|------------|
| sovereign.go not positioned as foundation | Medium | **FIXED** - Rewrote phase-3-step-1.md |
| Vault/Forge mentioned only as "future" | Low | **FIXED** - Added preview in phase-3.md |
| Priority ordering unclear | Low | **FIXED** - Phase 3A→3B→3C→3D structure |

### Final Coherence Score: **8/10** (After fixes)

---

## Files Modified

| File | Change |
|------|--------|
| `phase-3.md` | **REWRITTEN** as overview with links to sub-phases |
| `phase-3a.md` | **NEW** - KernelSU phase overview |
| `phase-3a-step-1.md` | **NEW** - sovereign.go: The Foundation |
| `phase-3a-step-2.md` | **NEW** - KernelSU Integration (patches) |
| `phase-3a-step-3.md` | **NEW** - Build, Deploy, Verify Root |
| `phase-3b.md` | **NEW** - PostgreSQL phase overview |
| `phase-3b-step-1.md` | **NEW** - PostgreSQL VM Setup |
| `phase-3b-step-2.md` | **NEW** - Deploy & Start VM |
| `phase-3b-step-3.md` | **NEW** - Tailscale + Verify |
| `phase-3c.md` | **NEW** - Vaultwarden preview |
| `phase-3d.md` | **NEW** - Forgejo preview |
| `phase-3-step-*.md` | **DELETED** - Old flat structure removed |

---

## Handoff Notes

### What Was Done
- Verified vault/forge are documented in sovereign_vault.md
- Reorganized flat structure into hierarchical: phase-3 → phase-3a → phase-3a-step-X
- Made sovereign.go THE foundation (not just a helper)
- Added previews for Phase 3C (Vaultwarden) and 3D (Forgejo)
- Maintained self-deprecating AI warning tone throughout
- All 11 new files have detailed tasks, verification steps, and checkpoints

### New File Structure
```
phase-3.md              (overview)
├── phase-3a.md         (KernelSU overview)
│   ├── phase-3a-step-1.md  (sovereign.go foundation)
│   ├── phase-3a-step-2.md  (KernelSU patches)
│   └── phase-3a-step-3.md  (build/deploy/verify)
├── phase-3b.md         (PostgreSQL overview)
│   ├── phase-3b-step-1.md  (VM setup)
│   ├── phase-3b-step-2.md  (deploy & start)
│   └── phase-3b-step-3.md  (Tailscale + verify)
├── phase-3c.md         (Vaultwarden preview)
└── phase-3d.md         (Forgejo preview)
```

### For Next Team
- Start with phase-3a.md → phase-3a-step-1.md
- Each step has checkpoints that MUST pass before proceeding
- sovereign.go grows with each phase (--kernel → --sql → --vault → --forge)
- Phase 3C/3D step files to be created when 3B completes
