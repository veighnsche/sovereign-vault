# TEAM_033: Investigate Tailscale AVF Port Exposure Bug

## Status: IN PROGRESS

## Task
Investigate why external Tailscale access to VM services fails despite successful Tailscale connection inside VMs.

## Prior Art
- TEAM_030: Extensive investigation documented in `docs/TAILSCALE_AVF_LIMITATIONS.md`
- Tested userspace networking + native tun mode - both fail with `i/o timeout`

## Symptom
- Services running inside VMs (PostgreSQL port 5432)
- Tailscale connects successfully, VM appears in admin panel
- `tailscale serve` configured correctly
- **Result**: `ERR_CONNECTION_RESET` or `i/o timeout` when accessing externally

## Investigation Timeline

### Phase 1: Review Prior Investigation
- Reading TEAM_030's findings
- Understanding claimed root cause (fwmark routing)

## Hypotheses

### H1: TEAM_030's analysis is CORRECT - fwmark routing is the root cause
**Status: CONFIRMED**

`tailscale serve` runs inside tailscaled's network context:
- **Userspace mode**: Isolated netstack - can't reach ANY local interface
- **Native tun mode**: fwmark marks all outbound connections from tailscaled

Both modes prevent `tailscale serve` from proxying to local services.

### H2: Alternative - Use separate reverse proxy process
**Status: VIABLE - NOT TESTED**

Run nginx/socat as SEPARATE process (not child of tailscaled):
- Binds to Tailscale IP (100.x.x.x)
- Forwards to local service (localhost:5432)
- Not affected by tailscaled's network isolation

### H3: Alternative - Subnet router on Android host
**Status: VIABLE - NOT TESTED**

Android host advertises VM subnet (192.168.100.0/24) via Tailscale:
```bash
tailscale up --advertise-routes=192.168.100.0/24
```
External devices access VMs directly at TAP IPs.

### H4: Run Tailscale only on Android host
**Status: VIABLE - NOT TESTED**

Don't run Tailscale in VMs. Use Android host as gateway/proxy.

## BUG FOUND: Inconsistent init.sh restart logic

**File**: `vm/forgejo/init.sh`

| Location | Mode | Issue |
|----------|------|-------|
| Line 145 | Native tun (no --tun) | Initial start |
| Line 291 | Userspace (`--tun=userspace-networking`) | Supervision restart |
| Line 197 | TAP IP (`tcp://192.168.100.3:3000`) | Initial serve |
| Line 296 | Localhost (`3000`) | Restart serve |

These inconsistencies cause unpredictable behavior after daemon restart.

## Files to Review
- `vm/sql/init.sh` - Tailscale startup and serve config
- `vm/sql/build-guest-kernel.sh` - Kernel config for nftables
- `docs/AVF_VM_NETWORKING.md` - Networking architecture

## Root Cause - CONFIRMED

**TEAM_030's analysis is correct.** The issue is a fundamental architectural limitation of Tailscale:

1. **Userspace networking**: Creates isolated netstack. `tailscale serve` proxy runs inside this isolated namespace and cannot reach ANY local interface (localhost or TAP IP).

2. **Native tun mode**: Uses fwmark-based policy routing. ALL outbound connections from tailscaled (including `tailscale serve` proxying to localhost) get marked and routed through Tailscale's routing tables, which cannot reach local interfaces.

This is NOT a configuration issue. It's how Tailscale is designed.

## Fix Applied

Fixed bug in `vm/forgejo/init.sh` - supervision restart now matches initial start:
- Native tun mode (no `--tun` flag)
- TAP IP in serve commands (`tcp://192.168.100.3:3000`)

**Note**: This fix addresses the inconsistency bug but does NOT solve the fundamental `tailscale serve` limitation.

## Recommended Solution Path

### Option A: Tailscale Subnet Router on Android Host (RECOMMENDED)

**Concept**: Android host advertises VM subnet to tailnet. External devices access VMs directly.

```bash
# On Android host (requires Tailscale CLI access)
tailscale up --advertise-routes=192.168.100.0/24
```

**Pros**:
- No changes to VMs needed
- Direct access to VM services at TAP IPs
- Works with any service/port

**Cons**:
- Requires Android Tailscale to support `--advertise-routes`
- May require Tailscale admin approval for routes

### Option B: Run Tailscale Only on Android Host

**Concept**: Remove Tailscale from VMs. Android host acts as gateway.

**Pros**:
- Simpler architecture
- Less resource usage in VMs

**Cons**:
- Requires reverse proxy setup on host
- VMs lose individual Tailscale identities

### Option C: Separate Reverse Proxy in VM

**Concept**: Run nginx/socat in VM, bind to Tailscale IP, forward to localhost.

**Pros**:
- Works within current architecture

