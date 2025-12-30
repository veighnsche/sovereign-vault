#!/bin/bash
# TEAM_034: Shared guest kernel builder for all VMs
# Previously in vm/sql/ - moved to shared location since Forgejo reuses SQL's kernel
#
# Required by: PostgreSQL, Forgejo, future VMs
# Features:
# - VIRTIO (crosvm networking, block devices)
# - SYSVIPC (PostgreSQL shared memory)
# - TUN (Tailscale native mode)
# - Netfilter (Tailscale fwmark routing)

set -e

KERNEL_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
SOVEREIGN_DIR="${KERNEL_DIR}/sovereign"
BUILD_DIR="${KERNEL_DIR}/out/guest-kernel"
OUTPUT="${SOVEREIGN_DIR}/vm/sql/Image"  # SQL is the primary; Forgejo uses SharedKernel
CLANG_DIR="${KERNEL_DIR}/prebuilts/clang/host/linux-x86/clang-r487747c"

export PATH="${CLANG_DIR}/bin:${PATH}"

echo "=== Building Shared Guest Kernel ==="
echo "Features: VIRTIO + SYSVIPC + TUN + Netfilter"

mkdir -p "${BUILD_DIR}"
cd "${KERNEL_DIR}/aosp"

# Start with defconfig
make O="${BUILD_DIR}" ARCH=arm64 CC=clang CROSS_COMPILE=aarch64-linux-gnu- LLVM=1 defconfig

# Configure kernel options
./scripts/config --file "${BUILD_DIR}/.config" \
    --enable SYSVIPC \
    \
    --enable VIRTIO \
    --enable VIRTIO_PCI \
    --enable VIRTIO_NET \
    --enable VIRTIO_BLK \
    --enable VIRTIO_VSOCKETS \
    --enable HW_RANDOM \
    --enable HW_RANDOM_VIRTIO \
    \
    --enable TUN \
    --disable ANDROID_BINDER_IPC \
    \
    --enable NETFILTER \
    --enable NF_CONNTRACK \
    --enable NETFILTER_XTABLES \
    --enable NETFILTER_XT_MARK \
    --enable NETFILTER_XT_CONNMARK \
    --enable NETFILTER_XT_TARGET_MARK \
    --enable NETFILTER_XT_TARGET_CONNMARK \
    \
    --enable NF_TABLES \
    --enable NF_TABLES_INET \
    --enable NF_TABLES_IPV4 \
    --enable NFT_COMPAT \
    --enable NFT_CT \
    --enable NFT_NAT \
    --enable NFT_MASQ \
    --enable NFT_REDIR \
    --enable NFT_REJECT \
    --enable NFT_CHAIN_NAT \
    --enable NFT_CHAIN_ROUTE \
    \
    --enable IP_NF_IPTABLES \
    --enable IP_NF_FILTER \
    --enable IP_NF_NAT \
    --enable IP_NF_TARGET_MASQUERADE \
    --enable IP_NF_TARGET_REJECT \
    \
    --enable IP_ADVANCED_ROUTER \
    --enable IP_MULTIPLE_TABLES \
    --enable IP_ROUTE_FWMARK \
    --enable IP_ROUTE_MULTIPATH \
    --enable FIB_RULES

make O="${BUILD_DIR}" ARCH=arm64 CC=clang CROSS_COMPILE=aarch64-linux-gnu- LLVM=1 olddefconfig
make O="${BUILD_DIR}" ARCH=arm64 CC=clang CROSS_COMPILE=aarch64-linux-gnu- LLVM=1 -j$(nproc) Image

cp "${BUILD_DIR}/arch/arm64/boot/Image" "${OUTPUT}"
echo ""
echo "✓ Guest kernel built: ${OUTPUT}"
echo "  Used by: SQL VM (primary), Forgejo VM (SharedKernel)"
