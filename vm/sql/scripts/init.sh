#!/sbin/openrc-run
# TEAM_011: Sovereign SQL VM Init Script - OpenRC format

description="Sovereign SQL VM networking and services"

depend() {
    need localmount
    after bootmisc
}

start() {
    ebegin "Starting Sovereign SQL VM"
    
    # Debug marker
    echo "$(date): sovereign-init started" >> /var/log/sovereign-debug.log
    
    # Create device nodes (Alpine has no udev, devfs mounts tmpfs on /dev)
    if [ ! -e /dev/vsock ]; then
        einfo "Creating /dev/vsock device node"
        mknod /dev/vsock c 10 121
        chmod 666 /dev/vsock
    fi
    
    # TEAM_011: Create /dev/net/tun for gvforwarder TAP interface
    if [ ! -e /dev/net/tun ]; then
        einfo "Creating /dev/net/tun device node"
        mkdir -p /dev/net
        mknod /dev/net/tun c 10 200
        chmod 666 /dev/net/tun
    fi
    
    # Start gvforwarder to connect to host's gvproxy
    # Host gvproxy listens on vsock port 1024, CID 2 = host
    echo "$(date): checking gvforwarder" >> /var/log/sovereign-debug.log
    if [ -x /usr/local/bin/gvforwarder ]; then
        echo "$(date): starting gvforwarder" >> /var/log/sovereign-debug.log
        ls -la /dev/vsock >> /var/log/sovereign-debug.log 2>&1
        /usr/local/bin/gvforwarder -url vsock://2:1024/connect >> /var/log/gvforwarder.log 2>&1 &
        echo $! > /run/gvforwarder.pid
        echo "$(date): gvforwarder started with PID $(cat /run/gvforwarder.pid)" >> /var/log/sovereign-debug.log
        sleep 5
        
        # gvforwarder creates tap0 interface
        if ip link show tap0 >/dev/null 2>&1; then
            einfo "Configuring tap0 network interface"
            ip addr add 192.168.127.2/24 dev tap0
            ip link set tap0 up
            ip route add default via 192.168.127.1
            echo "nameserver 8.8.8.8" > /etc/resolv.conf
        else
            ewarn "tap0 not created by gvforwarder"
            echo "$(date): tap0 not found" >> /var/log/sovereign-debug.log
        fi
    else
        ewarn "gvforwarder not found"
        echo "$(date): gvforwarder not found" >> /var/log/sovereign-debug.log
    fi
    
    # Initialize PostgreSQL data directory if needed
    # TEAM_011: Fix ownership (docker export can mess up uid mapping)
    echo "$(date): checking postgres data dir" >> /var/log/sovereign-debug.log
    chown -R postgres:postgres /data/postgres 2>/dev/null
    
    if [ ! -f /data/postgres/PG_VERSION ]; then
        einfo "Initializing PostgreSQL database"
        echo "$(date): running initdb" >> /var/log/sovereign-debug.log
        su postgres -c "initdb -D /data/postgres" >> /var/log/sovereign-debug.log 2>&1
        cp /etc/postgresql/postgresql.conf /data/postgres/
        cp /etc/postgresql/pg_hba.conf /data/postgres/
        
        echo "$(date): starting pg for password setup" >> /var/log/sovereign-debug.log
        su postgres -c "pg_ctl -D /data/postgres -l /tmp/pg.log start"
        sleep 2
        su postgres -c "psql -c \"ALTER USER postgres PASSWORD 'sovereign';\"" >> /var/log/sovereign-debug.log 2>&1
        su postgres -c "pg_ctl -D /data/postgres stop"
        echo "$(date): pg init complete" >> /var/log/sovereign-debug.log
    fi
    
    # Start PostgreSQL
    einfo "Starting PostgreSQL"
    su postgres -c "pg_ctl -D /data/postgres -l /var/log/postgresql/postgresql.log start"
    
    # Start Tailscale
    AUTHKEY=$(cat /proc/cmdline | tr ' ' '\n' | grep tailscale.authkey | cut -d= -f2)
    if [ -z "$AUTHKEY" ] && [ -f /etc/tailscale/authkey ]; then
        AUTHKEY=$(cat /etc/tailscale/authkey)
    fi
    if [ -n "$AUTHKEY" ]; then
        einfo "Joining Tailscale network"
        # TEAM_011: Use userspace networking - kernel lacks netfilter for iptables
        tailscaled --tun=userspace-networking --state=/var/lib/tailscale/tailscaled.state &
        sleep 3
        tailscale up --authkey="$AUTHKEY" --hostname=sql-vm
    else
        ewarn "No Tailscale auth key provided"
    fi
    
    eend 0
}

stop() {
    ebegin "Stopping Sovereign SQL VM"
    tailscale down 2>/dev/null || true
    su postgres -c "pg_ctl -D /data/postgres stop" 2>/dev/null || true
    if [ -f /run/gvforwarder.pid ]; then
        kill $(cat /run/gvforwarder.pid) 2>/dev/null || true
        rm -f /run/gvforwarder.pid
    fi
    eend 0
}
