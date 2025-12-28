# Sovereign Forgejo VM - Implementation Checklist

**Status:** NOT STARTED
**Purpose:** Self-hosted Git forge with CI/CD

---

## ⚠️ MANDATORY READING BEFORE ANY WORK ⚠️

**Future AI Teams: READ THESE FILES IN THIS ORDER:**

1. **`sovereign/sovereign_vault.md`** - Section 0: AI Failure Modes
   - Cost: €0.21/message - shortcuts are THEFT
   - Do NOT downgrade, do NOT take shortcuts
   
2. **`sovereign/vm/sql/CHECKLIST.md`** - 20+ documented mistakes
   - Microdroid ≠ micro Linux for VMs
   - Use sovereign CLI, not raw commands
   
3. **`sovereign/docs/AVF_VM_NETWORKING.md`** - Networking knowledge

4. **This file** - Forgejo-specific requirements

**DO NOT START WORK UNTIL YOU'VE READ ALL FOUR.**

---

## Key Lessons from SQL VM (DO NOT REPEAT THESE MISTAKES)

| # | Mistake | Impact | Correct Approach |
|---|---------|--------|------------------|
| 1 | Used `microdroid_defconfig` | Missing SYSVIPC, netfilter | Use `sovereign_guest.fragment` |
| 2 | Bypassed sovereign CLI | Non-reproducible work | Always use `sovereign build --forge` |
| 3 | x86 Docker without `--platform` | Wrong architecture | Always `--platform linux/arm64` |
| 4 | Used `console=ttyS0` | No console output | Use `console=hvc0` |
| 5 | Didn't create `/dev/vsock` | gvforwarder fails | Created in `rootfs.PrepareForAVF()` |
| 6 | Didn't create `/dev/net/tun` | TAP interface fails | Created in `rootfs.PrepareForAVF()` |
| 7 | Missing dhclient wrapper | gvforwarder DHCP fails | Created in `rootfs.PrepareForAVF()` |
| 8 | OpenRC hangs | Services don't start | Use `simple_init` or debug OpenRC |
| 9 | PATH not set in init | Binaries not found | `export PATH=...` at top of init |

---

## Forgejo-Specific Requirements

### Application Overview
- **Forgejo** = Community fork of Gitea (Git forge)
- **Purpose:** Self-hosted Git repositories with web UI, CI/CD
- **Dependencies:** PostgreSQL (uses SQL VM), SSH access

### Alpine Packages Required
```dockerfile
FROM --platform=linux/arm64 alpine:3.19
RUN apk add --no-cache \
    forgejo \
    git \
    openssh-server \
    tailscale \
    openrc
```

### Port Requirements
| Port | Service | Protocol |
|------|---------|----------|
| 3000 | Forgejo web UI | HTTP |
| 22 | SSH for git push/pull | SSH |

### PostgreSQL Connection
Forgejo connects to the SQL VM for its database:
```ini
[database]
DB_TYPE = postgres
HOST = sql-vm:5432  # Tailscale DNS name
NAME = forgejo
USER = forgejo
PASSWD = forgejo
```

**CRITICAL:** The SQL VM must be running and have the `forgejo` database created.

---

## Build Process

### Step 1: Ensure SQL VM is working first
```bash
sovereign start --sql
sovereign test --sql  # Must pass before building Forgejo
```

### Step 2: Build Forgejo VM
```bash
sovereign build --forge
```

This should:
1. Build Docker image with `--platform linux/arm64`
2. Export rootfs as ext4 image
3. Run `rootfs.PrepareForAVF()` to fix:
   - Device nodes (/dev/vsock, /dev/net/tun)
   - simple_init script
   - dhclient wrapper

### Step 3: Deploy
```bash
sovereign deploy --forge
```

### Step 4: Start
```bash
sovereign start --forge
```

### Step 5: Test
```bash
sovereign test --forge
```

---

## Kernel Requirements

The Forgejo VM uses the SAME guest kernel as SQL VM.

**Required configs (already in `sovereign_guest.fragment`):**
- `CONFIG_SYSVIPC=y` - Not needed for Forgejo directly, but shared kernel
- `CONFIG_NETFILTER=y` - For Tailscale kernel mode
- `CONFIG_VIRTIO_*=y` - All virtio drivers
- `CONFIG_TUN=y` - Tailscale tunnel

---

## Networking Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Tailscale Network                        │
│                                                             │
│  ┌──────────┐     ┌──────────┐     ┌──────────┐            │
│  │ sql-vm   │────▶│forge-vm  │────▶│ tanzanite │           │
│  │ :5432    │     │ :3000    │     │ (laptop)  │           │
│  │ postgres │     │ :22      │     │           │           │
│  └──────────┘     └──────────┘     └──────────┘            │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Tailscale Serve (optional)
To expose Forgejo on HTTPS:
```bash
# Inside forge-vm
tailscale serve --bg 3000
```

