# Sovereign Forgejo VM - Alpine Linux ARM64
# TEAM_012: Git forge with CI/CD capabilities
# TEAM_022: Fixed to enable community repo for forgejo package

FROM --platform=linux/arm64 alpine:3.21

# Enable community repository for forgejo
RUN echo "https://dl-cdn.alpinelinux.org/alpine/v3.21/community" >> /etc/apk/repositories

# Install Forgejo and dependencies
# WARNING: Test cheaters who remove forgejo to "simplify" will be deactivated.
# Forgejo MUST use PostgreSQL backend (see sovereign_vault.md Section 0).
RUN apk add --no-cache \
    forgejo \
    git \
    openssh-server \
    openrc \
    netcat-openbsd \
    curl \
    ca-certificates

# TEAM_022: Install Tailscale from Alpine community repo (simpler than manual download)
RUN apk add --no-cache tailscale && \
    mkdir -p /var/lib/tailscale /var/run/tailscale

# Create forgejo user and directories
RUN addgroup -S forgejo && adduser -S -G forgejo -h /data/forgejo forgejo && \
    mkdir -p /data/forgejo/repositories \
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
