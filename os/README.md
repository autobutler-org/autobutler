# AutoButler Raspberry Pi OS Images

This directory contains everything needed to create a plug-and-play Raspberry Pi image with AutoButler pre-installed.

## What's Here

| Path                     | Description                                                         |
| ------------------------ | ------------------------------------------------------------------- |
| `cloud-init/`            | Cloud-init config for installing AutoButler on a stock Ubuntu image |
| `pi-imager-catalog.json` | Raspberry Pi Imager catalog entry for custom OS list                |

## Quick Start: Flash the Pre-Built Image

The easiest way to get AutoButler running on a Raspberry Pi:

1. Download the latest image from [GitHub Releases](https://github.com/autobutler-org/autobutler/releases):
   - `autobutler-pi4.img.xz` for Raspberry Pi 4
   - `autobutler-pi5.img.xz` for Raspberry Pi 5
2. Open **Raspberry Pi Imager**
3. Click **Choose OS** → **Use custom** → select the downloaded `.img.xz` file
4. Choose your microSD card
5. Click **Write**
6. Insert the SD card into your Pi and power it on
7. Wait ~2 minutes for the first boot to complete
8. Open `http://autobutler.local` in your browser

### Using the Custom OS URL in Pi Imager

You can also add AutoButler to Pi Imager's OS list:

1. Open Raspberry Pi Imager
2. Click **Choose OS** → scroll to **Other**
3. Enter this URL when prompted: `https://raw.githubusercontent.com/autobutler-org/autobutler/main/os/pi-imager-catalog.json`

## Alternative: Cloud-Init on Stock Ubuntu

If you prefer to start from a stock Ubuntu Server image,
see [cloud-init/README.md](cloud-init/README.md). This method
requires an internet connection on first boot and takes 5–10
minutes for setup.

## Building Locally

This repository no longer includes a local image build path.

## First-Boot Experience

### Pre-built image (recommended)

1. Pi powers on and boots Ubuntu with AutoButler already installed
2. AutoButler service starts automatically
3. Avahi advertises `autobutler.local` via mDNS
4. User navigates to `http://autobutler.local` and sees the setup screen
5. Total time: ~1–2 minutes from power-on

### Cloud-init method

1. Pi powers on and runs Ubuntu first-boot
2. Cloud-init installs packages and downloads AutoButler (~5–10 minutes depending on internet speed)
3. AutoButler service starts
4. User navigates to `http://autobutler.local`

## Troubleshooting

### `autobutler.local` doesn't resolve

- Make sure your computer supports mDNS (macOS: built-in, Windows: install Bonjour, Linux: install `avahi-daemon`)
- Try accessing by IP address instead — check your router's DHCP client list for `autobutler`
- Wait a few minutes — the first boot may still be in progress

### AutoButler service isn't running

SSH into the Pi (default Ubuntu credentials: `ubuntu`/`ubuntu`) and check:

```bash
sudo systemctl status autobutler
sudo journalctl -u autobutler -f
cat /var/log/autobutler-setup.log  # cloud-init method only
```

### Cloud-init didn't complete

```bash
cloud-init status
cat /var/log/cloud-init-output.log
cat /var/log/autobutler-setup.log
```

### Firewall issues

The image configures `ufw` to allow only port 80 inbound. To allow SSH for debugging:

```bash
sudo ufw allow 22/tcp
```

## Architecture

Both Pi 4 and Pi 5 use the same Ubuntu Server 24.04 LTS
arm64 base image. The images are functionally identical -
separate builds exist for clarity and to allow future
hardware-specific tuning.

The systemd service definition matches the one in `internal/install/system_service.go`.

## ODROID N2

- The ODROID N2 uses an ARM64-compatible base but requires
   different bootloader and device-tree handling compared to
   Raspberry Pi.
- Cloud-init remains available as a provisioning path. See
   the `cloud-init/` directory for example `user-data` and
   provisioning steps that can be applied to stock Ubuntu
   images.
