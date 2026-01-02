# TEAM_039 — Root Cause Investigation and Documentation

## Mission
1. Investigate core issues instead of implementing workarounds
2. Clean up duplicated Tailscale machines
3. Update CLI tool to handle common operations (stop manual commands)
4. Document build/deploy/start/test for SQL, Vault, Forgejo

## Status: IN_PROGRESS

## Pre-Investigation Checklist
- [x] Team registered as TEAM_039
- [ ] Review workarounds added by TEAM_038
- [ ] Identify root causes vs symptoms
- [ ] Clean up Tailscale duplicates

## Bug Report Summary
TEAM_038 added several workarounds that may be masking deeper issues:
1. Added timeout to `tailscale up` - why does it hang?
2. Added ALTER USER for password - why doesn't CREATE work on fresh data.img?
3. Manual commands used instead of CLI tool

## Phase 1: Understand the Symptoms

### Symptom 1: Tailscale up hangs indefinitely
- **Expected**: `tailscale up` completes quickly when state exists
- **Actual**: Hangs forever, requiring timeout workaround
- **Question**: Why isn't Tailscale connecting with existing state?

### Symptom 2: Multiple Tailscale machines created
- **Expected**: One machine "sovereign-vault" reused across restarts
- **Actual**: sovereign-vault-1, -2, -3, -4, -5... created
- **Question**: Why isn't state being preserved correctly?

### Symptom 3: Password authentication failures despite ALTER USER
- **Expected**: CREATE USER sets password correctly
- **Actual**: Needed ALTER USER to fix password mismatch
- **Question**: Was the password ever wrong, or is there a timing issue?

## Hypotheses

### H1: Tailscale state file format changed
- Confidence: HIGH
- Evidence needed: Check tailscale version, state file format
- TEAM_038 fixed the grep check, but did it fix the root cause?

### H2: VMs dying due to Android process killer (not fully disabled)
- Confidence: MEDIUM
- Evidence needed: Check if phantom process killer is actually disabled

### H3: CLI tool has bugs that manual commands work around
- Confidence: HIGH  
- Evidence needed: Review CLI start/stop/deploy commands

## Root Causes Found and Fixed

### Issue 1: Tailscale State Detection (FIXED)
**Symptom**: New Tailscale machines created on every boot (sovereign-vault-1, -2, -3...)

**Root Cause**: The init.sh checked for state file AFTER starting tailscaled, but tailscaled creates the state file on startup. So the check always found a file (even if empty/new).

**Fix**: Check for existing state file BEFORE starting tailscaled.
```bash
# BEFORE (broken)
tailscaled --state=/data/tailscale/tailscaled.state &
sleep 3
if [ -f "$STATE_FILE" ]; then  # Always true!
    tailscale up ...

# AFTER (fixed)
if [ -f "$STATE_FILE" ] && [ -s "$STATE_FILE" ]; then
    HAS_EXISTING_STATE=true
fi
tailscaled --state=/data/tailscale/tailscaled.state &
sleep 3
if [ "$HAS_EXISTING_STATE" = "true" ]; then
    tailscale up ...
```

### Issue 2: No CLI Command for Tailscale Cleanup (FIXED)
**Symptom**: Manual deletion required for duplicate Tailscale machines

**Fix**: Added `./sovereign clean --vault` command to CLI. Implements Clean() method in VM interface, calls RemoveTailscaleRegistrations() via API.

## Files Modified

| File | Change |
|------|--------|
| `cmd/sovereign/main.go` | Added `clean` command |
| `internal/vm/vm.go` | Added `Clean()` to VM interface |
| `internal/vm/common/lifecycle.go` | Added `CleanVM()` function |
| `internal/vm/sql/lifecycle.go` | Implemented `Clean()` |
| `internal/vm/vault/lifecycle.go` | Implemented `Clean()` |
| `internal/vm/forge/lifecycle.go` | Implemented `Clean()` |
| `vm/vault/init.sh` | Fixed Tailscale state detection order |

## Documentation Created

- `.planning/VM_OPERATIONS_GUIDE.md` - Complete guide for build/deploy/start/test

## Handoff Checklist
- [x] Team registered
- [x] Root cause identified and fixed
- [x] CLI tool updated (not manual commands)
- [x] Tailscale duplicates cleaned up
- [x] Documentation created
- [x] Code compiles

## Investigation Log
