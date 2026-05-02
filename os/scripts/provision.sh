#!/usr/bin/env bash
# Autobutler image provisioner
# Idempotent shell provisioner for local image population.
# Installs the Autobutler binary, creates/enables a systemd service, applies UFW rules,
# and adds an Avahi service definition.
#
# Features:
# - Idempotent: safe to run multiple times
# - Chroot/mounted-image support: pass --root /path/to/mounted/image to write into image without running systemctl or apt
# - Non-interactive: uses DEBIAN_FRONTEND=noninteractive when installing packages
#
# Usage examples:
#  - Run on a live system (as root): sudo /usr/local/bin/provision.sh
#  - Run against a mounted image: sudo ./provision.sh --root /mnt/image
#
# Notes for mounted-image usage:
#  - When --root is provided the script will not attempt to run systemctl or apt-get inside the target image; it will only write files and set ownership/permissions. After booting the image, run the script without --root or enable services manually on first boot.

set -euo pipefail
IFS=$'\n\t'

# Defaults
SERVICE_NAME="autobutler"
BINARY_NAME="autobutler"
DEST_BIN_DIR="/usr/local/bin"
SYSTEMD_DIR="/etc/systemd/system"
AVAHI_DIR="/etc/avahi/services"
UFW_ALLOWED_PORTS_DEFAULT="8080"
UFW_ADDITIONAL_RULES=("allow in on lo to any" "allow OpenSSH")

ROOT_DIR="/"

usage() {
  cat <<EOF
Usage: $0 [--root <path>] [--binary <path>] [--ports <comma-separated>]

Options:
  --root PATH        Write files under PATH (mounted image root). When set, package installs and systemctl operations are skipped.
  --binary PATH      Path to the compiled Autobutler binary to install. If omitted, the script searches ./build/${BINARY_NAME} and ./${BINARY_NAME}.
  --ports PORTS      Comma-separated list of TCP ports to open in UFW (default: ${UFW_ALLOWED_PORTS_DEFAULT}).

Examples:
  sudo $0 --binary ./autobutler
  sudo $0 --root /mnt/image --binary ./autobutler --ports 80,8080
EOF
}

# Basic arg parsing
BINARY_PATH=""
PORTS="${UFW_ALLOWED_PORTS_DEFAULT}"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --root)
      ROOT_DIR="$2"; shift 2;;
    --binary)
      BINARY_PATH="$2"; shift 2;;
    --ports)
      PORTS="$2"; shift 2;;
    --help|-h)
      usage; exit 0;;
    *)
      echo "Unknown arg: $1"; usage; exit 2;;
  esac
done

# Helpers to operate against root dir
rpath() { # join root with path
  local p="$1"
  if [[ "$ROOT_DIR" == "/" ]]; then
    printf "%s" "$p"
  else
    # strip leading slash from p
    p="${p#/}"
    printf "%s/%s" "${ROOT_DIR%/}" "$p"
  fi
}

run_chroot_action() {
  # Run a command against the image when ROOT_DIR is '/'
  if [[ "$ROOT_DIR" == "/" ]]; then
    "$@"
  else
    # When operating against a mounted image, do not attempt to run services or package managers inside it.
    echo "[info] Skipping runtime action in mounted image: $*"
  fi
}

# Logging
info() { echo "[info] $*"; }
warn() { echo "[warn] $*" >&2; }
err() { echo "[error] $*" >&2; exit 1; }

# Fail early if not root when running against real system
if [[ "$ROOT_DIR" == "/" && $(id -u) -ne 0 ]]; then
  err "Must be run as root when installing on the live system. Use --root when writing to a mounted image."
fi

# Resolve binary path if not provided
if [[ -z "$BINARY_PATH" ]]; then
  if [[ -x ./build/${BINARY_NAME} ]]; then
    BINARY_PATH="./build/${BINARY_NAME}"
  elif [[ -x ./${BINARY_NAME} ]]; then
    BINARY_PATH="./${BINARY_NAME}"
  else
    # don't error yet; allow running only file-write parts when binary absent if --root used
    BINARY_PATH=""
  fi
fi

if [[ -n "$BINARY_PATH" && ! -f "$BINARY_PATH" ]]; then
  err "Binary not found at $BINARY_PATH"
fi

info "ROOT_DIR=${ROOT_DIR}"
info "BINARY_PATH=${BINARY_PATH:-<not provided>}"
info "PORTS=${PORTS}"

# Create necessary directories
mkdir -p "$(rpath "$DEST_BIN_DIR")"
mkdir -p "$(rpath "$SYSTEMD_DIR")"
mkdir -p "$(rpath "$AVAHI_DIR")"

# Ensure autobutler system user exists (idempotent)
if [[ "$ROOT_DIR" == "/" ]]; then
  if ! id -u autobutler >/dev/null 2>&1; then
    info "Creating system user: autobutler"
    useradd --system --no-create-home --shell /usr/sbin/nologin autobutler
  else
    info "System user autobutler already exists"
  fi
fi

# Install binary (if provided)
if [[ -n "$BINARY_PATH" ]]; then
  info "Installing binary to ${DEST_BIN_DIR}/${BINARY_NAME}"
  # Use install to set mode atomically
  install -m 0755 -o root -g root "$BINARY_PATH" "$(rpath "$DEST_BIN_DIR/$BINARY_NAME")"
else
  warn "No binary provided; skipping binary install. Provide --binary <path> to install the compiled artifact."
fi

