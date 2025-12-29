# Forgejo Implementation Plan

## Status: SPLIT INTO PHASES

> **This plan has been split into structured phase files.**
> 
> **Go to:** [`.plans/forgejo-implementation/`](forgejo-implementation/README.md)

**Created:** 2025-12-29 by TEAM_023  
**Reviewed:** 2025-12-29 by TEAM_024  
**Split:** 2025-12-29 by TEAM_024

---

## Phase Index

| Phase | File | Status |
|-------|------|--------|
| Discovery | [phase-1.md](forgejo-implementation/phase-1.md) | ✅ Complete |
| Design | [phase-2.md](forgejo-implementation/phase-2.md) | ✅ Complete |
| Implementation | [phase-3.md](forgejo-implementation/phase-3.md) | 🔲 Not Started |
| Integration | [phase-4.md](forgejo-implementation/phase-4.md) | 🔲 Not Started |
| Polish | [phase-5.md](forgejo-implementation/phase-5.md) | 🔲 Not Started |

---

# Original Plan (Archived Below)

---

## 1. Executive Summary

After weeks of debugging, we have a **working PostgreSQL VM** with:
- TAP networking (not vsock)
- Tailscale with persistent machine identity (no duplicate registrations!)
- Persistent data disk mounted at `/data`
- Custom init.sh (not OpenRC - it hangs in AVF)
- Console output via `ttyS0` (not `hvc0`)

**Forgejo must mirror these patterns exactly.**

---

## 2. Current State Comparison

### What PostgreSQL Has (Working)

| Component | PostgreSQL Implementation | Status |
|-----------|---------------------------|--------|
| Networking | TAP interface `vm_sql` at 192.168.100.x | ✅ Working |
| Init System | Custom `/sbin/init.sh` (OpenRC hangs) | ✅ Working |
| Console | `console=ttyS0` with `--serial type=stdout` | ✅ Working |
| Data Disk | `/dev/vdb` mounted to `/data` (persistent) | ✅ Working |
| Tailscale | State persisted on data disk, reconnects on restart | ✅ Working |
| Port Exposure | `tailscale serve --tcp 5432 5432` | ✅ Working |
| BDD Tests | 47 scenarios in `sql.feature` | ✅ All passing |

### What Forgejo Has (Broken/Stale)

| Component | Forgejo Implementation | Status |
|-----------|------------------------|--------|
| Networking | Tries to use gvproxy/vsock | ❌ Wrong approach |
| Init System | Uses OpenRC via `/sbin/init` | ❌ Will hang |
| Console | `console=hvc0` with virtio-console | ❌ Wrong device |
| Data Disk | Passed to crosvm but not mounted in init | ⚠️ Incomplete |
| Tailscale | Uses Alpine package, registers every time | ❌ Duplicates |
| Port Exposure | Uses `tailscale serve --https=443` | ⚠️ Needs review |
| BDD Tests | 85 lines in `forge.feature.disabled` | ❌ Stale/disabled |

---

## 3. Gap Analysis: What Must Change

### 3.1 Start Script (`vm/forgejo/start.sh`)

**Current (Wrong):**
```bash
# Uses gvproxy and vsock - this doesn't work in AVF
$VM_DIR/bin/gvproxy -listen vsock://:1024 ...
crosvm run ... --cid 4 --serial type=stdout,hardware=virtio-console ...
```

**Must Change To (Mirror PostgreSQL):**
```bash
# TAP networking like PostgreSQL
TAP_NAME="vm_forge"
TAP_HOST_IP="192.168.101.1"
TAP_GUEST_IP="192.168.101.2"

# Set up TAP interface
ip tuntap add mode tap name ${TAP_NAME}
ip addr add ${TAP_HOST_IP}/24 dev ${TAP_NAME}
ip link set ${TAP_NAME} up

# NAT and forwarding
iptables -t nat -A POSTROUTING -s 192.168.101.0/24 -o wlan0 -j MASQUERADE
ip rule add from all lookup main pref 1  # CRITICAL for Android

# crosvm with TAP networking
crosvm run \
    --disable-sandbox \
    --mem 1024 \
    --cpus 2 \
    --block path="${VM_DIR}/rootfs.img",root \
    --block path="${VM_DIR}/data.img" \
    --params "console=ttyS0 root=/dev/vda rw init=/sbin/init.sh" \
    --serial type=stdout \
    --net tap-name=${TAP_NAME} \
    --socket "${VM_DIR}/vm.sock" \
    "${KERNEL}"
```

