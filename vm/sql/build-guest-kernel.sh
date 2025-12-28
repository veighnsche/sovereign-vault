#!/bin/bash
# TEAM_006: Build guest kernel for Sovereign SQL VM
# Uses microdroid_defconfig which has virtio drivers built-in
# This builds a RAW ARM64 Image (not EFI stub) that crosvm can boot

set -e

# KERNEL_DIR is the parent of sovereign (where aosp/ lives)
KERNEL_DIR="$(cd "$(dirname "$0")/../../.." && pwd)"
SOVEREIGN_DIR="${KERNEL_DIR}/sovereign"
BUILD_DIR="${KERNEL_DIR}/out/guest-kernel"
OUTPUT="${SOVEREIGN_DIR}/vm/sql/Image"

# Use Android's prebuilt clang
CLANG_DIR="${KERNEL_DIR}/prebuilts/clang/host/linux-x86/clang-r487747c"
export PATH="${CLANG_DIR}/bin:${PATH}"

echo "=== Building Guest Kernel for SQL VM ==="
echo "Kernel source: ${KERNEL_DIR}/aosp"
echo "Build dir: ${BUILD_DIR}"
echo "Output: ${OUTPUT}"
echo "Clang: ${CLANG_DIR}"

mkdir -p "${BUILD_DIR}"

# TEAM_018: Use gki_defconfig - has better networking support than microdroid
# microdroid_defconfig is too minimal (missing virtio_net, netfilter)
cd "${KERNEL_DIR}/aosp"

# Configure with clang
make O="${BUILD_DIR}" ARCH=arm64 \
    CC=clang \
    CROSS_COMPILE=aarch64-linux-gnu- \
    LLVM=1 \
    gki_defconfig

# TEAM_018: Enable required options using scripts/config
# This is REQUIRED for Tailscale and networking to work
echo "=== Enabling required kernel options ==="
cd "${KERNEL_DIR}/aosp"
./scripts/config --file "${BUILD_DIR}/.config" \
    --enable VIRTIO_NET \
    --enable TUN \
    --enable NETFILTER \
    --enable NETFILTER_ADVANCED \
    --enable NETFILTER_XTABLES \
    --enable NETFILTER_XT_MARK \
    --enable NETFILTER_XT_TARGET_MARK \
    --enable NETFILTER_XT_MATCH_MARK \
    --enable NETFILTER_XT_MATCH_STATE \
    --enable NETFILTER_XT_MATCH_CONNTRACK \
    --enable NF_CONNTRACK \
    --enable NF_CONNTRACK_MARK \
    --enable NF_NAT \
    --enable NF_TABLES \
    --enable NFT_COMPAT \
    --enable IP_NF_IPTABLES \
    --enable IP_NF_FILTER \
    --enable IP_NF_NAT \
    --enable IP_NF_TARGET_MASQUERADE \
    --enable IP_NF_MANGLE \
    --enable SYSVIPC

make O="${BUILD_DIR}" ARCH=arm64 \
    CC=clang \
    CROSS_COMPILE=aarch64-linux-gnu- \
    LLVM=1 \
    olddefconfig

# Build with clang
make O="${BUILD_DIR}" ARCH=arm64 \
    CC=clang \
    CROSS_COMPILE=aarch64-linux-gnu- \
    LLVM=1 \
    -j$(nproc) Image

# Copy output
cp "${BUILD_DIR}/arch/arm64/boot/Image" "${OUTPUT}"

echo ""
echo "✓ Guest kernel built: ${OUTPUT}"
file "${OUTPUT}"