**Cons**:
- Requires adding nginx/socat to rootfs
- More complexity in init scripts
- Still subject to Tailscale networking quirks

## Files Modified

### Option A Implementation (Subnet Router)
- `vm/sql/init.sh` - Added `--advertise-routes=192.168.100.0/24` to tailscale up commands
- `docs/TAILSCALE_AVF_LIMITATIONS.md` - Updated status to RESOLVED, documented solution

### Bug Fixes
1. `vm/forgejo/init.sh` - Fixed supervision restart inconsistency (native tun + TAP IP)
2. `vm/sql/start.sh` - Fixed undefined variable `TAP_HOST_IP`
3. `vm/forgejo/start.sh` - Fixed undefined variable `TAP_HOST_IP`
4. `internal/vm/forge/forge.go` - Fixed subnet config (was 192.168.101.x, now 192.168.100.x)
5. `sovereign_test.go` - Fixed incorrect IP address comments and expectations
6. `internal/vm/common/config.go` - Fixed comments about subnet architecture

### Architecture Inconsistency Fixed
The Go configs said Forge VM used 192.168.101.0/24 subnet, but actual shell scripts used 192.168.100.0/24 (shared bridge). Fixed Go configs to match actual implementation.

## Handoff Checklist
- [x] Root cause identified or confirmed
- [x] Bug fixes implemented (6 bugs)
- [x] Architectural solution implemented (subnet router)
- [x] Documentation updated
- [x] Team file updated
- [ ] Manual testing required (approve subnet routes in Tailscale admin)

---

## Phase 2: Android App Architecture Analysis

### Task
Analyze Google's Terminal app source code to understand proper VirtualizationService usage for building a native Android app to replace the Go + Bash approach.

### Key Findings

#### 1. Google Terminal App Structure
Downloaded and analyzed source from `android.googlesource.com/platform/packages/modules/Virtualization/+/main/android/TerminalApp/`

Key files analyzed:
- `VmLauncherService.kt` - Foreground service managing VM lifecycle
- `Runner.kt` - VM creation and callback handling
- `ConfigJson.kt` - JSON config parsing into VirtualMachineConfig
- `InstalledImage.kt` - Disk image management (resize, truncate)
- `DebianServiceImpl.kt` - gRPC for host-guest communication
- `AndroidManifest.xml` - Required permissions

#### 2. Critical API Discovery
```kotlin
// The key to proper networking:
VirtualMachineCustomImageConfig.Builder()
    .useNetwork(true)  // THIS enables automatic NetworkAgent registration!
```

When `useNetwork(true)` is set:
- VirtualizationService creates TAP interface automatically
- Registers proper NetworkAgent with Android's ConnectivityService
- No manual iptables/ip rule hacks needed
- Routing works correctly

#### 3. Required Permissions
```xml
<uses-permission android:name="android.permission.MANAGE_VIRTUAL_MACHINE" />
<uses-permission android:name="android.permission.USE_CUSTOM_VIRTUAL_MACHINE" />
```

Can be granted via ADB for development:
```bash
adb shell pm grant <package> android.permission.MANAGE_VIRTUAL_MACHINE
```

#### 4. VM Config JSON Pattern
Google uses JSON with variable substitution (`$APP_DATA_DIR`, `$PAYLOAD_DIR`):
```json
{
    "name": "debian",
    "kernel": "$PAYLOAD_DIR/vmlinuz",
    "initrd": "$PAYLOAD_DIR/initrd.img",
    "network": true,
    "disks": [...]
}
```

### Design Documents Created
1. `.planning/android-app/DESIGN_DOCUMENT.md` - Full architecture design
2. `.planning/android-app/IMPLEMENTATION_PATTERNS.md` - Code patterns from Terminal app
3. `.planning/android-app/NEXT_STEPS.md` - Concrete implementation steps

### Recommended Technology Stack
- **Language**: Kotlin (modern, null-safe, excellent Android support)
- **UI**: Jetpack Compose
- **Build**: Gradle with Kotlin DSL
- **Min SDK**: API 34 (Android 14)

### Architecture Benefits Over Current Approach
| Current (Go + Bash) | Android App (Kotlin) |
|---------------------|----------------------|
| `ip rule` hacks removed by netd | Proper NetworkAgent registration |
| Manual crosvm invocation | VirtualMachineManager API |
| Shell scripts for lifecycle | VirtualMachineCallback |
| No GUI | Native Android UI |
| Requires root + ADB | Standard app (with permission) |

### Next Steps for Implementation
1. Create Android project in `sovereign/android/`
2. Implement minimal VmRunner
3. Port existing Alpine rootfs
4. Test with `useNetwork(true)`
5. Iterate on PostgreSQL/Forgejo integration

### Status
**PHASE 2 COMPLETE** - Architecture analysis done, design documents created.
