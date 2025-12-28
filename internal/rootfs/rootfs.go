// Package rootfs provides rootfs AVF preparation utilities
// TEAM_010: Extracted from main.go during CLI refactor
package rootfs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PrepareForAVF fixes Alpine rootfs for AVF/crosvm compatibility
// This is IDEMPOTENT - safe to run multiple times
// TEAM_007: Original implementation
// TEAM_011: Added dbPassword parameter for secure credential handling
func PrepareForAVF(rootfsPath string, dbPassword string) error {
	mountDir := "/tmp/sovereign-rootfs-prep"
	os.MkdirAll(mountDir, 0755)

	// Mount rootfs
	if err := exec.Command("sudo", "mount", rootfsPath, mountDir).Run(); err != nil {
		return fmt.Errorf("mount failed: %w", err)
	}
	defer func() {
		exec.Command("sudo", "umount", mountDir).Run()
		os.RemoveAll(mountDir)
	}()

	// Fix 1: Ensure gvforwarder init script has correct dependencies
	// The script must run AFTER devfs mounts tmpfs over /dev
	gvforwarderInit := mountDir + "/etc/init.d/gvforwarder"
	if _, err := os.Stat(gvforwarderInit); err == nil {
		// Read current content
		content, err := os.ReadFile(gvforwarderInit)
		if err != nil {
			return fmt.Errorf("failed to read gvforwarder init: %w", err)
		}

		// Check if already fixed (idempotent check)
		if !strings.Contains(string(content), "need devfs") {
			// Fix the depend() function to require devfs
			newContent := strings.Replace(string(content),
				"need localmount",
				"need localmount\n\tneed devfs",
				1)

			if err := exec.Command("sudo", "sh", "-c",
				fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", gvforwarderInit, newContent)).Run(); err != nil {
				return fmt.Errorf("failed to update gvforwarder init: %w", err)
			}
			fmt.Println("  ✓ Fixed gvforwarder init script (added devfs dependency)")
		} else {
			fmt.Println("  ✓ gvforwarder init script already fixed")
		}
	}

	// Fix 2: Create local.d script for early device node creation (belt and suspenders)
	// This runs early in boot and ensures critical device nodes exist
	localDDir := mountDir + "/etc/local.d"
	exec.Command("sudo", "mkdir", "-p", localDDir).Run()

	devNodesScript := localDDir + "/00-avf-devices.start"
	devNodesContent := `#!/bin/sh
# TEAM_007: Ensure AVF-required device nodes exist
# This script runs early via local service

# Console devices (required for init)
[ -e /dev/console ] || mknod /dev/console c 5 1
[ -e /dev/tty ] || mknod /dev/tty c 5 0
[ -e /dev/tty0 ] || mknod /dev/tty0 c 4 0
[ -e /dev/null ] || mknod /dev/null c 1 3

# vsock device (required for gvisor-tap-vsock networking)
[ -e /dev/vsock ] || mknod /dev/vsock c 10 121
[ -e /dev/vhost-vsock ] || mknod /dev/vhost-vsock c 10 241

# TUN device (required for gvforwarder to create tap interface)
# TEAM_011: Added - gvforwarder fails with "cannot create tap device" without this
mkdir -p /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

# Set permissions
chmod 666 /dev/null /dev/tty 2>/dev/null
chmod 600 /dev/console 2>/dev/null
`
	// Write the script (idempotent - overwrites if exists)
	writeCmd := fmt.Sprintf("cat > %s << 'EOFSCRIPT'\n%sEOFSCRIPT", devNodesScript, devNodesContent)
	if err := exec.Command("sudo", "sh", "-c", writeCmd).Run(); err != nil {
		return fmt.Errorf("failed to create device nodes script: %w", err)
	}
	exec.Command("sudo", "chmod", "+x", devNodesScript).Run()
	fmt.Println("  ✓ Created /etc/local.d/00-avf-devices.start")

	// Fix 3: Ensure 'local' service is enabled in default runlevel
	localLink := mountDir + "/etc/runlevels/default/local"
	if _, err := os.Stat(localLink); os.IsNotExist(err) {
		exec.Command("sudo", "ln", "-sf", "/etc/init.d/local", localLink).Run()
		fmt.Println("  ✓ Enabled 'local' service in default runlevel")
	}

	// Fix 4: Ensure devfs runs in sysinit runlevel
	devfsLink := mountDir + "/etc/runlevels/sysinit/devfs"
	if _, err := os.Stat(devfsLink); os.IsNotExist(err) {
		exec.Command("sudo", "ln", "-sf", "/etc/init.d/devfs", devfsLink).Run()
		fmt.Println("  ✓ Enabled 'devfs' service in sysinit runlevel")
	}

	// Fix 5: Pre-create critical device nodes directly in rootfs
	// TEAM_011: The local.d script runs too late - sovereign-init needs these earlier
	devDir := mountDir + "/dev"
	devNetDir := devDir + "/net"
	exec.Command("sudo", "mkdir", "-p", devNetDir).Run()

	// Create device nodes if they don't exist
	devNodes := []struct{ path, major, minor string }{
		{devDir + "/console", "5", "1"},
		{devDir + "/null", "1", "3"},
		{devDir + "/zero", "1", "5"},
		{devDir + "/tty", "5", "0"},
		{devDir + "/random", "1", "8"},
		{devDir + "/urandom", "1", "9"},
		{devDir + "/vsock", "10", "121"},
		{devNetDir + "/tun", "10", "200"},
	}
	for _, dev := range devNodes {
		if _, err := os.Stat(dev.path); os.IsNotExist(err) {
			exec.Command("sudo", "mknod", dev.path, "c", dev.major, dev.minor).Run()
			exec.Command("sudo", "chmod", "666", dev.path).Run()
		}
	}
	fmt.Println("  ✓ Pre-created device nodes in /dev")

	// Fix 6: Create simple_init script to bypass OpenRC (which hangs on AVF)
	// TEAM_012: OpenRC doesn't complete on AVF - use simple shell init instead
	// TEAM_011: Inject DB password securely (not in shell history or git)
	// TEAM_016/017: TAP networking with all fixes from debugging session
	simpleInitPath := mountDir + "/sbin/simple_init"
	simpleInitContent := fmt.Sprintf(`#!/bin/sh
# TEAM_012: Simple init script bypassing OpenRC for AVF
# TEAM_016/017: TAP networking with verified working configuration
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export DB_PASSWORD="%s"`, dbPassword) + `

# TEAM_018: Log ALL output to file for debugging
LOG=/var/log/init.log
mkdir -p /var/log
exec > $LOG 2>&1
set -x
echo "=== SIMPLE INIT START ==="

mount -t proc proc /proc
mount -t sysfs sysfs /sys

# TEAM_017: Mount tmpfs for shared memory (required for PostgreSQL)
mkdir -p /dev/shm /tmp
mount -t tmpfs -o mode=1777 tmpfs /dev/shm
mount -t tmpfs tmpfs /tmp

# TEAM_017: Set system time (required for TLS certificate validation)
# Without this, Tailscale fails with certificate errors
date -s "2025-12-28 22:00:00" 2>/dev/null || true

# Create device nodes
mkdir -p /dev/net
[ -e /dev/net/tun ] || mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun

# TEAM_016/017: Configure TAP networking
# Host TAP is 192.168.100.1, guest is 192.168.100.2
echo "=== Configuring TAP Network ==="
sleep 3

# Find network interface dynamically (may not be eth0)
IFACE=$(ls /sys/class/net/ | grep -v lo | head -1)
echo "Found interface: $IFACE"

if [ -n "$IFACE" ]; then
    ip addr add 192.168.100.2/24 dev "$IFACE"
    ip link set "$IFACE" up
    ip route add default via 192.168.100.1
    echo "nameserver 8.8.8.8" > /etc/resolv.conf
    echo "Network configured on $IFACE"
    ip addr show "$IFACE"
else
    echo "WARNING: No network interface found"
    ip link
fi

# Test connectivity
sleep 3
echo "=== Testing Network ==="
ping -c 3 8.8.8.8 2>&1 || echo "Ping failed"

# Start Tailscale
# TEAM_018: Persist state on /data, always use authkey if provided
echo "=== Starting Tailscale ==="
mkdir -p /data/tailscale /var/run/tailscale
/usr/sbin/tailscaled --tun=userspace-networking \
    --state=/data/tailscale/tailscaled.state \
    --socket=/var/run/tailscale/tailscaled.sock &
sleep 10

AUTHKEY=$(cat /proc/cmdline | tr ' ' '\n' | grep tailscale.authkey | cut -d= -f2)
if [ -n "$AUTHKEY" ]; then
    # Authkey handles both new registration and reconnection
    /usr/bin/tailscale up --authkey="$AUTHKEY" --hostname=sovereign-sql 2>&1
else
    echo "WARNING: No authkey provided"
fi
sleep 5
/usr/bin/tailscale status 2>&1

# Start PostgreSQL
echo "=== Starting PostgreSQL ==="
mkdir -p /run/postgresql /data/postgres /var/log
touch /var/log/postgresql.log
chown -R postgres:postgres /run/postgresql /data/postgres /var/log/postgresql.log

if [ ! -f /data/postgres/PG_VERSION ]; then
    echo "Initializing PostgreSQL database..."
    su postgres -c "initdb -D /data/postgres" 2>&1
    # TEAM_017: Configure PostgreSQL for AVF environment
    cat >> /data/postgres/postgresql.conf << 'PGCONF'
listen_addresses = '*'
dynamic_shared_memory_type = mmap
shared_buffers = 32MB
PGCONF
    echo "host all all 0.0.0.0/0 md5" >> /data/postgres/pg_hba.conf
    echo "host all all ::/0 md5" >> /data/postgres/pg_hba.conf
fi

su postgres -c "pg_ctl -D /data/postgres -l /var/log/postgresql.log start" 2>&1
sleep 3
su postgres -c "psql -c \"ALTER USER postgres PASSWORD '$DB_PASSWORD';\"" 2>&1
echo "PostgreSQL version:"
su postgres -c "psql -c \"SELECT version();\"" 2>&1

echo "=== INIT COMPLETE ==="
while true; do sleep 3600; done
`
	simpleInitCmd := fmt.Sprintf("cat > %s << 'EOFSCRIPT'\n%sEOFSCRIPT", simpleInitPath, simpleInitContent)
	if err := exec.Command("sudo", "sh", "-c", simpleInitCmd).Run(); err != nil {
		return fmt.Errorf("failed to create simple_init: %w", err)
	}
	exec.Command("sudo", "chmod", "+x", simpleInitPath).Run()
	fmt.Println("  ✓ Created /sbin/simple_init (OpenRC bypass)")

	// Fix 7: Create dhclient wrapper for gvforwarder
	// TEAM_012: gvforwarder calls dhclient but Alpine uses udhcpc
	dhclientPath := mountDir + "/usr/bin/dhclient"
	dhclientContent := `#!/bin/sh
# TEAM_012: dhclient wrapper - gvforwarder expects dhclient, Alpine uses udhcpc
IFACE="${@: -1}"
exec /sbin/udhcpc -i "$IFACE" -n -q -f -S 2>/dev/null
`
	dhclientCmd := fmt.Sprintf("cat > %s << 'EOFSCRIPT'\n%sEOFSCRIPT", dhclientPath, dhclientContent)
	if err := exec.Command("sudo", "sh", "-c", dhclientCmd).Run(); err != nil {
		return fmt.Errorf("failed to create dhclient wrapper: %w", err)
	}
	exec.Command("sudo", "chmod", "+x", dhclientPath).Run()
	fmt.Println("  ✓ Created /usr/bin/dhclient wrapper")

	fmt.Println("  ✓ Rootfs prepared for AVF")
	return nil
}
