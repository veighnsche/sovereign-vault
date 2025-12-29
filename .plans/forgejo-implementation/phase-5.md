# Phase 5: Polish & Cleanup

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** `.plans/forgejo-implementation/`  
**Depends On:** [Phase 4: Integration & Testing](phase-4.md)

---

## Overview

Final polish, documentation updates, and cleanup of dead code.

---

## Tasks

### Task 1: Fix app.ini Hostnames

Update `vm/forgejo/config/app.ini`:

```ini
[server]
DOMAIN = sovereign-forge
ROOT_URL = https://sovereign-forge/
SSH_DOMAIN = sovereign-forge

[database]
HOST = sovereign-sql:5432
```

### Task 2: Remove Dead Code

Delete from codebase:
- Any gvproxy references in forge code
- Old/unused scripts in `vm/forgejo/`
- Commented-out code

### Task 3: Update Documentation

Files to update:
- `README.md` - Add Forgejo section
- `vm/forgejo/CHECKLIST.md` - Update with lessons learned
- Add `vm/forgejo/README.md` if needed

### Task 4: Update Team File

Document in `.teams/TEAM_0XX_*.md`:
- Summary of changes made
- Files modified
- Tests passing
- Handoff notes

### Task 5: Final Verification

```bash
# Full cycle test
sovereign stop --forge
sovereign remove --forge
sovereign build --forge
sovereign deploy --forge
sovereign start --forge
sovereign test --forge

# Verify no Tailscale duplicates after restart
sovereign stop --forge
sovereign start --forge
tailscale status | grep -c sovereign-forge  # Should be 1
```

---

## Handoff Checklist

Before marking complete:

- [ ] Project builds cleanly (`go build ./...`)
- [ ] All BDD tests pass
- [ ] `sovereign test --sql` still passes (no regression)
- [ ] `sovereign test --forge` passes
- [ ] Both VMs run simultaneously
- [ ] Tailscale shows exactly one `sovereign-forge` registration
- [ ] app.ini uses correct hostnames
- [ ] No dead code or gvproxy references remain
- [ ] Team file updated with summary
- [ ] Documentation updated

---

## Success Criteria (From Phase 1)

Verify all original criteria are met:

1. ✅ All BDD tests pass
2. ✅ `sovereign build --forge` creates rootfs.img and data.img
3. ✅ `sovereign deploy --forge` pushes files to device
4. ✅ `sovereign start --forge` starts VM with TAP networking
5. ✅ Forgejo web UI accessible via Tailscale
6. ✅ Git operations work via SSH through Tailscale
7. ✅ Restart preserves Tailscale identity (no duplicates)
8. ✅ Forgejo connects to PostgreSQL on SQL VM
9. ✅ Both VMs run simultaneously without conflict

---

## Feature Complete

When all checkboxes above are checked, the Forgejo implementation is complete.

Update the main plan file:
```
## Status: ✅ COMPLETE
```
