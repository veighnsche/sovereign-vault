# Sovereign Forgejo VM - Alpine Linux ARM64
# TEAM_012: Git forge with CI/CD capabilities

FROM --platform=linux/arm64 alpine:3.19

# Install Forgejo and dependencies
RUN apk add --no-cache \
    forgejo \
    git \
    openssh-server \
    openrc \
    busybox-initscripts \
    netcat-openbsd \
    curl \
    ca-certificates

# TEAM_011: Install Tailscale static binary (NOT Alpine package)
# Alpine package is outdated. Download static binary directly.
RUN TAILSCALE_VERSION="1.92.3" && \
    curl -fsSL "https://pkgs.tailscale.com/stable/tailscale_${TAILSCALE_VERSION}_arm64.tgz" -o /tmp/tailscale.tgz && \
    tar -xzf /tmp/tailscale.tgz -C /tmp && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscale /usr/bin/ && \
    cp /tmp/tailscale_${TAILSCALE_VERSION}_arm64/tailscaled /usr/sbin/ && \
    rm -rf /tmp/tailscale* && \
    mkdir -p /var/lib/tailscale /var/run/tailscale

# Create directories
RUN mkdir -p /data/forgejo/repositories \
    /var/log/forgejo \
    /etc/forgejo \
    /var/lib/tailscale \
    /run/openrc \
    /run/sshd

# Generate SSH host keys
RUN ssh-keygen -A

# Configure OpenRC
RUN touch /run/openrc/softlevel \
    && rc-update add sshd default \
    && rc-update add local default

# Copy configuration files
COPY config/app.ini /etc/forgejo/app.ini
COPY scripts/init.sh /etc/init.d/sovereign-init

# Make init script executable and enable it
RUN chmod +x /etc/init.d/sovereign-init \
    && rc-update add sovereign-init default

# Set correct ownership
RUN chown -R forgejo:forgejo /data/forgejo /var/log/forgejo /etc/forgejo

# Expose ports (documentation only - actual access via Tailscale)
EXPOSE 3000 22

CMD ["/sbin/init"]
