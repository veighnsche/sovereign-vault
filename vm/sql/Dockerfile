# TEAM_006: PostgreSQL VM for Sovereign Vault
# Architecture: ARM64 (runs on Pixel 6 under pKVM)
FROM alpine:3.19

# Install PostgreSQL and dependencies - NO kernel package
# TEAM_006: We will use the host Android kernel or a custom-built kernel
# because Alpine's linux-virt uses EFI stub format which crosvm can't boot
RUN apk add --no-cache \
    postgresql15 \
    postgresql15-contrib \
    openrc \
    iptables \
    ip6tables \
    iproute2 \
    curl

# TEAM_011: Install Tailscale static binary (NOT Alpine package)
# Alpine package is outdated. Download static binary directly.
RUN TAILSCALE_VERSION="1.92.3" && \
    curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_arm64.tgz" -o /tmp/tailscale.tgz && \
    tar -xzf /tmp/tailscale.tgz -C /tmp && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscale /usr/bin/ && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscaled /usr/sbin/ && \
    rm -rf /tmp/tailscale* && \
    mkdir -p /var/lib/tailscale /var/run/tailscale

# Create data directory
RUN mkdir -p /data/postgres && \
    chown postgres:postgres /data/postgres

# PostgreSQL config: listen on ALL interfaces (not just localhost)
# TEAM_006: listen_addresses='*' is CRITICAL for Tailscale access
RUN mkdir -p /etc/postgresql && \
    echo "listen_addresses = '*'" > /etc/postgresql/postgresql.conf && \
    echo "port = 5432" >> /etc/postgresql/postgresql.conf && \
    echo "data_directory = '/data/postgres'" >> /etc/postgresql/postgresql.conf && \
    echo "# HBA - allow connections from anywhere (Tailscale provides security)" > /etc/postgresql/pg_hba.conf && \
    echo "local all all trust" >> /etc/postgresql/pg_hba.conf && \
    echo "host all all 0.0.0.0/0 md5" >> /etc/postgresql/pg_hba.conf && \
    echo "host all all ::/0 md5" >> /etc/postgresql/pg_hba.conf

# Init script for Sovereign Vault
COPY scripts/init.sh /etc/init.d/sovereign-init
RUN chmod +x /etc/init.d/sovereign-init

# TEAM_017: gvforwarder removed - using TAP networking instead

# TEAM_011: Create tailscale auth key directory
RUN mkdir -p /etc/tailscale /var/lib/tailscale

# Enable services at boot (tailscale started by simple_init, not OpenRC)
RUN rc-update add sovereign-init default

CMD ["/sbin/init"]
