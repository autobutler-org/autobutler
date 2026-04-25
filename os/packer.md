# Packer-based image builds

This repository uses HashiCorp Packer to build Raspberry Pi OS images for AutoButler.
The legacy `os/build-image.sh` script has been retired in favor of a declarative Packer template: `os/autobutler.pkr.hcl`.

Quick commands:

1. Install Packer
   - macOS: `brew tap hashicorp/tap && brew install hashicorp/tap/packer`
   - Linux: download from https://www.packer.io/downloads

2. Initialize and build

```bash
packer init os/autobutler.pkr.hcl
# Build a single target (pi4 or pi5)
packer build -only=pi4 os/autobutler.pkr.hcl
# Build all configured targets
packer build os/autobutler.pkr.hcl
```

CI notes:
- In CI, install Packer and any required builder plugins, run `packer init`, then `packer build` and upload the produced artifacts and checksums as workflow artifacts.
- Packer is more CI-friendly than the old loopback/chroot script because it isolates builders, is declarative, and supports reproducible artifacts.

For details on the template and provisioner scripts, see `os/` and the repository's design docs.
