#!/sbin/openrc-run
# TEAM_012: Sovereign Forgejo VM Init Script - OpenRC format

description="Sovereign Forgejo VM networking and services"

depend() {
    need localmount
    after bootmisc
}

start() {
    ebegin "Starting Sovereign Forgejo VM"
    
    # Debug marker
    echo "$(date): sovereign-init started" >> /var/log/sovereign-debug.log
    
    # Create device nodes (Alpine has no udev)
    if [ ! -e /dev/vsock ]; then
        einfo "Creating /dev/vsock device node"
        mknod /dev/vsock c 10 121
        chmod 666 /dev/vsock
    fi
    
    # TEAM_012: Create /dev/net/tun for gvforwarder TAP interface
    if [ ! -e /dev/net/tun ]; then
        einfo "Creating /dev/net/tun device node"
        mkdir -p /dev/net
        mknod /dev/net/tun c 10 200
        chmod 666 /dev/net/tun
    fi
    
    # Start gvforwarder to connect to host's gvproxy
    echo "$(date): checking gvforwarder" >> /var/log/sovereign-debug.log
    if [ -x /usr/local/bin/gvforwarder ]; then
        echo "$(date): starting gvforwarder" >> /var/log/sovereign-debug.log
        /usr/local/bin/gvforwarder -url vsock://2:1024/connect >> /var/log/gvforwarder.log 2>&1 &
        echo $! > /run/gvforwarder.pid
        sleep 5
        
        # Configure tap0 if created
        if ip link show tap0 >/dev/null 2>&1; then
            einfo "Configuring tap0 network interface"
            ip addr add 192.168.127.2/24 dev tap0
            ip link set tap0 up
            ip route add default via 192.168.127.1
            echo "nameserver 8.8.8.8" > /etc/resolv.conf
        else
            ewarn "tap0 not created by gvforwarder"
        fi
    fi
    
    # Start Tailscale
    AUTHKEY=$(cat /proc/cmdline | tr ' ' '\n' | grep tailscale.authkey | cut -d= -f2)
    if [ -z "$AUTHKEY" ] && [ -f /etc/tailscale/authkey ]; then
        AUTHKEY=$(cat /etc/tailscale/authkey)
    fi
    if [ -n "$AUTHKEY" ]; then
        einfo "Joining Tailscale network"
        # Use userspace networking until kernel has netfilter
        tailscaled --tun=userspace-networking --state=/var/lib/tailscale/tailscaled.state &
        sleep 3
        tailscale up --authkey="$AUTHKEY" --hostname=forge-vm
    else
        ewarn "No Tailscale auth key provided"
    fi
    
    # Wait for SQL VM to be reachable
    einfo "Waiting for sql-vm PostgreSQL..."
    RETRIES=30
    while [ $RETRIES -gt 0 ]; do
        if nc -z sql-vm 5432 2>/dev/null; then
            einfo "sql-vm reachable"
            break
        fi
        sleep 2
        RETRIES=$((RETRIES - 1))
    done
    if [ $RETRIES -eq 0 ]; then
        ewarn "sql-vm not reachable after 60s - Forgejo may fail to start"
    fi
    
    # Start Forgejo
    einfo "Starting Forgejo"
    su forgejo -c "forgejo web --config /etc/forgejo/app.ini" >> /var/log/forgejo/forgejo.log 2>&1 &
    echo $! > /run/forgejo.pid
    
    eend 0
}

stop() {
    ebegin "Stopping Sovereign Forgejo VM"
    tailscale down 2>/dev/null || true
    if [ -f /run/forgejo.pid ]; then
        kill $(cat /run/forgejo.pid) 2>/dev/null || true
        rm -f /run/forgejo.pid
    fi
    if [ -f /run/gvforwarder.pid ]; then
        kill $(cat /run/gvforwarder.pid) 2>/dev/null || true
        rm -f /run/gvforwarder.pid
    fi
    eend 0
}
