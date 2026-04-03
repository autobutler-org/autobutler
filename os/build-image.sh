#!/usr/bin/env bash
set -euo pipefail

#
# build-image.sh — Build a custom Ubuntu image for Raspberry Pi with AutoButler pre-installed.
#
# Usage:
#   sudo ./build-image.sh pi4
#   sudo ./build-image.sh pi5
#   sudo ./build-image.sh all
#
# Requirements:
#   - arm64 Linux host (or x86_64 with QEMU binfmt_misc registered)
#   - Root privileges (for loopback mount and chroot)
#   - Packages: xz-utils, qemu-user-static (if cross-arch), curl, jq, kpartx
#   - ~8GB free disk space per target
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="${SCRIPT_DIR}/build"
UBUNTU_IMAGE_URL="https://cdimage.ubuntu.com/releases/24.04/release/ubuntu-24.04.4-preinstalled-server-arm64+raspi.img.xz"
UBUNTU_IMAGE_XZ="${WORK_DIR}/ubuntu-base.img.xz"
AUTOBUTLER_REPO="autobutler-org/autobutler"

log() { echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $*"; }

cleanup() {
    log "Cleaning up..."
    if mountpoint -q "${WORK_DIR}/mnt/proc" 2>/dev/null; then umount "${WORK_DIR}/mnt/proc" || true; fi
    if mountpoint -q "${WORK_DIR}/mnt/sys" 2>/dev/null; then umount "${WORK_DIR}/mnt/sys" || true; fi
    if mountpoint -q "${WORK_DIR}/mnt/dev/pts" 2>/dev/null; then umount "${WORK_DIR}/mnt/dev/pts" || true; fi
    if mountpoint -q "${WORK_DIR}/mnt/dev" 2>/dev/null; then umount "${WORK_DIR}/mnt/dev" || true; fi
    if mountpoint -q "${WORK_DIR}/mnt/boot/firmware" 2>/dev/null; then umount "${WORK_DIR}/mnt/boot/firmware" || true; fi
    if mountpoint -q "${WORK_DIR}/mnt" 2>/dev/null; then umount "${WORK_DIR}/mnt" || true; fi
    if [ -n "${LOOP_DEV:-}" ]; then losetup -d "$LOOP_DEV" 2>/dev/null || true; fi
}
trap cleanup EXIT

check_deps() {
    local missing=()
    for cmd in curl jq xz losetup mount chroot kpartx; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done
    if [ ${#missing[@]} -gt 0 ]; then
        log "ERROR: Missing required commands: ${missing[*]}"
        log "Install them with: apt-get install -y xz-utils kpartx curl jq qemu-user-static"
        exit 1
    fi
}

download_base_image() {
    mkdir -p "$WORK_DIR"
    if [ ! -f "$UBUNTU_IMAGE_XZ" ]; then
        log "Downloading Ubuntu Server 24.04 LTS arm64+raspi image..."
        curl -fSL -o "$UBUNTU_IMAGE_XZ" "$UBUNTU_IMAGE_URL"
    else
        log "Base image already downloaded, skipping."
    fi
}

fetch_autobutler_binary() {
    local dest="$1"
    log "Fetching latest AutoButler release for arm64..."
    local url
    url=$(curl -fsSL "https://api.github.com/repos/${AUTOBUTLER_REPO}/releases/latest" \
        | jq -r '.assets[] | select(.name | test("Linux_arm64")) | .browser_download_url')
    if [ -z "$url" ]; then
        log "ERROR: Could not find arm64 release asset"
        exit 1
    fi
    log "Downloading: $url"
    local temp_dir="${HOME}/tmp/autobutler_extract"
    mkdir -p "$temp_dir"
    curl -fsSL -o "$temp_dir/package.tar.gz" "$url"
    tar -xzf "$temp_dir/package.tar.gz" -C "$temp_dir"
    # Find the autobutler binary (may be at root or in a subdirectory)
    if [ -f "$temp_dir/autobutler" ]; then
        cp "$temp_dir/autobutler" "$dest"
    elif [ -f "$temp_dir/autobutler/"* ]; then
        cp "$temp_dir"/autobutler/* "$dest"
    else
        log "ERROR: Could not find autobutler binary in tarball"
        exit 1
    fi
    rm -rf "$temp_dir"
    chmod 755 "$dest"
}

build_image() {
    local target="$1"
    local output="${SCRIPT_DIR}/autobutler-${target}.img"
    local output_xz="${output}.xz"

    log "=== Building AutoButler image for ${target} ==="

    download_base_image

    log "Decompressing base image..."
    xz -dk "$UBUNTU_IMAGE_XZ" -c > "$output"

    log "Setting up loopback device..."
    LOOP_DEV=$(losetup --find --show --partscan "$output")
    log "Loop device: $LOOP_DEV"

    sleep 1

    local root_part="${LOOP_DEV}p2"
    local boot_part="${LOOP_DEV}p1"

    if [ ! -b "$root_part" ]; then
        kpartx -av "$LOOP_DEV"
        sleep 1
        local loop_name
        loop_name=$(basename "$LOOP_DEV")
        root_part="/dev/mapper/${loop_name}p2"
        boot_part="/dev/mapper/${loop_name}p1"
    fi

    log "Mounting root partition..."
    mkdir -p "${WORK_DIR}/mnt"
    mount "$root_part" "${WORK_DIR}/mnt"

    if [ -b "$boot_part" ]; then
        log "Mounting boot partition..."
        mkdir -p "${WORK_DIR}/mnt/boot/firmware"
        mount "$boot_part" "${WORK_DIR}/mnt/boot/firmware"
    fi

    log "Setting up chroot bind mounts..."
    mount --bind /dev "${WORK_DIR}/mnt/dev"
    mount --bind /dev/pts "${WORK_DIR}/mnt/dev/pts"
    mount --bind /proc "${WORK_DIR}/mnt/proc"
    mount --bind /sys "${WORK_DIR}/mnt/sys"

    cp /etc/resolv.conf "${WORK_DIR}/mnt/etc/resolv.conf" 2>/dev/null || true

    log "Downloading AutoButler binary..."
    fetch_autobutler_binary "${WORK_DIR}/mnt/usr/local/bin/autobutler"

    log "Installing systemd service..."
    cat > "${WORK_DIR}/mnt/etc/systemd/system/autobutler.service" <<'EOF'
[Unit]
Description=AutoButler Service
After=network.target

[Service]
User=autobutler
Group=autobutler
ExecStart=/usr/local/bin/autobutler serve
Environment="PORT=80"
Environment="GIN_MODE=release"
Restart=always
StandardOutput=append:/var/log/autobutler.app
StandardError=append:/var/log/autobutler.err

[Install]
WantedBy=multi-user.target
EOF

    log "Installing sudoers rule..."
    cat > "${WORK_DIR}/mnt/etc/sudoers.d/autobutler" <<'EOF'
autobutler ALL=(root) NOPASSWD: /bin/mount * /var/lib/autobutler/mounts/*, /bin/umount /var/lib/autobutler/mounts/*
EOF
    chmod 440 "${WORK_DIR}/mnt/etc/sudoers.d/autobutler"

    log "Running chroot setup..."
    chroot "${WORK_DIR}/mnt" /bin/bash -e <<'CHROOT_SCRIPT'
export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y --no-install-recommends avahi-daemon ufw udisks2 curl

useradd --system --no-create-home --shell /usr/sbin/nologin --comment "AutoButler service account" autobutler 2>/dev/null || true

mkdir -p /var/lib/autobutler
chown autobutler:autobutler /var/lib/autobutler

echo "autobutler" > /etc/hostname
sed -i 's/127\.0\.1\.1.*/127.0.1.1\tautobutler/' /etc/hosts || echo "127.0.1.1	autobutler" >> /etc/hosts

systemctl enable autobutler.service
systemctl enable avahi-daemon.service

ufw default deny incoming
ufw default allow outgoing
ufw allow 80/tcp comment "AutoButler web UI"
ufw --force enable

apt-get clean
rm -rf /var/lib/apt/lists/*
CHROOT_SCRIPT

    log "Unmounting chroot bind mounts..."
    umount "${WORK_DIR}/mnt/proc" || true
    umount "${WORK_DIR}/mnt/sys" || true
    umount "${WORK_DIR}/mnt/dev/pts" || true
    umount "${WORK_DIR}/mnt/dev" || true

    if mountpoint -q "${WORK_DIR}/mnt/boot/firmware" 2>/dev/null; then
        umount "${WORK_DIR}/mnt/boot/firmware" || true
    fi
    umount "${WORK_DIR}/mnt"

    losetup -d "$LOOP_DEV"
    LOOP_DEV=""

    log "Compressing image..."
    xz -9 -T0 "$output"

    local checksum
    checksum=$(sha256sum "$output_xz" | awk '{print $1}')
    log "=== Build complete ==="
    log "Output: $output_xz"
    log "SHA256: $checksum"
    echo "$checksum  $(basename "$output_xz")" >> "${SCRIPT_DIR}/checksums.sha256"
}

usage() {
    echo "Usage: sudo $0 <pi4|pi5|all>"
    echo ""
    echo "Build a custom Raspberry Pi image with AutoButler pre-installed."
    echo ""
    echo "Targets:"
    echo "  pi4   Build image for Raspberry Pi 4"
    echo "  pi5   Build image for Raspberry Pi 5"
    echo "  all   Build images for both Pi 4 and Pi 5"
    exit 1
}

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: This script must be run as root (for loopback mounts and chroot)"
    exit 1
fi

check_deps

TARGET="${1:-}"
case "$TARGET" in
    pi4)  build_image pi4 ;;
    pi5)  build_image pi5 ;;
    all)  build_image pi4; build_image pi5 ;;
    *)    usage ;;
esac

log "All done. Checksums written to ${SCRIPT_DIR}/checksums.sha256"