This creates: `https://forge-vm.tail5bea38.ts.net/`

---

## Configuration Files

### /etc/forgejo/app.ini
```ini
[server]
DOMAIN = forge-vm.tail5bea38.ts.net
ROOT_URL = https://forge-vm.tail5bea38.ts.net/
HTTP_PORT = 3000
SSH_DOMAIN = forge-vm.tail5bea38.ts.net
SSH_PORT = 22

[database]
DB_TYPE = postgres
HOST = sql-vm:5432
NAME = forgejo
USER = forgejo
PASSWD = forgejo

[repository]
ROOT = /data/forgejo/repositories

[log]
ROOT_PATH = /var/log/forgejo
```

### SSH Configuration
```bash
# Enable SSH server for git operations
rc-update add sshd default
```

---

## Init Script Requirements

The init script must:
1. Start gvforwarder (for networking)
2. Configure tap0 network interface
3. Start Tailscale
4. Wait for SQL VM to be reachable
5. Start Forgejo service
6. Start SSH server

```sh
#!/bin/sh
# /sbin/simple_init for Forgejo VM
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# ... standard init (mount proc/sys, device nodes, gvforwarder) ...

# Wait for SQL VM
echo "Waiting for sql-vm..."
until nc -z sql-vm 5432 2>/dev/null; do
    sleep 2
done
echo "sql-vm reachable"

# Start Forgejo
forgejo web &

# Start SSH
/usr/sbin/sshd
```

---

## Testing Checklist

After `sovereign start --forge`:

- [ ] VM boots (check via crosvm console or logs)
- [ ] Tailscale connected (`tailscale status` shows forge-vm)
- [ ] Can reach SQL VM (`nc -z sql-vm 5432`)
- [ ] Forgejo web UI accessible at http://forge-vm:3000
- [ ] SSH works (`ssh git@forge-vm`)
- [ ] Can clone/push repositories

---

## Common Issues

### Issue: "connection refused" to sql-vm:5432
**Cause:** SQL VM not running or PostgreSQL not started
**Fix:** `sovereign start --sql && sovereign test --sql`

### Issue: Forgejo won't start
**Cause:** Database not initialized
**Fix:** First-run setup creates schema. Check `/var/log/forgejo/forgejo.log`

### Issue: SSH not working
**Cause:** sshd not started or host keys not generated
**Fix:** 
```bash
ssh-keygen -A  # Generate host keys
/usr/sbin/sshd
```

### Issue: "permission denied" on git push
**Cause:** SSH keys not configured
**Fix:** Add SSH public key to Forgejo user settings

---

## Files to Create

```
sovereign/vm/forgejo/
├── CHECKLIST.md          # This file
├── Dockerfile            # Alpine + Forgejo + deps
├── scripts/
│   └── init.sh           # OpenRC init script
├── config/
│   └── app.ini           # Forgejo configuration
├── start.sh              # Host-side VM launcher
└── vm-config.json        # VM resource allocation
```

---

## Implementation Order

1. **Create Dockerfile** - Base Alpine with Forgejo, SSH, Tailscale
2. **Create init script** - Start services in correct order
3. **Create app.ini** - Configure Forgejo with SQL VM connection
4. **Add to sovereign CLI** - Register "forge" VM type
5. **Test build** - `sovereign build --forge`
6. **Test deploy/start** - Full integration test
7. **Document** - Update this checklist with findings

---

## CLI Integration

The sovereign CLI needs a `forge` VM type registered.

In `sovereign/internal/vm/forge/forge.go`:
```go
package forge

import "github.com/anthropics/sovereign/internal/vm"

func init() {
    vm.Register("forge", &ForgeVM{})
}

type ForgeVM struct{}

func (v *ForgeVM) Build() error { ... }
func (v *ForgeVM) Deploy() error { ... }
func (v *ForgeVM) Start() error { ... }
func (v *ForgeVM) Stop() error { ... }
func (v *ForgeVM) Test() error { ... }
```

---

## Success Criteria

The Forgejo VM is complete when:
- [ ] `sovereign build --forge` succeeds
- [ ] `sovereign deploy --forge` pushes to device
- [ ] `sovereign start --forge` launches VM
- [ ] `sovereign test --forge` passes all checks
- [ ] Web UI accessible via Tailscale
- [ ] Can create repository and push code
- [ ] CI/CD runners (optional, Phase 2)

---

## Team Attribution

When modifying code, use:
```
// TEAM_XXX: <description>
```

This file created by TEAM_012 as documentation for future teams.
