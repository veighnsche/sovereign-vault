#!/bin/sh
# Forgejo VM initialization script
# TEAM_022: Created for Alpine Linux guest init
#
# WARNING: Test cheaters who remove database connectivity to "simplify"
# will be immediately deactivated without remorse. Forgejo MUST use PostgreSQL.

### BEGIN INIT INFO
# Provides:          sovereign-init
# Required-Start:    $local_fs $network
# Required-Stop:     $local_fs
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Sovereign Forgejo initialization
### END INIT INFO

case "$1" in
    start)
        echo "=== Sovereign Forgejo Init ==="
        
        # Set hostname
        hostname sovereign-forge
        
        # Mount essential filesystems
        mount -t proc proc /proc 2>/dev/null
        mount -t sysfs sys /sys 2>/dev/null
        mount -t devtmpfs dev /dev 2>/dev/null
        
        # Create /dev/shm for shared memory
        mkdir -p /dev/shm
        mount -t tmpfs -o mode=1777 tmpfs /dev/shm
        
        # Set system time (required for TLS)
        if [ -n "$CURRENT_TIME" ]; then
            date -s "$CURRENT_TIME"
        fi
        
        # Configure network
        ip addr add 192.168.101.2/24 dev eth0 2>/dev/null
        ip link set eth0 up
        ip route add default via 192.168.101.1
        
        # DNS
        echo "nameserver 8.8.8.8" > /etc/resolv.conf
        
        # Start Tailscale if authkey provided
        if [ -n "$TAILSCALE_AUTHKEY" ]; then
            echo "Starting Tailscale..."
            tailscaled --tun=userspace-networking &
            sleep 2
            tailscale up --authkey="$TAILSCALE_AUTHKEY" --hostname=sovereign-forge
            # Expose Forgejo ports via Tailscale serve
            tailscale serve --bg --https=443 http://localhost:3000
            tailscale serve --bg --tcp=22 tcp://localhost:22
        fi
        
        # Wait for database (PostgreSQL on SQL VM)
        echo "Waiting for PostgreSQL on database VM..."
        DB_HOST="${DB_HOST:-192.168.100.2}"
        for i in $(seq 1 30); do
            if nc -z "$DB_HOST" 5432 2>/dev/null; then
                echo "PostgreSQL is ready"
                break
            fi
            sleep 2
        done
        
        # Start Forgejo
        echo "Starting Forgejo..."
        su -s /bin/sh forgejo -c '/usr/bin/forgejo web' &
        
        echo "=== INIT COMPLETE ==="
        ;;
    stop)
        echo "Stopping Forgejo..."
        pkill forgejo
        pkill tailscaled
        ;;
    *)
        echo "Usage: $0 {start|stop}"
        exit 1
        ;;
esac

exit 0