### 3.2 Init Script (`vm/forgejo/init.sh`)

**Must Create:** A new `/sbin/init.sh` like PostgreSQL's, NOT use OpenRC.

Key sections to include:
1. Mount essential filesystems (`/proc`, `/sys`, `/dev`)
2. Mount data disk (`/dev/vdb` → `/data`)
3. Set system time for TLS
4. Configure TAP networking (192.168.101.2)
5. Start Tailscale with persistent state
6. Wait for PostgreSQL database
7. Start Forgejo
8. Supervision loop

### 3.3 Dockerfile (`vm/forgejo/Dockerfile`)

**Changes Needed:**
- Use same Tailscale installation pattern as PostgreSQL (static binary, not Alpine package)
- Remove OpenRC configuration
- Ensure `init.sh` is properly installed

### 3.4 Go Code (`internal/vm/forge/`)

Currently exists (252 lines) but uses **WRONG patterns** (gvproxy, wrong paths). Must refactor:
- `forge.go` - Build, Deploy functions (split lifecycle out)
- `lifecycle.go` - Start, Stop, Remove functions  
- `verify.go` - Test verification

**Key fixes needed:**
- Change device path from `/data/sovereign/forgejo/` to `/data/sovereign/vm/forgejo/`
- Remove gvproxy references from Deploy
- Add TAP networking setup to Start

---

## 4. Reusable Components to Extract

> **NOTE (TEAM_024 Review):** This is a **FUTURE OPTIMIZATION**. Get Forgejo working first,
> then extract common patterns. Do NOT implement this in the initial pass.

### 4.1 Create `internal/vm/common/` Package

Extract these shared patterns:

```go
// common/networking.go
func SetupTAPNetwork(tapName, hostIP, guestIP string) error
func TeardownTAPNetwork(tapName string) error

// common/tailscale.go
func RemoveTailscaleRegistrations(hostname string) error

// common/init.go
func GenerateInitScript(config InitConfig) string
type InitConfig struct {
    Hostname    string
    TAPGuestIP  string
    Service     string // "postgresql" or "forgejo"
    Ports       []PortMapping
}

// common/datadisk.go
func CreateDataDisk(path string, sizeGB int) error
func MountDataDiskInGuest() string // Returns shell script fragment
```

### 4.2 Shared Init Script Template

Create `vm/common/init.sh.tmpl`:
```bash
#!/bin/sh
# Common AVF init template
# Service: {{.Service}}
# Hostname: {{.Hostname}}

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# --- COMMON INIT (same for all services) ---
mount -t proc proc /proc
mount -t sysfs sysfs /sys
mount -t devtmpfs devtmpfs /dev
mkdir -p /dev/shm /tmp
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /tmp

# Mount persistent data disk
mkdir -p /data
mount /dev/vdb /data || mount -t tmpfs tmpfs /data

# Set time
date -s "{{.CurrentDate}}" 2>/dev/null

# --- NETWORKING ---
ip addr add {{.TAPGuestIP}}/24 dev eth0
ip link set eth0 up
ip route add default via {{.TAPGateway}}
echo "nameserver 8.8.8.8" > /etc/resolv.conf

# --- TAILSCALE (persistent identity) ---
mkdir -p /data/tailscale /var/run/tailscale
/usr/sbin/tailscaled --tun=userspace-networking \
    --state=/data/tailscale/tailscaled.state \
    --socket=/var/run/tailscale/tailscaled.sock &

if [ -f /data/tailscale/tailscaled.state ]; then
    /usr/bin/tailscale up --hostname={{.Hostname}}
else
    /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname={{.Hostname}}
fi

{{range .Ports}}
/usr/bin/tailscale serve --bg --tcp {{.External}} {{.Internal}}
{{end}}

# --- SERVICE-SPECIFIC ---
{{.ServiceInit}}
```

---

## 5. Network Subnet Allocation

