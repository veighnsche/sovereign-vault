# Phase 5 — Polish, Documentation, and Handoff

**Feature:** KernelSU integration for stock raviole kernel
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Phase 4 (Testing)

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I, Cascade (TEAM_001), am writing this handoff phase because proper handoff is how knowledge survives between AI sessions. If I don't document properly, the next AI will repeat my mistakes.
>
> **My failure modes to watch for:**
> - **Declaring victory too early** without completing documentation
> - **Leaving undocumented changes** that confuse future teams
> - **Skipping the handoff checklist** because "it's obvious"
> - **Not updating the team file** with final status
>
> **The rule:** The project is NOT complete until another AI can pick up where I left off without confusion.
>
> **If you are a future AI reading this:** Thank the previous team by reading their handoff notes. Then continue their work properly.

---

## 1. Documentation Updates

### 1.1 Update sovereign_vault.md

> 🤖 **AI Warning:** The sovereign_vault.md is the AUTHORITATIVE REFERENCE. If it doesn't match reality, future teams will be confused. Update it.

**Section 5 (Kernel Architecture):**
- [ ] Confirm kernel version documented (should be 6.1.124)
- [ ] Confirm dual kernel design still accurate
- [ ] Update any paths that changed

**Section 6 (KernelSU Implementation):**
- [ ] Confirm CONFIG_KSU=y documented
- [ ] Confirm LOCALVERSION="-sovereign" documented
- [ ] Add the symlink location: `aosp/drivers/kernelsu`
- [ ] Document the defconfig fragment: `kernelsu.fragment`

**Section 9 (Build System):**
- [ ] Confirm build command documented
- [ ] Add any new build steps if needed

### 1.2 Update Team File

**File:** `.teams/TEAM_001_kernelsu_integration.md`

**Required Updates:**
- [ ] Final progress log entry
- [ ] List of all files modified
- [ ] Any gotchas or lessons learned
- [ ] Handoff notes for next team
- [ ] Completion status (DONE / BLOCKED / PARTIAL)

---

## 2. Code Cleanup

> 🤖 **AI Warning:** Dead code is debt. Test artifacts left behind confuse future teams. Clean up.

### 2.1 Remove Test Artifacts

- [ ] Remove `/data/adb/service.d/test_ksu.sh` from device (if created during testing)
- [ ] Remove `/data/local/tmp/ksu_boot_test_*` marker files
- [ ] Remove any temporary scripts created during debugging

### 2.2 Verify No Debug Code

- [ ] Check `kernelsu.fragment` has no debug options enabled
- [ ] Verify `CONFIG_KSU_DEBUG` is NOT set
- [ ] No temporary logging added to source files

### 2.3 File Hygiene

- [ ] All modified files have TEAM_001 comments where appropriate
- [ ] No trailing whitespace or formatting issues introduced
- [ ] Symlink is relative, not absolute

---

## 3. Handoff Checklist

> 🤖 **AI Warning:** This checklist is from Rule 10. Do NOT skip any item. A skipped item is a lie.

### 3.1 Build Verification

- [ ] `./build_raviole.sh` completes without errors
- [ ] `grep CONFIG_KSU out/raviole/dist/.config` shows `CONFIG_KSU=y`
- [ ] `out/raviole/dist/boot.img` exists

### 3.2 Test Verification

- [ ] TC-1 (Boot): PASS
- [ ] TC-2 (KernelSU Detection): PASS
- [ ] TC-3 (Root Shell): PASS
- [ ] TC-4 (Boot Script): PASS
- [ ] TC-5 (Sovereign Vault): PASS or documented blocker
- [ ] TC-6 (Regression): No critical failures

### 3.3 Documentation Verification

- [ ] sovereign_vault.md updated
- [ ] Team file updated with final status
- [ ] Phase files reflect actual work done
- [ ] Any open questions documented in `.questions/`

### 3.4 Final Verification

- [ ] No open questions remaining (or all documented)
- [ ] No uncommitted changes in working directory
- [ ] TODOs tracked if any remain

---

## 4. Future Improvements

> 🤖 **AI Note:** These are tracked in sovereign_vault.md TODO section. Do not create duplicate tracking.

| Item | Priority | Notes | Tracked In |
|------|----------|-------|------------|
| Guest kernel with sovereign config | Medium | For VM kernel, not host | Deferred to separate feature |
| Automated KernelSU version updates | Low | Script to update symlink | Future improvement |
| CI/CD integration | Low | Auto-build on commit | Future improvement |
| Remove --disable-sandbox from crosvm | High | Security hardening | SEC-3 in sovereign_vault.md |

---

## 5. Completion Criteria

> 🤖 **AI Warning:** Do not mark complete until ALL criteria are met. Partial completion is NOT completion.

Feature is **COMPLETE** when ALL of the following are true:

| Criterion | Evidence Required | Status |
|-----------|-------------------|--------|
| Kernel builds with KernelSU | `CONFIG_KSU=y` in .config | [ ] |
| Device boots normally | TC-1 PASS | [ ] |
| KernelSU detected by manager | TC-2 PASS | [ ] |
| Root shell works | TC-3 PASS | [ ] |
| Boot scripts execute | TC-4 PASS | [ ] |
| Sovereign Vault can start | TC-5 PASS | [ ] |
| No critical regressions | TC-6 PASS | [ ] |
| sovereign_vault.md updated | Diff shows changes | [ ] |
| Team file complete | Handoff notes present | [ ] |
| Handoff checklist complete | All items checked | [ ] |

---

## 6. Handoff Notes Template

When completing this phase, add the following to the team file:

```markdown
## Handoff Notes

### What Was Done
- [List of changes made]

### Files Modified
- `aosp/drivers/kernelsu` (symlink)
- `aosp/drivers/Makefile`
- `aosp/drivers/Kconfig`
- `private/devices/google/raviole/kernelsu.fragment`
- `private/devices/google/raviole/BUILD.bazel`

### Known Issues
- [Any issues discovered but not fixed]

### Lessons Learned
- [What would you do differently?]

### For Next Team
- [What should the next team know?]
```

---

## 7. Final Sign-Off

When all criteria are met:

1. Update team file status to **COMPLETE**
2. Add final progress log entry with date
3. Verify all phase files are accurate
4. Notify USER that feature is ready

> 🤖 **AI Reminder:** Completing a feature properly is not "extra work." It is the job. Do it right.
