# Vaultwarden Implementation Guide

> **TEAM_034**: This document captures critical knowledge from the Forgejo implementation
> that directly applies to Vaultwarden. Read this BEFORE starting Vaultwarden work.

## 1. Critical Requirements

### 1.1 HTTPS is MANDATORY

**Vaultwarden will NOT work without valid HTTPS.** The WebCrypto API used for password
encryption requires a secure context. Unlike Forgejo where you can click through
certificate warnings, Vaultwarden simply won't function.

```
❌ http://vaultwarden:8080              → WebCrypto disabled, unusable
❌ https://100.x.x.x:443                → Cert mismatch, WebCrypto disabled
✅ https://sovereign-vault.tail5bea38.ts.net → Valid cert, works!
```

### 1.2 Port 443 (With sysctl Fix)

VMs run services as non-root users. By default, non-root cannot bind to ports < 1024.
**Solution:** Lower the unprivileged port start in init.sh (runs as root):

```bash
# In init.sh, before starting the service
echo 443 > /proc/sys/net/ipv4/ip_unprivileged_port_start
```

Then use port 443 directly:

```ini
# In app config
ROCKET_PORT=443
ROCKET_TLS={certs="/data/vault/tls/cert.pem",key="/data/vault/tls/key.pem"}
```

This gives you clean URLs without port numbers: `https://sovereign-vault.tail5bea38.ts.net`

> **TEAM_035 Note:** The implementation uses port 443 (not 8443 as mentioned elsewhere).
> This is the correct approach for clean URLs.

---

## 2. TLS Certificate Pattern

### 2.1 Dynamic Hostname Detection

Tailscale may assign `sovereign-vault-1`, `sovereign-vault-2`, etc. based on
existing registrations. The cert MUST match the actual assigned hostname.

**Pattern from Forgejo init.sh:**

```bash
# Get the ACTUAL Tailscale hostname
TS_FQDN=$(/usr/bin/tailscale status --json | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')
if [ -z "$TS_FQDN" ]; then
    # Fallback: get from tailscale cert error output
    TS_FQDN=$(/usr/bin/tailscale cert 2>&1 | grep -o '[a-z0-9-]*\.tail[a-z0-9]*\.ts\.net' | head -1)
fi
echo "Tailscale FQDN: $TS_FQDN"

# Generate cert for actual hostname
/usr/bin/tailscale cert \
    --cert-file=/data/vault/tls/cert.pem \
    --key-file=/data/vault/tls/key.pem \
    "$TS_FQDN"

# Save for config updates
echo "$TS_FQDN" > /data/vault/tls/fqdn.txt
```

### 2.2 Update Config at Runtime

Static configs baked into rootfs won't have the correct hostname.
Update configs at boot time:

```bash
if [ -f /data/vault/tls/fqdn.txt ]; then
    TS_FQDN=$(cat /data/vault/tls/fqdn.txt)
    # Update Vaultwarden's DOMAIN setting
    export DOMAIN="https://$TS_FQDN:8443"
fi
```

---

## 3. Database Connection

### 3.1 Use Internal Bridge Network

Vaultwarden connects to PostgreSQL over the internal bridge, NOT Tailscale:

```
Vaultwarden VM (192.168.100.4) → Android Bridge → PostgreSQL VM (192.168.100.2)
```

**Connection string:**
```
DATABASE_URL=postgresql://vaultwarden:PASSWORD@192.168.100.2:5432/vaultwarden
```

### 3.2 SSL for Internal DB: Not Required

Traffic on the bridge network:
- Never leaves the Android device
- Is isolated from external networks
- Only accessible by VMs on that bridge

SSL between VMs is optional (defense-in-depth only).

### 3.3 Wait for PostgreSQL

Vaultwarden may start before PostgreSQL is ready. Use the same pattern as Forgejo:

```bash
DB_HOST="192.168.100.2"
DB_PORT="5432"

if ! nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
    echo "Waiting for PostgreSQL..."
    for i in $(seq 1 30); do
        if nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
            echo "PostgreSQL ready"
            break
        fi
        [ $i -eq 30 ] && echo "FATAL: PostgreSQL unavailable" && exit 1
        sleep 1
    done
fi
```

---

## 4. VM Configuration

### 4.1 Suggested VMConfig

```go
VaultConfig = common.VMConfig{
    Name:         "vault",
    LocalPath:    "vm/vault",
    DevicePath:   "/data/sovereign/vm/vault",
    TAPInterface: "vm_vault",
    TAPIP:        "192.168.100.4",  // Next after Forgejo (192.168.100.3)
    GatewayIP:    "192.168.100.1",
    Memory:       512,              // Vaultwarden is lightweight
    CPUs:         1,
    SharedKernel: true,
    KernelSource: "vm/sql/Image",   // Reuse SQL kernel
    Dependencies: []string{"sql"},
}
```

### 4.2 Required Packages (Dockerfile)

