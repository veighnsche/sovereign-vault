# Sovereign Forgejo VM - Alpine Linux ARM64
# TEAM_025: Git forge with TAP networking and persistent Tailscale
# Modeled after vm/sql/Dockerfile - the gold standard for AVF VMs
#
# WARNING: Test cheaters who remove forgejo to "simplify" will be deactivated.
# Forgejo MUST use PostgreSQL backend (see sovereign_vault.md Section 0).

# TEAM_029: Mirror SQL VM exactly - Alpine 3.23, no platform flag
FROM alpine:3.21

# Enable community repository for forgejo
RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.23/community" >> /etc/apk/repositories

# Create forgejo user BEFORE installing forgejo package
RUN addgroup -S forgejo && adduser -S -G forgejo -h /var/lib/forgejo -s /bin/sh forgejo

# Install dependencies (NOT Alpine's forgejo - it lacks PostgreSQL support)
# TEAM_025: NO openrc - we use custom init.sh which bypasses it (OpenRC hangs in AVF)
# TEAM_029: Alpine's forgejo only has SQLite. Download official binary with PostgreSQL.
RUN apk add --no-cache \
    git \
    openssh-server \
    netcat-openbsd \
    curl \
    ca-certificates \
    iproute2 \
    bash \
    postgresql-client \
    libcap

# TEAM_029: Extract Forgejo binary from official Docker image
# The Docker image has PostgreSQL support, the standalone binary doesn't!
# Copy from multi-arch image to get ARM64 binary with all database drivers
COPY --from=codeberg.org/forgejo/forgejo:9.0.3 /app/gitea/gitea /usr/bin/forgejo
RUN chmod +x /usr/bin/forgejo && forgejo --version

# TEAM_025: Install Tailscale static binary (NOT Alpine package)
# Alpine package is outdated. Download static binary directly.
# Same pattern as vm/sql/Dockerfile
# TEAM_029: Use same Tailscale version as SQL VM (1.92.3)
RUN TAILSCALE_VERSION="1.92.3" && \
    curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_arm64.tgz" -o /tmp/tailscale.tgz && \
    tar -xzf /tmp/tailscale.tgz -C /tmp && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscale /usr/bin/ && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscaled /usr/sbin/ && \
    rm -rf /tmp/tailscale* && \
    mkdir -p /var/lib/tailscale /var/run/tailscale

# Create directories for Forgejo data
RUN mkdir -p /data/forgejo/repositories \
    /var/log/forgejo \
    /etc/forgejo \
    /run/sshd

# Generate SSH host keys
RUN ssh-keygen -A

# Copy configuration files
COPY config/app.ini /etc/forgejo/app.ini

# Set correct ownership
RUN chown -R forgejo:forgejo /data/forgejo /var/log/forgejo /etc/forgejo

# Create tailscale directories
RUN mkdir -p /etc/tailscale /var/lib/tailscale

# Expose ports (documentation only - actual access via Tailscale)
EXPOSE 3000 22

# TEAM_025: No CMD or ENTRYPOINT
# The init script is injected by rootfs.go during PrepareForAVF()
# It will be at /sbin/init.sh and symlinked to /sbin/init
CMD ["/bin/sh"]
