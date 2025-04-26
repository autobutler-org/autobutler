# WireGuard VPN Setup for Home Network

This directory contains scripts to set up a WireGuard VPN server on a Raspberry Pi, allowing secure remote access to your home network.

## Prerequisites

- One Raspberry Pi device (192.168.1.195)
- WireGuard app installed on your mobile device
- WireGuard app installed on your local macOS machine
- SSH access to the Raspberry Pi

## Setup

### Quick Setup

Run the setup script to automatically configure WireGuard:

```bash
./setup_wireguard.sh
```

This script will:

1. Install WireGuard on the Raspberry Pi
2. Configure the Pi as the WireGuard server
3. Generate client configurations for your phone and macOS device
4. Generate a QR code for easy setup on your phone

### Router Configuration

To access your VPN from outside your home network, you need to:

1. Configure your router to forward UDP port 51820 to the Pi (192.168.1.195)
2. Ensure your router has a static public IP or use a dynamic DNS service

## Usage

### Managing WireGuard from Your MacBook

You can use the following commands to manage WireGuard:

```bash
# Start the VPN connection
wireguard up

# Stop the VPN connection
wireguard down

# Restart the VPN connection
wireguard restart

# Check VPN status
wireguard status
# or
vpn_status
```

### Monitoring

To check the health of your WireGuard setup:

```bash
wireguard_monitor
```

This will:

- Check if WireGuard is running on the Raspberry Pi
- Verify connectivity through the VPN
- Show connected clients
- Attempt to restart services if they're down

## Configuration Details

WireGuard uses the following IP ranges:

- WireGuard server (Pi): 10.0.0.1/24
- Clients: 10.0.0.2/24 onwards

Client configs are stored in:

- `./wireguard_configs/phone.conf`
- `./wireguard_configs/macbook.conf`

## Troubleshooting

If you encounter issues:

1. Check if the WireGuard service is running on the Pi:

   ```
   ssh brandonapol@192.168.1.195 "sudo systemctl status wg-quick@wg0"
   ```

2. Verify port forwarding on your router is correctly configured

3. Make sure your firewall allows UDP port 51820

4. Check connectivity with:

   ```
   ping 10.0.0.1
   ```

5. Run the monitoring script to detect and fix common issues:
   ```
   wireguard_monitor
   ```

## Security Considerations

- Keep your private keys secure
- Regularly update your Raspberry Pi
- Consider changing the default port (51820) to a non-standard port
- Restrict SSH access to your Raspberry Pi
