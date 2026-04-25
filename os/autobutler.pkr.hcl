// Autobutler Packer template (HCL)
// Purpose: Declarative Packer template to build compressed images for
//   - autobutler-pi4.img.xz
//   - autobutler-pi5.img.xz
//   - autobutler-odroid-n2.img.xz
//
// Notes:
// - This template uses Packer HCL2. It targets QEMU (built-in) or a community
//   ARM image builder plugin. If using a plugin, install it before running
//   `packer build` (instructions below).
// - Shared provisioning is handled by calling: os/scripts/provision.sh
// - Post-build compression and checksum generation are executed with a
//   shell-local post-processor (so the final artifact is .img.xz and a .sha256)
//
// Plugin / Tools
// - QEMU builder: included with Packer core (good for reproducible builds using qemu)
// - Alternative ARM image plugin (example): https://github.com/arm-builder/packer-plugin-arm-image
//   If using a plugin, follow that repo's install instructions and run:
//     packer init os/autobutler.pkr.hcl
//   Or install a plugin by placing its binary in ~/.packer.d/plugins/
//
// How to build
// - Build all targets:  packer build os/autobutler.pkr.hcl
// - Build a single target: packer build -only=pi4 os/autobutler.pkr.hcl
//   (replace pi4 with pi5 or odroid-n2)
//
// Verification note (manual): After a successful build each target must
// produce a compressed image named e.g. autobutler-pi4.img.xz and a checksum
// file autobutler-pi4.img.xz.sha256 in the current working directory.

packer {
  required_version = "~> 1.15.0"
  required_plugins {
    qemu = {
      version = "~> 1.1.4"
      source  = "github.com/hashicorp/qemu"
    }
  }
}

// -----------------------
// Variables / locals
// -----------------------
variable "image_prefix" {
  type    = string
  default = "autobutler"
}

locals {
  // Naming template for outputs
  name_pi4       = "${var.image_prefix}-pi4"
  name_pi5       = "${var.image_prefix}-pi5"
  name_odroid_n2 = "${var.image_prefix}-odroid-n2"

  // Path to the common provision script (relative to os/ directory where packer build is run)
  provision_script = "${path.root}/scripts/provision.sh"
}

// -----------------------
// Shared sources
// -----------------------
// These define the QEMU builder configuration per target. Adapt paths and
// options according to the chosen builder or plugin (e.g. image URL, kernel,
// qemu-system-* binary, machine type, cpu, etc.). The settings below are
// intentionally minimal and meant as a starting point.

source "qemu" "pi4" {
  // QEMU builder options (tune for your environment)
  communicator     = "none"
  format           = "raw"
  output_directory = "output/pi4"
  disk_size        = 8192
  headless         = true
  iso_url          = "https://cdimage.ubuntu.com/releases/24.04/release/ubuntu-24.04-preinstalled-server-arm64+raspi.img.xz"
  iso_checksum     = "none"
  // Consider adding: qemu_binary, qemuargs, boot_command, etc.
}

source "qemu" "pi5" {
  communicator     = "none"
  format           = "raw"
  output_directory = "output/pi5"
  disk_size        = 16384
  iso_url          = "https://cdimage.ubuntu.com/releases/24.04/release/ubuntu-24.04-preinstalled-server-arm64+raspi.img.xz"
  iso_checksum     = "none"
  headless         = true
}

source "qemu" "odroid-n2" {
  communicator     = "none"
  format           = "raw"
  output_directory = "output/odroid-n2"
  disk_size        = 12288
  iso_url          = "https://cdimage.ubuntu.com/releases/24.04/release/ubuntu-24.04-preinstalled-server-arm64+raspi.img.xz"
  iso_checksum     = "none"
  headless         = true
}

// -----------------------
// Shared provisioner (invokes the repo script)
// -----------------------
// The builds below all call the same local provision script (os/scripts/provision.sh).
// The script is expected to implement the full provisioning steps (install packages,
// configure users, set up services, clean up, etc.). Keep provisioning idempotent.

// -----------------------
// Build targets
// -----------------------

build {
  name    = "pi4"
  sources = ["source.qemu.pi4"]

  // Shared provisioner usage (calls the shared script)
  provisioner "shell" {
    script = local.provision_script
    // run as root (default) — change with `execute_command` if needed
  }

  // After the builder finishes, run a shell-local post-processor to compress
  // the raw image into .img.xz and generate a sha256 checksum. The command
  // below uses a best-effort approach: it looks for the raw image produced
  // under the output directory and compresses it into the required name.
  post-processor "shell-local" {
    inline = [
      // locate the produced .img or .raw file, compress to .img.xz, and generate checksum
      "set -e",
      "OUT_DIR=output/pi4; IMG=$(find \"${PWD}/$OUT_DIR\" -maxdepth 1 -type f -name '*.img' -o -name '*.raw' | head -n1)",
      "if [ -z \"$IMG\" ]; then echo 'ERROR: produced image not found in' $OUT_DIR; exit 1; fi",
      "DEST=\"${local.name_pi4}.img.xz\"",
      "xz -T0 -9 -c \"$IMG\" > \"$DEST\"",
      "sha256sum \"$DEST\" | awk '{print $1\"  \" $2}' > \"${local.name_pi4}.img.xz.sha256\"",
      "echo 'Produced:' \"$DEST\"",
    ]
  }
}

build {
  name    = "pi5"
  sources = ["source.qemu.pi5"]

  provisioner "shell" {
    script = local.provision_script
  }

  post-processor "shell-local" {
    inline = [
      "set -e",
      "OUT_DIR=output/pi5; IMG=$(find \"$${PWD}/$OUT_DIR\" -maxdepth 1 -type f -name '*.img' -o -name '*.raw' | head -n1)",
      "if [ -z \"$IMG\" ]; then echo 'ERROR: produced image not found in' $OUT_DIR; exit 1; fi",
      "DEST=\"${local.name_pi5}.img.xz\"",
      "xz -T0 -9 -c \"$IMG\" > \"$DEST\"",
      "sha256sum \"$DEST\" | awk '{print $1\"  \" $2}' > \"${local.name_pi5}.img.xz.sha256\"",
      "echo 'Produced:' \"$DEST\"",
    ]
  }
}

build {
  name    = "odroid-n2"
  sources = ["source.qemu.odroid-n2"]

  provisioner "shell" {
    script = local.provision_script
  }

  post-processor "shell-local" {
    inline = [
      "set -e",
      "OUT_DIR=output/odroid-n2; IMG=$(find \"$${PWD}/$OUT_DIR\" -maxdepth 1 -type f -name '*.img' -o -name '*.raw' | head -n1)",
      "if [ -z \"$IMG\" ]; then echo 'ERROR: produced image not found in' $OUT_DIR; exit 1; fi",
      "DEST=\"${local.name_odroid_n2}.img.xz\"",
      "xz -T0 -9 -c \"$IMG\" > \"$DEST\"",
      "sha256sum \"$DEST\" | awk '{print $1\"  \" $2}' > \"${local.name_odroid_n2}.img.xz.sha256\"",
      "echo 'Produced:' \"$DEST\"",
    ]
  }
}
