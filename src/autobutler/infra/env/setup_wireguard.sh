#!/bin/bash

# Setup WireGuard VPN on Raspberry Pi and generate client configs
# Usage: ./setup_wireguard.sh

set -e

# Configuration variables
SERVER_IP="192.168.1.195"  # Primary Pi will be the WireGuard server
SERVER_USER="brandonapol"
SERVER_PUBLIC_IP=""  # Will be determined automatically
SERVER_PORT="51820"
SERVER_INTERFACE="wg0"
CLIENT_NAMES=("phone" "macbook")
HOME_NETWORK="192.168.1.0/24"  # Your home network subnet
WG_SUBNET="10.0.0.0/24"        # WireGuard virtual network subnet

# Function to install WireGuard on the Pi server
install_wireguard() {
  echo "Installing WireGuard on $SERVER_IP..."
  ssh $SERVER_USER@$SERVER_IP "sudo apt update && sudo apt install -y wireguard"
}

# Function to enable IP forwarding on server
enable_ip_forwarding() {
  echo "Enabling IP forwarding on server..."
  ssh $SERVER_USER@$SERVER_IP "sudo sh -c 'echo net.ipv4.ip_forward=1 > /etc/sysctl.d/99-wireguard.conf' && sudo sysctl -p /etc/sysctl.d/99-wireguard.conf"
}

# Function to generate key pairs
generate_keypair() {
  local name=$1
  local dir=$2
  
  mkdir -p "$dir"
  
  echo "Generating key pair for $name..."
  wg genkey | tee "$dir/${name}_private.key" | wg pubkey > "$dir/${name}_public.key"
  
  echo "Keys generated for $name"
}

# Generate server and client configs
generate_configs() {
  local config_dir="./wireguard_configs"
  mkdir -p "$config_dir"
  
  # Generate server keypair
  generate_keypair "server" "$config_dir"
  SERVER_PRIVATE_KEY=$(cat "$config_dir/server_private.key")
  SERVER_PUBLIC_KEY=$(cat "$config_dir/server_public.key")
  
  # Get public IP
  SERVER_PUBLIC_IP=$(curl -s https://ifconfig.me)
  
  # Create server config
  echo "Generating server config..."
  cat > "$config_dir/wg0.conf" << EOF
[Interface]
Address = 10.0.0.1/24
PrivateKey = $SERVER_PRIVATE_KEY
ListenPort = $SERVER_PORT
PostUp = iptables -A FORWARD -i $SERVER_INTERFACE -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i $SERVER_INTERFACE -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
EOF
  
  # Generate client configs
  local client_ip_last=2
  for client in "${CLIENT_NAMES[@]}"; do
    generate_keypair "$client" "$config_dir"
    local client_private_key=$(cat "$config_dir/${client}_private.key")
    local client_public_key=$(cat "$config_dir/${client}_public.key")
    
    # Add client to server config
    cat >> "$config_dir/wg0.conf" << EOF

[Peer]
# $client
PublicKey = $client_public_key
AllowedIPs = 10.0.0.${client_ip_last}/32
EOF
    
    # Create client config
    echo "Generating config for $client..."
    cat > "$config_dir/${client}.conf" << EOF
[Interface]
PrivateKey = $client_private_key
Address = 10.0.0.${client_ip_last}/24
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = $SERVER_PUBLIC_KEY
Endpoint = ${SERVER_PUBLIC_IP}:${SERVER_PORT}
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
EOF
    
    # Generate QR code for mobile clients
    if [[ "$client" == "phone" ]]; then
      echo "Generating QR code for phone config..."
      if ! command -v qrencode &> /dev/null; then
        brew install qrencode
      fi
      qrencode -t png -o "$config_dir/${client}_qr.png" < "$config_dir/${client}.conf"
      echo "QR code saved to $config_dir/${client}_qr.png"
    fi
    
    ((client_ip_last++))
  done
}

# Deploy server config to Pi
deploy_server_config() {
  echo "Deploying WireGuard server config to Pi..."
  local config_dir="./wireguard_configs"
  
  # Copy server config
  scp "$config_dir/wg0.conf" $SERVER_USER@$SERVER_IP:/tmp/wg0.conf
  ssh $SERVER_USER@$SERVER_IP "sudo mv /tmp/wg0.conf /etc/wireguard/ && sudo chmod 600 /etc/wireguard/wg0.conf"
  
  # Enable and start WireGuard
  ssh $SERVER_USER@$SERVER_IP "sudo systemctl enable wg-quick@wg0 && sudo systemctl start wg-quick@wg0"
  
  echo "WireGuard server configured and started"
}

# Setup local client (MacBook)
setup_local_client() {
  echo "Setting up WireGuard client on local machine..."
  local config_dir="./wireguard_configs"
  
  # Create path for WireGuard config
  local wg_path="$HOME/Library/Application Support/WireGuard/Tunnels"
  mkdir -p "$wg_path"
  
  # Copy config
  cp "$config_dir/macbook.conf" "$wg_path/home_vpn.conf"
  
  echo "WireGuard client configured on local machine"
  echo "You can now activate the 'home_vpn' tunnel from the WireGuard app"
}

# Main execution
echo "Starting WireGuard setup..."

# Install on Pi (server)
install_wireguard

# Enable IP forwarding on server
enable_ip_forwarding

# Generate all configs
generate_configs

# Deploy configs
deploy_server_config
setup_local_client

echo "WireGuard setup completed successfully!"
echo "Instructions:"
echo "1. On your phone, scan the QR code at ./wireguard_configs/phone_qr.png using the WireGuard app"
echo "2. On your MacBook, the WireGuard tunnel 'home_vpn' has been configured"
echo "3. Make sure your router forwards UDP port $SERVER_PORT to Pi ($SERVER_IP)"
echo ""
echo "Your WireGuard server public key: $SERVER_PUBLIC_KEY"
echo "Your WireGuard server public IP: $SERVER_PUBLIC_IP"
echo "Your WireGuard server port: $SERVER_PORT" 