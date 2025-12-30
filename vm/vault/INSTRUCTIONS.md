# Vaultwarden VM Implementation Instructions

> **For the next team implementing Vaultwarden**
> 
> TEAM_034 has prepared this environment. Read this BEFORE starting.

## 1. What's Already Done ✅

| Component | Status | File |
|-----------|--------|------|
| Directory structure | ✅ | `vm/vault/` |
| Dockerfile | ✅ | `vm/vault/Dockerfile` |
| init.sh | ✅ | `vm/vault/init.sh` |
| Config template | ✅ | `vm/vault/config/env.template` |
| Go VM config | ✅ | `internal/vm/vault/vault.go` |
| Lifecycle methods | ✅ | `internal/vm/vault/lifecycle.go` |
| Test methods | ✅ | `internal/vm/vault/verify.go` |
| CLI --vault flag | ✅ | `cmd/sovereign/main.go` |

## 2. What You Need To Do

### 2.1 Database Creation (AUTOMATIC) ✅

**TEAM_035:** Database is created automatically by SQL VM on startup.

The SQL VM init.sh (`vm/sql/init.sh:257-262`) creates:
- User: `vaultwarden`
- Database: `vaultwarden`
- Password: See `vm/vault/CREDENTIALS.md`

No manual steps required - just ensure SQL VM starts before Vault VM.

### 2.2 Build and Deploy

```bash
# Build
./sovereign build --vault

# Deploy (first time, use --fresh-data)
./sovereign deploy --vault --fresh-data

# Start (requires SQL VM running first)
./sovereign start --sql   # If not already running
./sovereign start --vault

# Test
./sovereign test --vault
```

### 2.4 Verify It Works

1. Open browser: `https://sovereign-vault.tail5bea38.ts.net` (or whatever hostname Tailscale assigns)
2. Create account
3. Test saving/retrieving passwords
4. Test browser extension
5. **IMPORTANT:** Set `SIGNUPS_ALLOWED=false` after creating your account!

## 3. Key Patterns Already Implemented

### 3.1 Dynamic TLS Hostname (init.sh:127-147)

```bash
TS_FQDN=$(/usr/bin/tailscale status --json | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')
/usr/bin/tailscale cert --cert-file=/data/vault/tls/cert.pem --key-file=/data/vault/tls/key.pem "$TS_FQDN"
```

This handles `sovereign-vault-1`, `sovereign-vault-2`, etc.

### 3.2 Port 443 Binding (init.sh:171)

```bash
echo 443 > /proc/sys/net/ipv4/ip_unprivileged_port_start
```

Allows non-root user to bind port 443.

### 3.3 PostgreSQL Wait (init.sh:152-169)

Already implemented - waits up to 30s for PostgreSQL.

## 4. Potential Issues

### 4.1 Vaultwarden Binary Download

The Dockerfile downloads from GitHub releases. Check if version exists:
- Current: `VAULTWARDEN_VERSION=1.35.0` (TEAM_035: Updated from 1.32.5)
- Verify: https://github.com/dani-garcia/vaultwarden/releases

If download fails, the URL format may have changed. Check release assets.

### 4.2 Web Vault Version

Web vault is downloaded separately:
- Current: `v2024.6.2c`
- Verify: https://github.com/dani-garcia/bw_web_builds/releases

### 4.3 TLS Cert Generation Fails

If you see "could not determine Tailscale FQDN":
1. Check Tailscale is running: `tailscale status`
2. Check hostname was assigned
3. May need to wait a few seconds after `tailscale up`

### 4.4 Database Connection Fails

If PostgreSQL errors:
1. Verify SQL VM is running: `./sovereign start --sql`
2. Check database exists: `psql -h 192.168.100.2 -U vaultwarden -d vaultwarden`
3. Verify password matches

## 5. Testing Checklist

- [ ] `./sovereign build --vault` succeeds
- [ ] `./sovereign deploy --vault` succeeds
- [ ] `./sovereign start --vault` succeeds (SQL VM must be running)
- [ ] Tailscale shows `sovereign-vault` (or `-1`, `-2`)
- [ ] TLS cert generated (check console output)
- [ ] `https://sovereign-vault.tail5bea38.ts.net` loads
- [ ] Can create account
- [ ] Can save password
- [ ] Can retrieve password
- [ ] Browser extension works
- [ ] Mobile app works
- [ ] Survives VM restart
- [ ] `SIGNUPS_ALLOWED=false` set after setup

## 6. Files Reference

```
vm/vault/
├── Dockerfile              # Container build
├── INSTRUCTIONS.md         # This file
├── init.sh                 # VM init script (TLS, network, startup)
└── config/
    └── env.template        # Environment variable reference

internal/vm/vault/
├── vault.go                # VMConfig and registration
├── lifecycle.go            # Start/Stop/Remove
└── verify.go               # Test methods
```

## 7. Documentation

- `docs/VAULTWARDEN_IMPLEMENTATION_GUIDE.md` - Comprehensive guide
- `docs/AVF_VM_NETWORKING.md` - Networking architecture
- `.teams/TEAM_034_verify_tailscale_solution.md` - TLS/port 443 discoveries

## 8. Support

If stuck, check:
1. Console output: `adb shell su -c 'cat /data/sovereign/vm/vault/console.log'`
2. Tailscale status: `tailscale status`
3. PostgreSQL: `nc -zv 192.168.100.2 5432`

Good luck! 🔐
