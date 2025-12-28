#!/bin/bash
# TEAM_006: Build guest kernel for Sovereign SQL VM
# Uses microdroid_defconfig which has virtio drivers built-in
# This builds a RAW ARM64 Image (not EFI stub) that crosvm can boot

set -e

KERNEL_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BUILD_DIR="${KERNEL_DIR}/out/guest-kernel"
OUTPUT="${KERNEL_DIR}/vm/sql/Image"

# Use Android's prebuilt clang
CLANG_DIR="${KERNEL_DIR}/prebuilts/clang/host/linux-x86/clang-r487747c"
export PATH="${CLANG_DIR}/bin:${PATH}"

echo "=== Building Guest Kernel for SQL VM ==="
echo "Kernel source: ${KERNEL_DIR}/aosp"
echo "Build dir: ${BUILD_DIR}"
echo "Output: ${OUTPUT}"
echo "Clang: ${CLANG_DIR}"

mkdir -p "${BUILD_DIR}"

# Use microdroid_defconfig as base - it has virtio built-in
cd "${KERNEL_DIR}/aosp"

# Configure with clang
make O="${BUILD_DIR}" ARCH=arm64 \
    CC=clang \
    CROSS_COMPILE=aarch64-linux-gnu- \
    LLVM=1 \
    microdroid_defconfig

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
