/*
variables.pkrvars.hcl

Per-board variables for Packer builds. Captures image URL, checksum, partition offsets,
and board identifier so a single Packer template can be parameterized for multiple
boards (pi4, pi5, odroid-n2).

Guidance:
 - Do NOT commit real long checksums or sensitive URLs if your CI keeps secrets.
   Prefer CI secrets or artifacts for large/secret values and provide them at build time.
 - Checksums shown are placeholders. Fill them at build time or via automation.
 - Partition offsets are given as example start sectors (in 512-byte sectors). Verify
   exact offsets from the image's partition table before using in production.
*/

# Select the target board. Can be overridden with -var 'board=odroid-n2' or via CI.
board = "pi4"

# Map of supported boards. Each board object contains:
#  - image_url        : URL to the official base image (prefer official vendor URLs)
#  - image_checksum   : SHA256 checksum prefixed as "sha256:..." (or other algorithm)
#  - partition_offsets: list of numeric start sectors for partitions used by the
#                       provisioning scripts (order matters; see template mapping below)
#  - board_id         : a short identifier used inside templates/provisioners
boards = {
  pi4 = {
    # Example official Raspberry Pi OS (32/64-bit) image location:
    # https://downloads.raspberrypi.org/raspios_lite_arm64/images/
    image_url      = "https://downloads.raspberrypi.org/raspios_lite_arm64/images/raspios_lite_arm64-2024-xx-xx.img"
    image_checksum = "sha256:REPLACE_WITH_ACTUAL_CHECKSUM"
    # Example partition start sectors (sectors of 512 bytes). Example values only.
    # Typical RasPi images: boot at 8192 (4 MiB), rootfs at 532480 (~260 MiB).
    partition_offsets = [8192, 532480]
    board_id = "raspberrypi4"
  }

  pi5 = {
    # Raspberry Pi 5 uses the same Raspberry Pi Foundation download page:
    # https://www.raspberrypi.com/software/operating-systems/
    image_url      = "https://downloads.raspberrypi.org/raspios_arm64/images/raspios_arm64-2024-xx-xx.img"
    image_checksum = "sha256:REPLACE_WITH_ACTUAL_CHECKSUM"
    # Example offsets; verify for the specific image used.
    partition_offsets = [8192, 532480]
    board_id = "raspberrypi5"
  }

  odroid-n2 = {
    # ODROID N2 official images are available from Hardkernel:
    # https://forum.odroid.com/viewforum.php?f=125 or https://odroid.com
    image_url      = "https://odroid.example.org/images/odroid-n2-ubuntu-xx-xx.img"
    image_checksum = "sha256:REPLACE_WITH_ACTUAL_CHECKSUM"
    # ODROID N2 images frequently use different partition layouts; example offsets:
    # eMMC/uSD boot may start at sector 2048 (1 MiB) and rootfs at 122880 (~60 MiB)
    partition_offsets = [2048, 122880]
    board_id = "odroid-n2"
  }
}

/*
How the Packer template should reference these values (example snippets):

locals {
  selected = var.boards[var.board]
}

source "file" "base_image" {
  url      = local.selected.image_url
  checksum = local.selected.image_checksum
}

build "example" {
  sources = ["source.file.base_image"]

  provisioner "shell" {
    # Example: pass partition offsets as a comma-separated string to the script
    environment_vars = ["PARTITION_OFFSETS=${join(",", local.selected.partition_offsets)}", "BOARD_ID=${local.selected.board_id}"]
    inline = ["/usr/local/bin/provision.sh"]
  }
}

Notes:
 - In HCL Packer templates, use local.selected (see above) so the same template
   works for all boards by changing the 'board' variable or supplying a different
   var-file in CI.
 - For CI, prefer injecting long URLs and checksums via secure variables or
   artifact storage and avoid committing them directly to the repo.
*/
