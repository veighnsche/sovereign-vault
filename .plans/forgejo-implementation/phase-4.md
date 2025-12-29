# Phase 4: Integration & Testing

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** `.plans/forgejo-implementation/`  
**Depends On:** [Phase 3: Implementation](phase-3.md)

---

## Overview

This phase enables and expands the BDD tests, then runs full integration testing.

| Step | Focus | Files |
|------|-------|-------|
| [Step 1](phase-4-step-1.md) | Enable BDD Tests | `features/forge.feature` |
| [Step 2](phase-4-step-2.md) | Integration Testing | Run full test suite |

---

## Step 1: Enable and Expand BDD Tests

### Task 1: Rename feature file

```bash
mv features/forge.feature.disabled features/forge.feature
```

### Task 2: Update scenarios to match implementation

The existing `forge.feature.disabled` has 85 lines with basic scenarios. Expand to cover:

**Build Behaviors:**
- Build with Docker
- Build fails without shared kernel
- Dockerfile uses correct Alpine version
- Dockerfile installs Tailscale static binary
- Rootfs creates init.sh (not OpenRC)

**Deploy Behaviors:**
- Deploy creates VM directory at `/data/sovereign/vm/forgejo`
- Deploy creates start.sh with TAP networking
- Deploy does NOT include gvproxy

**Start Behaviors:**
- Start creates TAP interface `vm_forge`
- Start configures guest IP 192.168.101.2
- Start registers Tailscale as `sovereign-forge`
- Restart preserves Tailscale identity
- Start waits for PostgreSQL

**Stop Behaviors:**
- Stop kills VM process
- Stop removes TAP interface
- Stop is idempotent

**Test Behaviors:**
- Test checks VM process
- Test checks Tailscale
- Test checks Forgejo web UI

**Multi-VM Behaviors:**
- SQL and Forge run simultaneously
- Forge connects to SQL database

---

## Step 2: Integration Testing

### Prerequisites

- [ ] SQL VM is running (`sovereign test --sql` passes)
- [ ] All Phase 3 implementation complete
- [ ] BDD tests updated

### Test Sequence

```bash
# 1. Build Forgejo VM
sovereign build --forge

# 2. Deploy to device
sovereign deploy --forge

# 3. Start VM
sovereign start --forge

# 4. Run tests
sovereign test --forge

# 5. Verify multi-VM
# Both should be running
adb shell su -c "ps -ef | grep crosvm"

# 6. Check Tailscale
tailscale status | grep sovereign

# 7. Access Forgejo web UI
curl -I http://sovereign-forge:3000
```

### Success Criteria

- [ ] `sovereign build --forge` completes without error
- [ ] `sovereign deploy --forge` creates correct directory structure
- [ ] `sovereign start --forge` shows "INIT COMPLETE"
- [ ] `sovereign test --forge` shows all tests passing
- [ ] Tailscale shows `sovereign-forge` connected
- [ ] Forgejo web UI responds on port 3000
- [ ] Restart does not create duplicate Tailscale registration

---

## Expected Outputs

- Enabled: `features/forge.feature` (~200 lines)
- All BDD scenarios passing
- Integration test log showing success

---

## Next Phase

→ [Phase 5: Polish & Cleanup](phase-5.md)