| Service | TAP Interface | Host IP | Guest IP | Tailscale Hostname |
|---------|---------------|---------|----------|-------------------|
| PostgreSQL | vm_sql | 192.168.100.1 | 192.168.100.2 | sovereign-sql |
| Forgejo | vm_forge | 192.168.101.1 | 192.168.101.2 | sovereign-forge |
| Vaultwarden | vm_vault | 192.168.102.1 | 192.168.102.2 | sovereign-vault |
| (Future) | vm_XXX | 192.168.10X.1 | 192.168.10X.2 | sovereign-XXX |

---

## 6. BDD Test Requirements for Forgejo

The next team must write comprehensive BDD tests **BEFORE** implementing. This is the order:

### 6.1 Re-enable and Expand `forge.feature`

```bash
mv features/forge.feature.disabled features/forge.feature
```

### 6.2 Scenarios to Add (Modeled After sql.feature)

See Section 7 for complete feature file template.

---

## 7. Complete Forgejo BDD Test Template

```gherkin
Feature: Forge VM Lifecycle
  As a developer
  I want to manage the Forgejo VM
  So that I can run git hosting on my Android device

  # ==========================================================================
  # BUILD BEHAVIORS
  # ==========================================================================

  @build
  Scenario: Build Forge VM with Docker
    Given Docker is available
    And the shared kernel Image exists at "vm/sql/Image"
    When I build the Forge VM
    Then the Docker image "sovereign-forge" should exist
    And the rootfs should be created at "vm/forgejo/rootfs.img"
    And the data disk should be created at "vm/forgejo/data.img"

  @build @dockerfile
  Scenario: Dockerfile uses verified Alpine version
    Given Docker is available
    When I build the Forge VM
    Then the Dockerfile should use Alpine version "3.21"

  @build @dockerfile @forgejo
  Scenario: Dockerfile installs Forgejo from community repo
    Given Docker is available
    When I build the Forge VM
    Then the Dockerfile should enable Alpine community repository
    And the Dockerfile should install "forgejo" package
    And the Dockerfile should install "git" package
    And the Dockerfile should install "openssh-server" package

  @build @dockerfile @tailscale
  Scenario: Dockerfile installs Tailscale static binary
    Given Docker is available
    When I build the Forge VM
    Then the Dockerfile should download Tailscale version "1.92.3"
    And the Tailscale binaries should be at "/usr/bin/tailscale" and "/usr/sbin/tailscaled"

  @build @rootfs @init
  Scenario: Rootfs preparation creates init script
    Given the Docker build completed
    When the rootfs is prepared for AVF
    Then "/sbin/init.sh" should be created from "vm/forgejo/init.sh"
    And "/sbin/init.sh" should be executable
    And OpenRC should NOT be the init system

  @build @error
  Scenario: Build fails without shared kernel
    Given Docker is available
    But the shared kernel does not exist at "vm/sql/Image"
    When I try to build the Forge VM
    Then the build should fail with error containing "Build SQL VM first"

  # ==========================================================================
  # DEPLOY BEHAVIORS
  # ==========================================================================

  @deploy
  Scenario: Deploy Forge VM to device
    Given a device is connected
    And the Forge VM is built
    When I deploy the Forge VM
    Then the VM directory should exist on device at "/data/sovereign/vm/forgejo"
    And the start script should exist at "/data/sovereign/vm/forgejo/start.sh"
    And the rootfs should exist at "/data/sovereign/vm/forgejo/rootfs.img"
    And the data disk should exist at "/data/sovereign/vm/forgejo/data.img"

  @deploy @networking
  Scenario: Deploy creates TAP networking script
    Given a device is connected
    And the Forge VM is built
    When I deploy the Forge VM
    Then the start script should configure TAP interface "vm_forge"
    And the start script should use host IP "192.168.101.1"
    And the start script should NOT use gvproxy or vsock

  # ==========================================================================
  # START BEHAVIORS
  # ==========================================================================

  @start
  Scenario: Start Forge VM
    Given a device is connected
    And the Forge VM is deployed
    When I start the Forge VM
    Then the VM process should be running
    And the TAP interface "vm_forge" should be UP
    And the console output should show "INIT START"

  @start @networking
  Scenario: Start configures guest networking
    Given a device is connected
    And the Forge VM is deployed
    When I start the Forge VM
    Then the guest should have IP "192.168.101.2"
    And the guest should be able to ping "8.8.8.8"

  @start @tailscale
  Scenario: Start registers Tailscale with persistent identity
    Given a device is connected
    And the Forge VM is deployed
    And a .env file exists with TAILSCALE_AUTHKEY
    When I start the Forge VM
    Then Tailscale should connect as "sovereign-forge"
    And Tailscale state should be stored on persistent disk

  @start @tailscale @idempotent
  Scenario: Restart does not create duplicate Tailscale registration
    Given a device is connected
    And the Forge VM is running
    And Tailscale is connected as "sovereign-forge"
    When I stop the Forge VM
    And I start the Forge VM
    Then only one "sovereign-forge" registration should exist
    And the Tailscale IP should be the same as before

  @start @database
  Scenario: Start waits for PostgreSQL database
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is deployed
    When I start the Forge VM
    Then the init should wait for PostgreSQL on port 5432
    And the init should show "PostgreSQL is ready"

  @start @forgejo
  Scenario: Start launches Forgejo web service
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is deployed
    When I start the Forge VM
    Then Forgejo should be listening on port 3000
    And Forgejo should be exposed via Tailscale

  @start @idempotent
  Scenario: Start is idempotent when VM already running
    Given a device is connected
    And the Forge VM is running
    When I start the Forge VM
    Then the command should succeed
    And I should see "VM already running"
    And no new process should be started

  # ==========================================================================
  # STOP BEHAVIORS
  # ==========================================================================

  @stop
  Scenario: Stop running Forge VM
    Given a device is connected
    And the Forge VM is running
    When I stop the Forge VM
    Then the VM process should not be running
    And the TAP interface "vm_forge" should be removed
    And the socket file should be removed
    And the pid file should be removed
    And I should see "VM stopped"

  @stop @idempotent
  Scenario: Stop is idempotent when VM not running
    Given a device is connected
    And the Forge VM is not running
    When I stop the Forge VM
    Then the command should succeed
    And I should see "VM not running"

  # ==========================================================================
  # TEST BEHAVIORS
  # ==========================================================================

  @test
  Scenario: Test healthy Forge VM
    Given a device is connected
    And the Forge VM is running
    And Forgejo web UI is responding on port 3000
    When I test the Forge VM
    Then all tests should pass

  @test @tailscale
  Scenario: Test Tailscale connectivity
    Given a device is connected
    And the Forge VM is running
    When I test the Forge VM
    Then Tailscale should show "sovereign-forge" as connected
    And port 3000 should be accessible via Tailscale
    And port 22 should be accessible via Tailscale

  @test @database
  Scenario: Test database connectivity
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is running
    When I test the Forge VM
    Then Forgejo should be connected to PostgreSQL

  # ==========================================================================
  # REMOVE BEHAVIORS
  # ==========================================================================

  @remove
  Scenario: Remove Forge VM
    Given a device is connected
    And the Forge VM is deployed
    When I remove the Forge VM
    Then the VM directory should not exist on device
    And the TAP interface should be removed
    And the Tailscale registration should be removed
    And I should see "Forge VM removed"

  @remove @idempotent
  Scenario: Remove is idempotent when VM not deployed
    Given a device is connected
    And the Forge VM is not deployed
    When I remove the Forge VM
    Then the command should succeed

  # ==========================================================================
  # MULTI-VM SCENARIOS
  # ==========================================================================

  @multi-vm
  Scenario: Forge and SQL VMs run simultaneously
    Given a device is connected
    And the SQL VM is running
    And the Forge VM is running
    Then both VMs should have separate TAP interfaces
    And both VMs should have separate Tailscale IPs
    And Forge should connect to SQL via TAP network

  @multi-vm @restart
  Scenario: Forge reconnects to SQL after SQL restart
    Given a device is connected
    And both SQL and Forge VMs are running
    When I restart the SQL VM
    Then Forge should reconnect to PostgreSQL
    And Forgejo should continue working
```

