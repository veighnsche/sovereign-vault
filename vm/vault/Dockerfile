# Sovereign Vaultwarden VM - Alpine Linux ARM64
# TEAM_034: Password manager with Tailscale TLS and PostgreSQL backend
#
# Based on working Forgejo pattern. See docs/VAULTWARDEN_IMPLEMENTATION_GUIDE.md
#
# CRITICAL: Vaultwarden REQUIRES valid HTTPS (WebCrypto API needs secure context)

FROM alpine:3.23

# Install dependencies
# TEAM_034: NO openrc - we use custom init.sh (OpenRC hangs in AVF)
RUN apk add --no-cache \
    curl \
    ca-certificates \
    iproute2 \
    netcat-openbsd \
    tzdata

# TEAM_035: Download Vaultwarden ARM64 binary (updated from 1.32.5)
# Check https://github.com/dani-garcia/vaultwarden/releases for latest version
ARG VAULTWARDEN_VERSION=1.35.0
RUN curl -fsSL "https://github.com/dani-garcia/vaultwarden/releases/download/${VAULTWARDEN_VERSION}/vaultwarden-${VAULTWARDEN_VERSION}-linux-arm64.tar.gz" \
    -o /tmp/vaultwarden.tar.gz && \
    tar -xzf /tmp/vaultwarden.tar.gz -C /tmp && \
    mv /tmp/vaultwarden /usr/bin/vaultwarden && \
    chmod +x /usr/bin/vaultwarden && \
    rm -rf /tmp/vaultwarden*

# Download web vault (UI)
RUN curl -fsSL "https://github.com/dani-garcia/bw_web_builds/releases/download/v2024.6.2c/bw_web_v2024.6.2c.tar.gz" \
    -o /tmp/web-vault.tar.gz && \
    mkdir -p /data/vault/web-vault && \
    tar -xzf /tmp/web-vault.tar.gz -C /data/vault && \
    rm /tmp/web-vault.tar.gz

# TEAM_034: Install Tailscale static binary (same as SQL/Forgejo VMs)
RUN TAILSCALE_VERSION="1.92.3" && \
    curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_arm64.tgz" \
    -o /tmp/tailscale.tgz && \
    tar -xzf /tmp/tailscale.tgz -C /tmp && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscale /usr/bin/ && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscaled /usr/sbin/ && \
    rm -rf /tmp/tailscale* && \
    mkdir -p /var/lib/tailscale /var/run/tailscale

# Create vaultwarden user (runs as non-root for security)
RUN addgroup -S vaultwarden && adduser -S -G vaultwarden -h /data/vault -s /bin/sh vaultwarden

# Create directories
RUN mkdir -p \
    /data/vault/data \
    /data/vault/tls \
    /data/tailscale \
    /etc/vaultwarden

# Copy configuration template
COPY config/env.template /etc/vaultwarden/env.template

# Set ownership
RUN chown -R vaultwarden:vaultwarden /data/vault

# Create tailscale directories
RUN mkdir -p /etc/tailscale /var/lib/tailscale

# Expose ports (documentation only - actual access via Tailscale)
EXPOSE 443 80

# TEAM_034: No CMD or ENTRYPOINT
# The init script is injected by rootfs.go during PrepareForAVF()
CMD ["/bin/sh"]