# Create systemd service unit (idempotent)
UNIT_PATH_REL="${SYSTEMD_DIR}/${SERVICE_NAME}.service"
UNIT_PATH="$(rpath "$UNIT_PATH_REL")"
info "Writing systemd unit to ${UNIT_PATH_REL}"
cat > "$UNIT_PATH.tmp" <<'UNIT'
[Unit]
Description=Autobutler service
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=autobutler
Group=autobutler
ExecStart=/usr/local/bin/autobutler serve
Restart=on-failure
RestartSec=5s
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
UNIT

# Only move if changed (idempotent)
if [[ -f "$UNIT_PATH" ]]; then
  if cmp -s "$UNIT_PATH.tmp" "$UNIT_PATH"; then
    info "Systemd unit unchanged"
    rm -f "$UNIT_PATH.tmp"
  else
    mv "$UNIT_PATH.tmp" "$UNIT_PATH"
    info "Systemd unit updated"
  fi
else
  mv "$UNIT_PATH.tmp" "$UNIT_PATH"
  info "Systemd unit created"
fi

# Ensure permissions
chmod 0644 "$UNIT_PATH"
chown root:root "$UNIT_PATH"

# UFW rules (idempotent)
# If running against mounted image, only write a note file that documents desired UFW rules
IFS=',' read -r -a PORT_ARR <<< "$PORTS"

if [[ "$ROOT_DIR" == "/" ]]; then
  # Install ufw if missing
  if ! command -v ufw >/dev/null 2>&1; then
    info "Installing ufw (non-interactive)"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq ufw || warn "ufw install failed; continuing"
  fi

  # Ensure UFW enabled and rules applied idempotently
  info "Enabling UFW and applying rules"
  # Allow loopback and SSH and user ports
  ufw --force enable || warn "ufw enable failed"
  ufw allow OpenSSH >/dev/null || true
  for p in "${PORT_ARR[@]}"; do
    p_trim=$(echo "$p" | tr -d '[:space:]')
    if [[ -n "$p_trim" ]]; then
      # Check if rule exists: parse ufw status numbered
      if ufw status numbered | grep -qw "${p_trim}/tcp"; then
        info "UFW rule for port ${p_trim}/tcp already exists"
      else
        ufw allow ${p_trim}/tcp >/dev/null || warn "ufw allow ${p_trim}/tcp failed"
        info "Added UFW rule for port ${p_trim}/tcp"
      fi
    fi
  done
else
  # Write a small descriptor file under the image explaining UFW rules to apply on first boot
  UFW_NOTE_PATH="$(rpath "/etc/${SERVICE_NAME}-ufw.rules.txt")"
  {
    echo "Autobutler UFW rules (apply on first boot):"
    echo "Allow OpenSSH"
    for p in "${PORT_ARR[@]}"; do
      echo "Allow TCP port: $(echo "$p" | tr -d '[:space:]')"
    done
  } > "$UFW_NOTE_PATH"
  chmod 0644 "$UFW_NOTE_PATH"
  chown root:root "$UFW_NOTE_PATH"
  info "Wrote UFW rule instructions to ${UFW_NOTE_PATH}"
fi

# Avahi service definition (idempotent)
AVAHI_SERVICE_PATH_REL="${AVAHI_DIR}/${SERVICE_NAME}.service"
AVAHI_SERVICE_PATH="$(rpath "$AVAHI_SERVICE_PATH_REL")"
info "Writing Avahi service definition to ${AVAHI_SERVICE_PATH_REL}"
cat > "$AVAHI_SERVICE_PATH.tmp" <<AVAHI
<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name replace-wildcards="yes">Autobutler on %h</name>
  <service>
    <type>_http._tcp</type>
    <port>${PORT_ARR[0]}</port>
  </service>
</service-group>
AVAHI

if [[ -f "$AVAHI_SERVICE_PATH" ]]; then
  if cmp -s "$AVAHI_SERVICE_PATH.tmp" "$AVAHI_SERVICE_PATH"; then
    info "Avahi service definition unchanged"
    rm -f "$AVAHI_SERVICE_PATH.tmp"
  else
    mv "$AVAHI_SERVICE_PATH.tmp" "$AVAHI_SERVICE_PATH"
    info "Avahi service definition updated"
  fi
else
  mv "$AVAHI_SERVICE_PATH.tmp" "$AVAHI_SERVICE_PATH"
  info "Avahi service definition created"
fi
chmod 0644 "$AVAHI_SERVICE_PATH"
chown root:root "$AVAHI_SERVICE_PATH"

# Install avahi-daemon package if running live system and avahi-daemon missing
if [[ "$ROOT_DIR" == "/" ]]; then
  if ! command -v avahi-daemon >/dev/null 2>&1; then
    info "Installing avahi-daemon"
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq avahi-daemon || warn "avahi-daemon install failed; continuing"
  fi
else
  info "Skipping avahi package install for mounted image"
fi

# Enable and start systemd unit if possible
can_use_systemctl=false
if [[ "$ROOT_DIR" == "/" && -x "/bin/systemctl" || -x "/usr/bin/systemctl" ]]; then
  # check if systemd is PID 1
  if [[ "$(ps -p 1 -o comm=)" == "systemd" ]]; then
    can_use_systemctl=true
  fi
fi

if [[ "$can_use_systemctl" == true ]]; then
  info "Reloading systemd daemon and enabling service"
  systemctl daemon-reload
  systemctl enable --now ${SERVICE_NAME}.service || warn "systemctl enable/start failed"
else
  warn "systemctl unavailable or not running; skipped enabling the service. If this is a mounted image, enable the service after first boot with: systemctl enable ${SERVICE_NAME}.service"
fi

info "Provisioning complete"
