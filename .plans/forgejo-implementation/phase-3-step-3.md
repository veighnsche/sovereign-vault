# Phase 3, Step 3: Dockerfile

**Feature:** Forgejo VM Implementation  
**Status:** 🔲 NOT STARTED  
**Parent:** [Phase 3: Implementation](phase-3.md)

---

## Goal

Update `vm/forgejo/Dockerfile` to use static Tailscale binary and remove OpenRC.

---

## Reference Implementation

**Copy patterns from:** `vm/sql/Dockerfile`

---

## Current State (WRONG)

```dockerfile
# Uses Alpine tailscale package and OpenRC
RUN apk add --no-cache tailscale
RUN rc-update add sshd default
CMD ["/sbin/init"]
```

---

## Tasks

### Task 1: Update base and package installation

```dockerfile
# Sovereign Forgejo VM - Alpine Linux ARM64
# TEAM_0XX: Git forge with TAP networking and persistent Tailscale

FROM --platform=linux/arm64 alpine:3.21

# Enable community repository for forgejo
RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories

# Create forgejo user BEFORE installing forgejo package
RUN addgroup -S forgejo && adduser -S -G forgejo -h /var/lib/forgejo -s /bin/sh forgejo

# Install Forgejo and dependencies (NO openrc, NO tailscale package)
RUN apk add --no-cache \
    forgejo \
    git \
    openssh-server \
    netcat-openbsd \
    curl \
    ca-certificates
```

### Task 2: Install Tailscale static binary

```dockerfile
# Install Tailscale static binary (same pattern as SQL VM)
ARG TAILSCALE_VERSION=1.78.3
RUN curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_arm64.tgz" \
    | tar -xzf - -C /tmp && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscale /usr/bin/ && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscaled /usr/sbin/ && \
    rm -rf /tmp/tailscale_* && \
    mkdir -p /var/lib/tailscale /var/run/tailscale
```

### Task 3: Create directories

```dockerfile
# Create directories for Forgejo data
RUN mkdir -p /data/forgejo/repositories \
    /var/log/forgejo \
    /etc/forgejo \
    /run/sshd

# Generate SSH host keys
RUN ssh-keygen -A
```

### Task 4: Copy configuration

```dockerfile
# Copy configuration files
COPY config/app.ini /etc/forgejo/app.ini

# Set correct ownership
RUN chown -R forgejo:forgejo /data/forgejo /var/log/forgejo /etc/forgejo
```

### Task 5: No CMD - init.sh is injected by rootfs.go

```dockerfile
# Expose ports (documentation only - actual access via Tailscale)
EXPOSE 3000 22

# NOTE: No CMD or ENTRYPOINT
# The init script is injected by rootfs.go during PrepareForAVF()
# It will be at /sbin/init.sh and symlinked to /sbin/init
```

---

## Key Changes

| Aspect | Before | After |
|--------|--------|-------|
| Tailscale | Alpine package | Static binary |
| OpenRC | Installed, services enabled | Not installed |
| CMD | `/sbin/init` | None (injected by rootfs.go) |
| Init script | OpenRC service script | Standalone init.sh |

---

## Expected Output

- Updated file: `vm/forgejo/Dockerfile` (~40 lines)
- No `openrc` package
- No `tailscale` package (uses static binary)
- No `rc-update` commands
- No `CMD` directive

---

## Verification

1. No openrc in apk add
2. Tailscale installed from static binary
3. No rc-update commands
4. No CMD or ENTRYPOINT

---

## Next Step

→ [Phase 3, Step 4: Go Code](phase-3-step-4.md)
