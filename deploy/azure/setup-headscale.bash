#!/bin/bash
# AutoButler Headscale + Provisioning Service — VM Setup Script
# Runs as root on first boot via Azure CustomScript extension.
# Rendered into headscale.rendered.parameters.json by: make render/headscale
set -euox pipefail

# ── Variables substituted by `make render/headscale` ──────────────────────
DOMAIN="${HEADSCALE_DOMAIN}"
ADMIN_EMAIL='admin@autobutler.org'

# ── Helpers ────────────────────────────────────────────────────────────────
log() { echo "[setup] $*"; }
retry() {
  local n=0
  until [ "$n" -ge 5 ]; do
    "$@" && return 0
    n=$((n+1))
    log "Retrying ($n/5)..."
    sleep 5
  done
  return 1
}

# ── 1. System packages ─────────────────────────────────────────────────────
log "Installing system packages..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
  curl wget jq git nginx certbot python3-certbot-nginx \
  ca-certificates gnupg lsb-release

# ── 2. Go ──────────────────────────────────────────────────────────────────
log "Installing Go..."
GO_VERSION=1.22.3
wget -q -O /tmp/go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
rm -rf /usr/local/go
tar -C /usr/local -xzf /tmp/go.tar.gz
export PATH=/usr/local/go/bin:$PATH
export GOPATH=/root/go
export GOMODCACHE=/root/go/pkg/mod
go version

# ── 3. Headscale ───────────────────────────────────────────────────────────
log "Installing Headscale..."
HS_VERSION=v0.28.0
HS_VER="${HS_VERSION#v}"
retry wget -q -O /tmp/headscale.deb \
  "https://github.com/juanfont/headscale/releases/download/${HS_VERSION}/headscale_${HS_VER}_linux_amd64.deb"
dpkg -i /tmp/headscale.deb

mkdir -p /etc/headscale /var/lib/headscale

cat > /etc/headscale/config.yaml <<EOF
server_url: https://${DOMAIN}
listen_addr: 127.0.0.1:8080
grpc_listen_addr: 127.0.0.1:50443
metrics_listen_addr: 127.0.0.1:9090
log:
  level: info
db_type: sqlite3
db_path: /var/lib/headscale/db.sqlite
private_key_path: /var/lib/headscale/private.key
noise:
  private_key_path: /var/lib/headscale/noise_private.key
ip_prefixes:
  - 100.64.0.0/10
dns_config:
  override_local_dns: true
  base_domain: autobutler.net
  nameservers:
    - 1.1.1.1
derp:
  server:
    enabled: false
  urls:
    - https://controlplane.tailscale.com/derpmap/default
EOF

systemctl enable headscale
systemctl start headscale
log "Headscale started."

# ── 4. Nginx (HTTP only first — TLS after DNS propagates) ──────────────────
log "Configuring Nginx..."
rm -f /etc/nginx/sites-enabled/default

cat > /etc/nginx/sites-available/headscale <<EOF
server {
    listen 80;
    server_name ${DOMAIN};
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF
ln -sf /etc/nginx/sites-available/headscale /etc/nginx/sites-enabled/headscale
nginx -t
systemctl reload nginx
log "Nginx configured (HTTP only, TLS pending DNS)."

# ── 5. TLS — non-fatal; re-run manually once DNS is live ──────────────────
log "Attempting TLS certificate (non-fatal if DNS not propagated yet)..."
certbot --nginx -d "${DOMAIN}" \
  --non-interactive --agree-tos -m "${ADMIN_EMAIL}" \
  --redirect \
  && log "TLS certificate issued." \
  || log "WARNING: certbot failed. Once DNS points to this IP, run: sudo certbot --nginx -d ${DOMAIN} --non-interactive --agree-tos -m ${ADMIN_EMAIL} --redirect"

# ── 6. AutoButler provisioning service ───

rm -rf /opt/autobutler-src
git clone --depth 1 --branch main https://github.com/autobutler-org/autobutler.git /opt/autobutler-src

cd /opt/autobutler-src
GOPATH=/root/go GOMODCACHE=/root/go/pkg/mod GOCACHE=/root/go/cache \
  /usr/local/go/bin/go build -o /usr/local/bin/autobutler-provisioning ./cmd/provisioning/
log "Provisioning binary installed at /usr/local/bin/autobutler-provisioning"

# ── 7. Provisioning service systemd unit ──────────────────────────────────
cat > /etc/systemd/system/autobutler-provisioning.service <<'UNIT'
[Unit]
Description=AutoButler Provisioning Service
After=network.target headscale.service
Requires=headscale.service

[Service]
Type=simple
User=headscale
Environment=PORT=8081
Environment=HEADSCALE_URL=http://127.0.0.1:8080
# Set HEADSCALE_API_KEY post-boot:
#   sudo headscale apikeys create --expiration 9999d
#   echo "HEADSCALE_API_KEY=<key>" | sudo tee /etc/autobutler/provisioning.env
EnvironmentFile=-/etc/autobutler/provisioning.env
ExecStart=/usr/local/bin/autobutler-provisioning
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

mkdir -p /etc/autobutler
touch /etc/autobutler/provisioning.env
chown headscale:headscale /etc/autobutler/provisioning.env
chmod 600 /etc/autobutler/provisioning.env

systemctl daemon-reload
systemctl enable autobutler-provisioning
# Not started yet — API key required first (see post-deploy steps in README)

log "Setup complete. Public IP: $(curl -s ifconfig.me)"
log ""
log "Post-deploy steps:"
log "  1. Create API key: sudo headscale apikeys create --expiration 9999d"
log "  2. Store key: echo 'HEADSCALE_API_KEY=<key>' | sudo tee /etc/autobutler/provisioning.env"
log "  3. Start provisioning: sudo systemctl start autobutler-provisioning"
