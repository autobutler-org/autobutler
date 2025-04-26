#!/bin/bash

# WireGuard monitoring script
# This script checks if WireGuard is running and working properly
# Usage: ./wireguard_monitor.sh

set -e

# Configuration variables
SERVER_IP="192.168.1.195"
SERVER_USER="brandonapol"
TEST_IP="10.0.0.1"  # WireGuard server virtual IP

# Function to check if the WireGuard server is running
check_server() {
  echo "Checking WireGuard server status..."
  if ssh $SERVER_USER@$SERVER_IP "sudo systemctl is-active wg-quick@wg0"; then
    echo "✅ WireGuard server is running"
    return 0
  else
    echo "❌ WireGuard server is not running"
    return 1
  fi
}

# Function to check connectivity through VPN
check_connectivity() {
  # Check if we can ping the WireGuard server virtual IP from the local machine
  if ping -c 3 $TEST_IP >/dev/null 2>&1; then
    echo "✅ VPN connectivity test passed"
    return 0
  else
    echo "❌ VPN connectivity test failed"
    return 1
  fi
}

# Function to restart the WireGuard server if it's down
restart_server() {
  echo "Restarting WireGuard server..."
  ssh $SERVER_USER@$SERVER_IP "sudo systemctl restart wg-quick@wg0"
  sleep 5
  if ssh $SERVER_USER@$SERVER_IP "sudo systemctl is-active wg-quick@wg0"; then
    echo "✅ WireGuard server restarted successfully"
  else
    echo "❌ Failed to restart WireGuard server"
  fi
}

# Function to check connected clients
check_connections() {
  echo "Checking connected clients..."
  ssh $SERVER_USER@$SERVER_IP "sudo wg show"
}

# Main execution
echo "Starting WireGuard monitoring..."

# Check server status
if ! check_server; then
  restart_server
fi

# Check local WireGuard status
if command -v wg >/dev/null 2>&1; then
  echo "Local WireGuard status:"
  wg show
else
  echo "WireGuard tools not installed locally"
fi

# Check connectivity
check_connectivity

# Check connected clients
check_connections

echo "WireGuard monitoring completed" 