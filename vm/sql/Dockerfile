# TEAM_006: PostgreSQL VM for Sovereign Vault
# Architecture: ARM64 (runs on Pixel 6 under pKVM)
FROM alpine:3.32.2

# Install PostgreSQL and dependencies
# TEAM_020: Removed openrc - we use simple_init_tap which bypasses it
# Removed iptables/ip6tables - kernel uses userspace networking (no netfilter)
# TEAM_023: Added icu-data-full to fix musl collation bug (silent index corruption)
# Reference: Field Guide Section 2.1 - "musl libc Divergence"
RUN apk add --no-cache \
    postgresql15 \
    postgresql15-contrib \
    icu-data-full \
    iproute2 \
    curl

# TEAM_011: Install Tailscale static binary (NOT Alpine package)
# Alpine package is outdated. Download static binary directly.
RUN TAILSCALE_VERSION="1.92.4" && \
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

# TEAM_020: OpenRC not used - we use simple_init_tap which bypasses it entirely
# This is because OpenRC doesn't work reliably in crosvm environment

# Create tailscale directories
RUN mkdir -p /etc/tailscale /var/lib/tailscale

# TEAM_023: Fix musl thread stack size (default 128KB is too small for PostgreSQL)
# Reference: Field Guide Section 2.1 - "Default Thread Stack Size"
ENV PTHREAD_STACK_MIN=2097152

# Default CMD not used - crosvm boots with init=/sbin/simple_init
CMD ["/bin/sh"]
