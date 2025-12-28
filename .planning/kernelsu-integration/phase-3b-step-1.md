# Phase 3B, Step 1 — PostgreSQL VM Setup

**Phase:** 3B (PostgreSQL)
**Step:** 1 of 3
**Team:** TEAM_001
**Status:** Pending
**Depends On:** Phase 3A Complete

---

## 0. READ THIS FIRST: AI Assistant Warning

> 🤖 **AI Confession:** I am creating a VM image that will run under pKVM. My failure modes:
> - Building an x86 image when the device is ARM64
> - Forgetting to include Tailscale in the image
> - Using the wrong PostgreSQL configuration (localhost only)
> - Not testing the image locally before deploying
>
> **The rule:** The VM must be ARM64, include Tailscale, and PostgreSQL must listen on all interfaces.

---

## 1. Goal

Create the PostgreSQL VM image and update `sovereign.go` with `--sql` support.

---

## 2. Pre-Conditions

- [ ] Phase 3A complete (root working)
- [ ] Docker/Podman installed
- [ ] Tailscale auth key ready

---

## 3. Task 1: Create Directory Structure

```bash
mkdir -p vm/sql/scripts
```

---

## 4. Task 2: Create Dockerfile

**File:** `vm/sql/Dockerfile`

```dockerfile
FROM alpine:3.19

# Install PostgreSQL and Tailscale
RUN apk add --no-cache \
    postgresql15 \
    postgresql15-contrib \
    tailscale \
    openrc \
    busybox-initscripts

# Create data directory
RUN mkdir -p /data/postgres && \
    chown postgres:postgres /data/postgres

# PostgreSQL config: listen on ALL interfaces (not just localhost)
RUN mkdir -p /etc/postgresql && \
    echo "listen_addresses = '*'" > /etc/postgresql/postgresql.conf && \
    echo "port = 5432" >> /etc/postgresql/postgresql.conf && \
    echo "host all all 0.0.0.0/0 md5" >> /etc/postgresql/pg_hba.conf && \
    echo "host all all ::/0 md5" >> /etc/postgresql/pg_hba.conf

# Init script for Sovereign Vault
COPY scripts/init.sh /etc/init.d/sovereign-init
RUN chmod +x /etc/init.d/sovereign-init

# Enable services at boot
RUN rc-update add tailscale default && \
    rc-update add postgresql default && \
    rc-update add sovereign-init default

CMD ["/sbin/init"]
```

> 🤖 **AI Warning:** `listen_addresses = '*'` is CRITICAL. If set to `localhost`, PostgreSQL won't be reachable from other VMs or Tailscale.

---

## 5. Task 3: Create Init Script

**File:** `vm/sql/scripts/init.sh`

```bash
#!/bin/sh
### BEGIN INIT INFO
# Provides:          sovereign-init
# Required-Start:    $all
# Default-Start:     2 3 4 5
### END INIT INFO

case "$1" in
    start)
        echo "Sovereign SQL VM initializing..."
        
        # Initialize PostgreSQL data directory if needed
        if [ ! -f /data/postgres/PG_VERSION ]; then
            echo "Initializing PostgreSQL database..."
            su postgres -c "initdb -D /data/postgres"
            
            # Set postgres password
            su postgres -c "pg_ctl -D /data/postgres start"
            sleep 2
            su postgres -c "psql -c \"ALTER USER postgres PASSWORD 'sovereign';\""
            su postgres -c "pg_ctl -D /data/postgres stop"
        fi
        
        # Start Tailscale with auth key from kernel cmdline
        AUTHKEY=$(cat /proc/cmdline | tr ' ' '\n' | grep tailscale.authkey | cut -d= -f2)
        if [ -n "$AUTHKEY" ]; then
            echo "Joining Tailscale network..."
            tailscaled &
            sleep 2
            tailscale up --authkey="$AUTHKEY" --hostname=sql-vm
        else
            echo "WARNING: No Tailscale auth key provided"
        fi
        
        echo "Sovereign SQL VM ready"
        ;;
    stop)
        tailscale down
        ;;
esac
```

---

## 6. Task 4: Create .env File

**File:** `.env` (project root, if not exists)

```bash
TAILSCALE_AUTHKEY=tskey-auth-XXXXX-XXXXXXXXXXXXXXXXXX
```

Get your auth key from: https://login.tailscale.com/admin/settings/keys

> 🤖 **AI Warning:** This file contains secrets. Do NOT commit to git. Add to .gitignore.

---

## 7. Task 5: Update sovereign.go

Add `--sql` flag support. Key additions:

```go
var flagSQL bool

func init() {
    flag.BoolVar(&flagSQL, "sql", false, "Target PostgreSQL VM")
    // ... existing flags
}

// In cmdBuild():
if flagSQL {
    return buildSQL()
}

func buildSQL() error {
    fmt.Println("=== Building PostgreSQL VM ===")
    
    // Build with Docker
    cmd := exec.Command("docker", "build",
        "-t", "sovereign-sql",
        "-f", "vm/sql/Dockerfile",
        "vm/sql")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("docker build failed: %w", err)
    }
    
    fmt.Println("✓ PostgreSQL VM image built")
    return nil
}
```

---

## 8. Task 6: Build and Test Locally

```bash
# Build the image
go run sovereign.go build --sql

# Verify image exists
docker images | grep sovereign-sql

# Test run locally (just to verify it starts)
docker run --rm -it sovereign-sql /bin/sh -c "postgres --version"
```

---

## 9. Verification Checklist

```bash
echo "=== Phase 3B Step 1 Verification ==="

echo -n "1. Directory exists: "
[ -d vm/sql/scripts ] && echo "✓" || echo "✗ FAIL"

echo -n "2. Dockerfile exists: "
[ -f vm/sql/Dockerfile ] && echo "✓" || echo "✗ FAIL"

echo -n "3. Init script exists: "
[ -f vm/sql/scripts/init.sh ] && echo "✓" || echo "✗ FAIL"

echo -n "4. .env exists: "
[ -f .env ] && echo "✓" || echo "✗ FAIL"

echo -n "5. Docker image built: "
docker images | grep -q sovereign-sql && echo "✓" || echo "✗ FAIL"

echo "=== End Verification ==="
```

---

## 10. Checkpoint

- [ ] `vm/sql/` directory created
- [ ] Dockerfile created with PostgreSQL + Tailscale
- [ ] Init script handles Tailscale auth from cmdline
- [ ] `.env` file has Tailscale auth key
- [ ] `sovereign.go` updated with `--sql` flag
- [ ] `go run sovereign.go build --sql` succeeds
- [ ] Docker image `sovereign-sql` exists

---

## Next Step

Proceed to **[Phase 3B, Step 2 — Deploy & Start VM](phase-3b-step-2.md)**
