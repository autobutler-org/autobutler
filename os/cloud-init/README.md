# AutoButler Cloud-Init Configuration

This directory contains cloud-init files for installing AutoButler on a fresh Ubuntu Server 24.04 LTS image for Raspberry Pi.

## What This Does

On first boot, the cloud-init configuration will:

1. Set the hostname to `autobutler`
2. Install required packages (`avahi-daemon`, `ufw`, `udisks2`, `curl`, `jq`)
3. Download the latest AutoButler binary from GitHub Releases
4. Create the `autobutler` service user and data directory
5. Install and enable the systemd service
6. Configure the firewall (allow port 80 only)
7. Enable mDNS so the device is reachable at `http://autobutler.local`

## How to Use with Raspberry Pi Imager

### Method 1: Custom Image (Recommended)

Use the pre-built AutoButler image from the [releases page](https://github.com/autobutler-org/autobutler/releases). See the parent [os/README.md](../README.md) for instructions.

### Method 2: Cloud-Init on Stock Ubuntu

If you prefer to start from a stock Ubuntu image:

1. Open **Raspberry Pi Imager**
2. Choose **Ubuntu Server 24.04 LTS (64-bit)** as the OS
3. Choose your SD card as the target
4. Click the **gear icon** (⚙️) for advanced options
5. Flash the image to your SD card
6. After flashing, mount the `system-boot` partition
7. Copy `user-data` and `meta-data` from this directory to the `system-boot` partition, replacing the existing files
8. Eject the SD card and insert it into your Raspberry Pi
9. Power on and wait 5–10 minutes for the first-boot setup to complete
10. Navigate to `http://autobutler.local` in your browser

### Monitoring First-Boot Progress

If you have a monitor connected or SSH access:

```bash
tail -f /var/log/autobutler-setup.log
```

## Files

| File | Purpose |
|------|---------|
| `user-data` | Cloud-init config — installs and configures AutoButler |
| `meta-data` | Minimal instance metadata (hostname) |

## Requirements

- Raspberry Pi 4 or 5
- microSD card (8GB minimum, 16GB+ recommended)
- Internet connection on first boot (to download the AutoButler binary)
- Ubuntu Server 24.04 LTS arm64 base image
