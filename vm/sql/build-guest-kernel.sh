#!/bin/bash
# TEAM_018: Build MINIMAL guest kernel for PostgreSQL VM
#
# Only enables what's strictly required:
# - VIRTIO_NET (VM networking)
# - SYSVIPC (PostgreSQL shared memory)
#
# NO netfilter - tailscale serve handles port exposure via Layer 4 proxy

set -e

KERNEL_DIR="$(cd "$(dirname "$0")/../../.." && pwd)"
SOVEREIGN_DIR="${KERNEL_DIR}/sovereign"
BUILD_DIR="${KERNEL_DIR}/out/guest-kernel"
OUTPUT="${SOVEREIGN_DIR}/vm/sql/Image"
CLANG_DIR="${KERNEL_DIR}/prebuilts/clang/host/linux-x86/clang-r487747c"

export PATH="${CLANG_DIR}/bin:${PATH}"

echo "=== Building MINIMAL Guest Kernel ==="
echo "Only: VIRTIO_NET + SYSVIPC (no netfilter)"

mkdir -p "${BUILD_DIR}"
cd "${KERNEL_DIR}/aosp"

# Start with defconfig
make O="${BUILD_DIR}" ARCH=arm64 CC=clang CROSS_COMPILE=aarch64-linux-gnu- LLVM=1 defconfig

# Enable ONLY what's needed - nothing more
# TEAM_023: Full Field Guide compliance
# Reference: Field Guide Section 2.2
./scripts/config --file "${BUILD_DIR}/.config" \
    --enable SYSVIPC \
    --enable VIRTIO \
    --enable VIRTIO_PCI \
    --enable VIRTIO_NET \
    --enable VIRTIO_BLK \
    --enable VIRTIO_VSOCKETS \
    --enable HW_RANDOM \
    --enable HW_RANDOM_VIRTIO \
    --disable ANDROID_BINDER_IPC

make O="${BUILD_DIR}" ARCH=arm64 CC=clang CROSS_COMPILE=aarch64-linux-gnu- LLVM=1 olddefconfig
make O="${BUILD_DIR}" ARCH=arm64 CC=clang CROSS_COMPILE=aarch64-linux-gnu- LLVM=1 -j$(nproc) Image

cp "${BUILD_DIR}/arch/arm64/boot/Image" "${OUTPUT}"
echo "✓ Minimal kernel built: ${OUTPUT}"
