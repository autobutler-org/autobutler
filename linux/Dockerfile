# Builder for the Linux bundle and AppImage on a host that cannot build them natively: a
# Mac, or a Linux host of the other architecture. Nothing here ships: `make
# build/frontend/linux` and `make build/frontend/linux/appimage` build this image for
# LINUX_ARCH's platform and run the same target inside it. CI never uses it.
#
# Ubuntu 22.04 because that is what linux/AppImageBuilder.yml's apt sources name and what
# CI's runner is. The Makefile passes --platform: on Apple silicon linux/arm64 runs
# natively and linux/amd64 under emulation.
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

# Cloned rather than unpacked from the release tarball, whose Dart SDK is x86_64 only; a
# clone fetches the Dart SDK and engine for whichever architecture it runs on, which
# precache does here so the image carries them. safe.directory: the inner make runs
# `flutter` as root, and `flutter` asks git for its own version on every run.
RUN test -n "$FLUTTER_VERSION" || { \
        echo "ERROR: the FLUTTER_VERSION build arg is required." >&2; \
        echo "  Run \`make build/frontend/linux LINUX_ARCH=<x64|arm64>\`, which passes the version pubspec.yaml pins." >&2; \
        exit 1; }; \
    git clone -q --depth 1 -b "$FLUTTER_VERSION" https://github.com/flutter/flutter /opt/flutter \
    && git config --system --add safe.directory /opt/flutter \
    && /opt/flutter/bin/flutter --disable-analytics >/dev/null \
    && /opt/flutter/bin/flutter precache --linux
ENV PATH=/opt/flutter/bin:$PATH

# appimage-builder at the version the Makefile pins. For amd64, its release AppImage at the
# pinned checksum, unpacked here rather than installed by `make setup/appimage-builder`.
# An AppImage does not start under the x86_64 emulation Apple silicon runs this image with:
# the AppImage magic sits in the padding bytes of its ELF header (offset 8), and the
# emulator's loader answers "Exec format error". Zeroing those three bytes, after the
# checksum, leaves an ordinary executable that can unpack itself. The release has no
# aarch64 AppImage, so arm64 installs the same version from PyPI into a venv, hash-locked
# by appimage-builder-requirements.txt, which says why it pins what it does.
ARG TARGETARCH
COPY appimage-builder-requirements.txt /tmp/appimage-builder-requirements.txt
ARG APPIMAGE_BUILDER_VERSION
ARG APPIMAGE_BUILDER_SHA256
RUN test -n "$APPIMAGE_BUILDER_VERSION" && test -n "$APPIMAGE_BUILDER_SHA256" || { \
        echo "ERROR: the APPIMAGE_BUILDER_VERSION and APPIMAGE_BUILDER_SHA256 build args are required." >&2; \
        echo "  Run \`make build/frontend/linux/appimage LINUX_ARCH=<x64|arm64>\`, which passes both." >&2; \
        exit 1; }; \
    if [ "$TARGETARCH" = arm64 ]; then \
        python3 -m venv /opt/appimage-builder \
        && /opt/appimage-builder/bin/pip install --no-cache-dir --require-hashes \
            -r /tmp/appimage-builder-requirements.txt \
        && ln -s /opt/appimage-builder/bin/appimage-builder /usr/local/bin/appimage-builder; \
        exit; \
    fi; \
    curl -fsSL -o /tmp/appimage-builder "https://github.com/AppImageCrafters/appimage-builder/releases/download/v${APPIMAGE_BUILDER_VERSION}/appimage-builder-${APPIMAGE_BUILDER_VERSION}-x86_64.AppImage" \
    && echo "${APPIMAGE_BUILDER_SHA256}  /tmp/appimage-builder" | sha256sum -c - \
    && printf '\0\0\0' | dd of=/tmp/appimage-builder bs=1 seek=8 conv=notrunc status=none \
    && chmod +x /tmp/appimage-builder \
    && cd /opt && /tmp/appimage-builder --appimage-extract >/dev/null \
    && mv squashfs-root appimage-builder && rm /tmp/appimage-builder \
    && printf '#!/bin/sh\nexec /opt/appimage-builder/AppRun "$@"\n' > /usr/local/bin/appimage-builder \
    && chmod +x /usr/local/bin/appimage-builder

WORKDIR /work
