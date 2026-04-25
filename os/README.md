# AutoButler Raspberry Pi OS Images

This directory contains everything needed to create a plug-and-play Raspberry Pi image with AutoButler pre-installed.

## What's Here

| Path | Description |
|------|-------------|
| `cloud-init/` | Cloud-init config for installing AutoButler on a stock Ubuntu image |
| `autobutler.pkr.hcl` | Packer template to build OS images (replaces build-image.sh). See `os/packer.md` for usage. |
| `pi-imager-catalog.json` | Raspberry Pi Imager catalog entry for custom OS list |

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

If you prefer to start from a stock Ubuntu Server image, see [cloud-init/README.md](cloud-init/README.md). This method requires an internet connection on first boot and takes 5–10 minutes for setup.

## Building Locally

### Requirements

- Packer v1.8+ installed
- `packer-plugin-arm-image` (see plugin installation below) or let `packer init` download required plugins
- arm64 Linux host, **or** x86_64 with QEMU binfmt_misc (`qemu-user-static`)
- Root privileges
- Packages: `xz-utils`, `kpartx`, `curl`, `jq`
- ~8GB free disk space per target

### Installing the arm-image Packer plugin

The build uses the `packer-plugin-arm-image` plugin. Upstream repository: https://github.com/solo-io/packer-plugin-arm-image

Option A — let Packer download plugins referenced in the HCL:

```bash
# Initializes modules and downloads any required plugins referenced in the HCL
packer init os/autobutler.pkr.hcl
# Verify the plugin is installed
packer plugins installed
```

Option B — manual install (replace <VERSION>, <os>, <arch> as appropriate):

```bash
mkdir -p ~/.config/packer/plugins/github.com/solo-io/arm-image
curl -L -o ~/.config/packer/plugins/github.com/solo-io/arm-image/packer-plugin-arm-image \
  https://github.com/solo-io/packer-plugin-arm-image/releases/download/<VERSION>/packer-plugin-arm-image_<VERSION>_<os>_<arch>
chmod +x ~/.config/packer/plugins/github.com/solo-io/arm-image/packer-plugin-arm-image
# Verify
packer plugins installed
```

### Build examples

```bash
# Build all targets
packer build os/autobutler.pkr.hcl

# Build only the arm image target for Raspberry Pi 4 (builder id example)
packer build -only=arm-image.pi4 os/autobutler.pkr.hcl

# Build only the arm image target for Raspberry Pi 5
packer build -only=arm-image.pi5 os/autobutler.pkr.hcl

# Build only the Odroid N2 target (see ODROID N2 notes below)
packer build -only=arm-image.odroid-n2 os/autobutler.pkr.hcl
```

Output images are written to this directory as `autobutler-pi4.img.xz` and `autobutler-pi5.img.xz` (and `autobutler-odroid-n2.img.xz` when building ODROID), with checksums in `checksums.sha256`.

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

Both Pi 4 and Pi 5 use the same Ubuntu Server 24.04 LTS arm64 base image. The images are functionally identical — separate builds exist for clarity and to allow future hardware-specific tuning.

The systemd service definition matches the one in `internal/install/system_service.go`.

## ODROID N2

- The ODROID N2 uses an ARM64-compatible base but requires different bootloader and device-tree handling compared to Raspberry Pi. The Packer template includes a target named `arm-image.odroid-n2` which produces an image tuned for ODROID N2 hardware.
- To build an ODROID image:

```bash
packer build -only=arm-image.odroid-n2 os/autobutler.pkr.hcl
```

- Note: cloud-init remains available as an alternative provisioning path. See the `cloud-init/` directory for example `user-data` and provisioning steps that can be applied to stock Ubuntu images.