```dockerfile
FROM alpine:3.23

RUN apk add --no-cache \
    curl \
    ca-certificates \
    iproute2 \
    netcat-openbsd

# Vaultwarden binary (download ARM64 release)
RUN curl -fsSL "https://github.com/dani-garcia/vaultwarden/releases/download/X.X.X/vaultwarden-linux-arm64" \
    -o /usr/bin/vaultwarden && chmod +x /usr/bin/vaultwarden

# Tailscale (same as other VMs)
RUN TAILSCALE_VERSION="1.92.3" && \
    curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_arm64.tgz" \
    -o /tmp/tailscale.tgz && \
    tar -xzf /tmp/tailscale.tgz -C /tmp && \
    cp /tmp/tailscale_*/tailscale /usr/bin/ && \
    cp /tmp/tailscale_*/tailscaled /usr/sbin/
```

---

## 5. Environment Variables

Vaultwarden uses environment variables for configuration:

```bash
# Required
export DATABASE_URL="postgresql://vaultwarden:PASSWORD@192.168.100.2:5432/vaultwarden"
export DOMAIN="https://${TS_FQDN}"
export ROCKET_PORT=443
export ROCKET_TLS='{certs="/data/vault/tls/cert.pem",key="/data/vault/tls/key.pem"}'

# Recommended
export SIGNUPS_ALLOWED=false          # After initial setup
export ADMIN_TOKEN="your-admin-token" # For admin panel
export LOG_LEVEL=info
export DATA_FOLDER=/data/vault/data
```

---

## 6. Init Script Template

Based on the working Forgejo pattern:

```bash
#!/bin/sh
# /sbin/init.sh for Vaultwarden VM

# 1. Basic system setup (same as SQL/Forgejo)
mount -t proc proc /proc
mount -t sysfs sys /sys
mount -t devtmpfs dev /dev
mkdir -p /dev/pts /dev/shm
mount -t devpts devpts /dev/pts
mount -t tmpfs tmpfs /dev/shm

# 2. Mount data disk
mount /dev/vdb /data

# 3. Network setup
ip link set eth0 up
ip addr add 192.168.100.4/24 dev eth0
ip route add default via 192.168.100.1

# 4. Time sync (required for TLS)
ntpd -n -q -p pool.ntp.org

# 5. Start Tailscale
/usr/sbin/tailscaled --state=/data/tailscale/tailscaled.state &
sleep 3
/usr/bin/tailscale up --hostname=sovereign-vault

# 6. Generate TLS cert (dynamic hostname!)
TS_FQDN=$(/usr/bin/tailscale status --json | grep -o '"DNSName":"[^"]*"' | head -1 | cut -d'"' -f4 | sed 's/\.$//')
mkdir -p /data/vault/tls
/usr/bin/tailscale cert --cert-file=/data/vault/tls/cert.pem --key-file=/data/vault/tls/key.pem "$TS_FQDN"

# 7. Wait for PostgreSQL
while ! nc -z 192.168.100.2 5432; do sleep 1; done

# 8. Start Vaultwarden
export DOMAIN="https://$TS_FQDN:8443"
export DATABASE_URL="postgresql://vaultwarden:PASSWORD@192.168.100.2:5432/vaultwarden"
export ROCKET_PORT=8443
export ROCKET_TLS='{certs="/data/vault/tls/cert.pem",key="/data/vault/tls/key.pem"}'
/usr/bin/vaultwarden &

# 9. Supervision loop
while true; do
    # Check and restart services if needed
    sleep 30
done
```

---

## 7. Testing Checklist

Before considering Vaultwarden complete:

- [ ] HTTPS works without certificate warnings
- [ ] Can access `https://sovereign-vault-X.tail5bea38.ts.net`
- [ ] Can create account and login
- [ ] Can save/retrieve passwords
- [ ] Browser extension connects successfully
- [ ] Mobile app connects successfully
- [ ] Survives VM restart (persistent data)
- [ ] Survives phone reboot

---

## 8. Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Hardcoded hostname in config | Use dynamic detection from `tailscale status --json` |
| Port 443 permission denied | Use sysctl fix (see 1.2) |
| Cert for wrong hostname | Generate cert AFTER Tailscale connects |
| WebCrypto disabled | Must use valid HTTPS with matching cert |
| DB connection refused | Wait for PostgreSQL before starting |
| INSTALL_LOCK equivalent | Set `SIGNUPS_ALLOWED=false` after setup |

---

## 9. Access URLs

After successful deployment:

```
Web Vault:  https://sovereign-vault-X.tail5bea38.ts.net
Admin:      https://sovereign-vault-X.tail5bea38.ts.net/admin
```

For browser extensions and mobile apps, use the HTTPS URL (port 443 is default).

---

## 10. References

- Forgejo implementation: `vm/forgejo/init.sh`
- TLS cert pattern: `vm/forgejo/init.sh:248-274`
- Dynamic hostname: `vm/forgejo/init.sh:254-259`
- PostgreSQL wait: `vm/forgejo/init.sh:218-236`
- Vaultwarden docs: https://github.com/dani-garcia/vaultwarden/wiki