---

## 8. Implementation Checklist for Next Team

### Phase 1: BDD Tests (Do This First!)
- [ ] Rename `forge.feature.disabled` to `forge.feature`
- [ ] Add all scenarios from Section 7
- [ ] Implement step definitions in `sovereign_test.go`
- [ ] Run tests - they should all FAIL (this is expected)

### Phase 2: Init Script
- [ ] Create `vm/forgejo/init.sh` modeled after `vm/sql/init.sh`
- [ ] Implement persistent data disk mounting
- [ ] Implement persistent Tailscale identity
- [ ] Add PostgreSQL wait logic
- [ ] Add Forgejo startup

### Phase 3: Start Script
- [ ] Rewrite `vm/forgejo/start.sh` to use TAP networking
- [ ] Remove gvproxy/vsock code
- [ ] Use `console=ttyS0` not `console=hvc0`
- [ ] Pass data.img as second block device

### Phase 4: Go Code
- [ ] Refactor `internal/vm/forge/forge.go` (exists but wrong patterns)
- [ ] **CRITICAL:** Fix device path to `/data/sovereign/vm/forgejo/` (currently wrong!)
- [ ] Split out `internal/vm/forge/lifecycle.go`
- [ ] Implement `internal/vm/forge/verify.go`
- [ ] Use persistent Tailscale identity pattern

