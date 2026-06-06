#!/usr/bin/env bash
set -euo pipefail

#
# build-image.sh — Build a custom Ubuntu image for Raspberry Pi with cloud-init AutoButler setup.
#
# Usage:
#   sudo ./build-image.sh pi4
#   sudo ./build-image.sh pi5
#   sudo ./build-image.sh all
#
# This script creates customized Ubuntu Raspberry Pi images using cloud-init for first-boot setup.
# All configuration happens automatically on first boot, eliminating chroot complexity.
#
# Requirements:
#   - arm64 Linux host or macOS
#   - Root privileges (for loopback mount)
#   - Packages: xz-utils (or xz), curl, kpartx (Linux only)
#   - ~4GB free disk space per target
#

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORK_DIR="${SCRIPT_DIR}/build"
UBUNTU_IMAGE_URL="https://cdimage.ubuntu.com/releases/24.04/release/ubuntu-24.04.4-preinstalled-server-arm64+raspi.img.xz"
UBUNTU_IMAGE_XZ="${WORK_DIR}/ubuntu-base.img.xz"

log() { echo "[$(date -u '+%Y-%m-%d %H:%M:%S UTC')] $*"; }

cleanup() {
    log "Cleaning up..."
    if mountpoint -q "${WORK_DIR}/mnt" 2>/dev/null; then
        umount "${WORK_DIR}/mnt" 2>/dev/null || true
    fi
    if [ -n "${LOOP_DEV:-}" ] && [ -b "$LOOP_DEV" ]; then
        losetup -d "$LOOP_DEV" 2>/dev/null || true
    fi
}
trap cleanup EXIT

check_deps() {
    local missing=()
    for cmd in curl xz losetup mount; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done
    # kpartx is optional (only needed on Linux)
    if uname | grep -q Linux && ! command -v kpartx &>/dev/null; then
        missing+=("kpartx")
    fi
    if [ ${#missing[@]} -gt 0 ]; then
        log "ERROR: Missing required commands: ${missing[*]}"
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

    # Detect partition layout
    local root_part="${LOOP_DEV}p2"
    local boot_part="${LOOP_DEV}p1"

    # On some systems, partitions use different naming
    if [ ! -b "$root_part" ]; then
        if command -v kpartx &>/dev/null; then
            kpartx -av "$LOOP_DEV"
            sleep 1
            local loop_name
            loop_name=$(basename "$LOOP_DEV")
            root_part="/dev/mapper/${loop_name}p2"
            boot_part="/dev/mapper/${loop_name}p1"
        else
            log "ERROR: Could not find root partition and kpartx not available"
            exit 1
        fi
    fi

    log "Mounting root partition..."
    mkdir -p "${WORK_DIR}/mnt"
    mount "$root_part" "${WORK_DIR}/mnt"

    # Create cloud-init directory
    mkdir -p "${WORK_DIR}/mnt/etc/cloud/cloud.cfg.d"

    log "Injecting cloud-init configuration..."
    cat > "${WORK_DIR}/mnt/etc/cloud/cloud.cfg.d/99_autobutler.cfg" <<'CLOUD_INIT_EOF'
#cloud-config
# AutoButler setup via cloud-init on first boot

hostname: autobutler
fqdn: autobutler.local

users:
  - name: autobutler
    system: true
    shell: /usr/sbin/nologin
    home: /var/lib/autobutler

packages:
  - avahi-daemon
  - curl
  - jq

runcmd:
  # Create directories
  - mkdir -p /var/lib/autobutler
  - chown autobutler:autobutler /var/lib/autobutler

  # Download and install latest AutoButler release
  - |
    set -e
    RELEASE_URL=$(curl -fsSL "https://api.github.com/repos/autobutler-org/autobutler/releases/latest" \
      | jq -r '.assets[] | select(.name | test("Linux_arm64")) | .browser_download_url')
    if [ -z "$RELEASE_URL" ]; then
      echo "ERROR: Could not find AutoButler arm64 release"
      exit 1
    fi
    TEMP_DIR=$(mktemp -d)
    curl -fsSL -o "$TEMP_DIR/package.tar.gz" "$RELEASE_URL"
    tar -xzf "$TEMP_DIR/package.tar.gz" -C "$TEMP_DIR"
    cp "$TEMP_DIR/autobutler" /usr/local/bin/autobutler
    chmod 755 /usr/local/bin/autobutler
    rm -rf "$TEMP_DIR"

  # Create systemd service
  - tee /etc/systemd/system/autobutler.service > /dev/null <<'SERVICE_EOF'
[Unit]
Description=AutoButler Service
After=network.target

[Service]
User=autobutler
Group=autobutler
ExecStart=/usr/local/bin/autobutler serve
Environment="PORT=80"
Environment="GIN_MODE=release"
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
Restart=always
StandardOutput=append:/var/log/autobutler.app
StandardError=append:/var/log/autobutler.err

[Install]
WantedBy=multi-user.target
SERVICE_EOF

  # Enable services
  - systemctl daemon-reload
  - systemctl enable autobutler.service
  - systemctl enable avahi-daemon.service

  # Configure firewall
  - |
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow 80/tcp comment "AutoButler web UI"
    ufw --force enable

  # Sync filesystem
  - sync

power_state:
  mode: reboot
  message: "AutoButler setup complete, rebooting..."
  timeout: 10
CLOUD_INIT_EOF

    log "Unmounting filesystem..."
    umount "${WORK_DIR}/mnt"
    losetup -d "$LOOP_DEV"
    LOOP_DEV=""

    log "Compressing image (xz -6 for speed)..."
    xz -6 -T0 "$output"

    local checksum
    checksum=$(sha256sum "$output_xz" | awk '{print $1}')
    log "=== Build complete ==="
    log "Output: $output_xz"
    log "SHA256: $checksum"
    mkdir -p "$(dirname "${SCRIPT_DIR}/checksums.sha256")"
    echo "$checksum  $(basename "$output_xz")" >> "${SCRIPT_DIR}/checksums.sha256"
}

usage() {
    cat <<USAGE_EOF
Usage: sudo $0 <pi4|pi5|all>

Build a custom Ubuntu Raspberry Pi image with cloud-init AutoButler setup.

This script creates a minimalist image that uses cloud-init for first-boot
configuration. All setup (downloading binaries, installing services, etc.)
happens automatically when the image first boots on a Raspberry Pi.

Targets:
  pi4   Build image for Raspberry Pi 4
  pi5   Build image for Raspberry Pi 5
  all   Build images for both Pi 4 and Pi 5

Output:
  Images are written to: ${SCRIPT_DIR}/autobutler-{pi4,pi5}.img.xz
  Checksums are written to: ${SCRIPT_DIR}/checksums.sha256

Example:
  sudo $0 all
USAGE_EOF
    exit 1
}

if [ "$(id -u)" -ne 0 ]; then
    echo "ERROR: This script must be run as root (for loopback mounts)"
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
