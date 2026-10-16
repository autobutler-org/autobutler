#! /usr/bin/env bash
# cspell:words ndk ZipFile downloadIntermediates

# Install one Android NDK through sdkmanager, retrying the failure that took
# down build-android on #3311. sdkmanager cached a truncated archive
# ("Error on ZipFile unknown archive") and Flutter's Gradle plugin calls it
# only once, so that one bad download fails the build. Each try deletes the
# cached archive and any half-extracted NDK first.
#
# Usage: scripts/ensure-android-ndk.bash <sdk-root> <sdkmanager> <ndk-version>
# ENSURE_ANDROID_NDK_RETRY_DELAY, when set, replaces the pause between tries
# (tests set it to 0).

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || "$#" -ne 3 ]]; then
    cat <<'USAGE'
Usage: scripts/ensure-android-ndk.bash <sdk-root> <sdkmanager> <ndk-version>

Installs the NDK version into the Android SDK, dropping a corrupt
sdkmanager archive between tries. Exits 0 when the NDK is already installed.
USAGE
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        exit 0
    fi
    exit 2
fi

sdk_root="$1"
sdkmanager="$2"
version="$3"
marker="${sdk_root}/ndk/${version}/source.properties"

if [[ -z "${sdk_root}" || "${sdk_root}" == "/" || ! -d "${sdk_root}" ]]; then
    echo "ensure-android-ndk: SDK root is not a directory: ${sdk_root}" >&2
    exit 1
fi
# The version is interpolated into a path that this script deletes.
if [[ ! "${version}" =~ ^[0-9]+(\.[0-9]+)+$ ]]; then
    echo "ensure-android-ndk: refusing NDK version '${version}'" >&2
    exit 1
fi
if [[ ! -f "${sdkmanager}" ]]; then
    echo "ensure-android-ndk: sdkmanager not found at ${sdkmanager}" >&2
    exit 1
fi
if [[ -f "${marker}" ]]; then
    exit 0
fi

clear_partial() {
    rm -rf \
        "${sdk_root}/.downloadIntermediates" \
        "${sdk_root}/downloadIntermediates" \
        "${sdk_root}/.temp" \
        "${sdk_root}/temp" \
        "${sdk_root}/ndk/${version}"
}

attempt=1
while true; do
    clear_partial
    if "${sdkmanager}" --sdk_root="${sdk_root}" --install "ndk;${version}" && [[ -f "${marker}" ]]; then
        # A later sdkmanager call in the same build must not unpack an archive
        # this install left behind.
        rm -rf \
            "${sdk_root}/.downloadIntermediates" \
            "${sdk_root}/downloadIntermediates" \
            "${sdk_root}/.temp" \
            "${sdk_root}/temp"
        exit 0
    fi
    echo "ensure-android-ndk: installing NDK ${version} failed (attempt ${attempt})" >&2
    if [[ "${attempt}" -ge 3 ]]; then
        echo "ensure-android-ndk: could not install NDK ${version} after 3 attempts" >&2
        exit 1
    fi
    attempt="$((attempt + 1))"
    if [[ -n "${ENSURE_ANDROID_NDK_RETRY_DELAY:-}" ]]; then
        sleep "${ENSURE_ANDROID_NDK_RETRY_DELAY}"
    else
        sleep "$((attempt * 5))"
    fi
done
