# Builder for the Linux AppImage on a host that cannot build it natively, which in practice
# is a Mac. Nothing here ships: `make build/frontend/linux/appimage` builds this image on
# macOS and runs the same target inside it. Linux hosts and CI never use it.
#
# Ubuntu 22.04 on linux/amd64 because that is what linux/AppImageBuilder.yml's apt sources
# name and what CI's runner is; the Makefile passes --platform, so on Apple silicon this
# runs under emulation.
FROM ubuntu:22.04

# The Makefile passes the version pubspec.yaml pins, the one CI's setup-flutter installs.
ARG FLUTTER_VERSION
ENV DEBIAN_FRONTEND=noninteractive

# apt-packages.txt is the list CI's build-linux job installs. The rest is what a runner
# image already has and a bare ubuntu image does not: the tools Flutter's own bootstrap
# and the Makefile call.
COPY apt-packages.txt /tmp/apt-packages.txt
RUN apt-get update \
    && xargs apt-get install -y ca-certificates curl git make unzip xz-utils zip < /tmp/apt-packages.txt \
    && rm -rf /var/lib/apt/lists/*

# safe.directory: the SDK is unpacked as root from an archive owned by another uid, and
# `flutter` asks git for its own version on every run.
RUN test -n "$FLUTTER_VERSION" || { \
        echo "ERROR: the FLUTTER_VERSION build arg is required." >&2; \
        echo "  Run \`make build/frontend/linux/appimage\`, which passes the version pubspec.yaml pins." >&2; \
        exit 1; }; \
    curl -fsSL "https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_${FLUTTER_VERSION}-stable.tar.xz" \
        | tar -xJ -C /opt \
    && git config --system --add safe.directory /opt/flutter
ENV PATH=/opt/flutter/bin:$PATH

# appimage-builder at the version and checksum the Makefile pins, unpacked here rather
# than installed by `make setup/appimage-builder`. It ships as an AppImage, and an
# AppImage does not start under the x86_64 emulation Apple silicon runs this image with:
# the AppImage magic sits in the padding bytes of its ELF header (offset 8), and the
# emulator's loader answers "Exec format error". Zeroing those three bytes, after the
# checksum, leaves an ordinary executable that can unpack itself.
ARG APPIMAGE_BUILDER_VERSION
ARG APPIMAGE_BUILDER_SHA256
RUN test -n "$APPIMAGE_BUILDER_VERSION" && test -n "$APPIMAGE_BUILDER_SHA256" || { \
        echo "ERROR: the APPIMAGE_BUILDER_VERSION and APPIMAGE_BUILDER_SHA256 build args are required." >&2; \
        echo "  Run \`make build/frontend/linux/appimage\`, which passes both." >&2; \
        exit 1; }; \
    curl -fsSL -o /tmp/appimage-builder "https://github.com/AppImageCrafters/appimage-builder/releases/download/v${APPIMAGE_BUILDER_VERSION}/appimage-builder-${APPIMAGE_BUILDER_VERSION}-x86_64.AppImage" \
    && echo "${APPIMAGE_BUILDER_SHA256}  /tmp/appimage-builder" | sha256sum -c - \
    && printf '\0\0\0' | dd of=/tmp/appimage-builder bs=1 seek=8 conv=notrunc status=none \
    && chmod +x /tmp/appimage-builder \
    && cd /opt && /tmp/appimage-builder --appimage-extract >/dev/null \
    && mv squashfs-root appimage-builder && rm /tmp/appimage-builder \
    && printf '#!/bin/sh\nexec /opt/appimage-builder/AppRun "$@"\n' > /usr/local/bin/appimage-builder \
    && chmod +x /usr/local/bin/appimage-builder

WORKDIR /work