### Phase 5: Dockerfile
- [ ] Update to use static Tailscale binary
- [ ] Remove OpenRC as init system
- [ ] Ensure init.sh is installed to /sbin/init.sh

### Phase 6: Integration
- [ ] Test Forge + SQL running together
- [ ] Verify Forge connects to SQL via Tailscale
- [ ] Run full BDD test suite
- [ ] All tests must pass

---

## 9. Critical Lessons Learned (Don't Repeat These Mistakes)

### 9.1 OpenRC Hangs in AVF
**Problem:** OpenRC hangs during sysinit in crosvm.
**Solution:** Use custom init.sh script, NOT `/sbin/init`.

### 9.2 Console Device Mismatch
**Problem:** Using `console=hvc0` but crosvm `--serial` captures ttyS0.
**Solution:** Use `console=ttyS0` with `--serial type=stdout`.

### 9.3 Tailscale Duplicate Registrations
**Problem:** Every restart creates a new Tailscale machine.
**Solution:** Persist Tailscale state on data.img, not rootfs.img.

### 9.4 Android Policy Routing Blocks NAT
**Problem:** NAT works but return traffic doesn't reach VM.
**Solution:** `ip rule add from all lookup main pref 1` on Android host.

### 9.5 gvproxy/vsock Doesn't Work
**Problem:** gvproxy approach for networking doesn't work in AVF.
**Solution:** Use TAP networking with NAT.

---

## 10. Files to Reference

| File | Purpose |
|------|---------|
| `vm/sql/init.sh` | **THE GOLD STANDARD** - copy this pattern |
| `vm/sql/start.sh` | TAP networking setup |
| `internal/vm/sql/lifecycle.go` | Start/Stop/Remove with persistent Tailscale |
| `features/sql.feature` | BDD test patterns |
| `.teams/TEAM_019_tailscale_idempotency.md` | Tailscale duplicate fix documentation |
| `docs/AVF_VM_NETWORKING.md` | Networking deep dive |

---

## 11. Questions to Answer Before Starting

> **TEAM_024 Review:** These questions have been moved to `.questions/TEAM_024_forgejo_decisions.md`
> and answered. Summary below:

1. **DB connection method:** Use Tailscale (`sovereign-sql`) for reliability ✓
2. **Ports to expose:** 3000 (web) + 22 (SSH) ✓
3. **Shared kernel:** Defer - current approach works ✓

**Also noted:** `app.ini` currently uses `sql-vm:5432` - needs update to `sovereign-sql:5432`

---

## 12. Success Criteria

The Forgejo implementation is complete when:

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

## Document History

| Date | Team | Change |
|------|------|--------|
| 2025-12-29 | TEAM_023 | Created initial plan based on working PostgreSQL implementation |
| 2025-12-29 | TEAM_024 | Reviewed plan; fixed Section 3.4 (Go code exists), marked Section 4 as future, moved questions to `.questions/`, identified critical path issues |
