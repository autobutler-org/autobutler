#! /usr/bin/env bash
# cspell:words ndk ZipFile downloadIntermediates sdkmanager

# Install the Android NDK before build-android runs Gradle.
#
# Flutter's plugin calls sdkmanager once while the app project is configured.
# A truncated archive ("Error on ZipFile unknown archive") is cached, and that
# single call fails the job. scripts/ensure-android-ndk.bash retries the same
# failure from Gradle, but a pull request published from main does not contain
# that script: the retry landed on a lower layer. build-android on #3314 died
# that way, at android/build.gradle.kts, before any of this pull request's
# Dart was compiled.
#
# With no arguments this discovers the SDK and the NDK version Flutter pins,
# which is what the workflow runs. With three arguments it installs that
# version. CI_ANDROID_NDK_RETRY_DELAY, when set, replaces the pause between
# tries (tests set it to 0).

set -euo pipefail

usage() {
    cat <<'USAGE'
Usage: scripts/ci-android-ndk.bash [<sdk-root> <sdkmanager> <ndk-version>]

Installs Flutter's NDK into the Android SDK, dropping a corrupt sdkmanager
archive between tries. With no arguments, finds the SDK and the version
Flutter pins. Exits 0 when the NDK is already installed.
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi
if [[ "$#" -ne 0 && "$#" -ne 3 ]]; then
    usage >&2
    exit 2
fi

install_ndk() {
    local sdk_root="$1" sdkmanager="$2" version="$3"
    local marker="${sdk_root}/ndk/${version}/source.properties"

    if [[ -z "${sdk_root}" || "${sdk_root}" == "/" || ! -d "${sdk_root}" ]]; then
        echo "ci-android-ndk: SDK root is not a directory: ${sdk_root}" >&2
        exit 1
    fi
    # The version is interpolated into a path that this script deletes.
    if [[ ! "${version}" =~ ^[0-9]+(\.[0-9]+)+$ ]]; then
        echo "ci-android-ndk: refusing NDK version '${version}'" >&2
        exit 1
    fi
    if [[ ! -f "${sdkmanager}" ]]; then
        echo "ci-android-ndk: sdkmanager not found at ${sdkmanager}" >&2
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

    local attempt=1
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
        echo "ci-android-ndk: installing NDK ${version} failed (attempt ${attempt})" >&2
        if [[ "${attempt}" -ge 3 ]]; then
            echo "ci-android-ndk: could not install NDK ${version} after 3 attempts" >&2
            exit 1
        fi
        attempt="$((attempt + 1))"
        if [[ -n "${CI_ANDROID_NDK_RETRY_DELAY:-}" ]]; then
            sleep "${CI_ANDROID_NDK_RETRY_DELAY}"
        else
            sleep "$((attempt * 5))"
        fi
    done
}

discover() {
    local flutter_bin flutter_sdk gradle_utils
    sdk_root="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
    if [[ -z "${sdk_root}" || ! -d "${sdk_root}" ]]; then
        echo "ci-android-ndk: Android SDK not found; set ANDROID_SDK_ROOT" >&2
        exit 1
    fi
    sdkmanager="${sdk_root}/cmdline-tools/latest/bin/sdkmanager"
    if [[ -n "${FLUTTER_ROOT:-}" && -d "${FLUTTER_ROOT}" ]]; then
        flutter_sdk="${FLUTTER_ROOT}"
    else
        flutter_bin="$(command -v flutter || true)"
        if [[ -z "${flutter_bin}" ]]; then
            echo "ci-android-ndk: flutter is not on PATH" >&2
            exit 1
        fi
        if readlink -f "${flutter_bin}" >/dev/null 2>&1; then
            flutter_bin="$(readlink -f "${flutter_bin}")"
        fi
        flutter_sdk="$(cd "$(dirname "${flutter_bin}")/.." && pwd)"
    fi
    gradle_utils="${flutter_sdk}/packages/flutter_tools/lib/src/android/gradle_utils.dart"
    if [[ ! -f "${gradle_utils}" ]]; then
        echo "ci-android-ndk: could not read Flutter's NDK version from ${flutter_sdk}" >&2
        exit 1
    fi
    # Flutter pins this in gradle_utils.dart and copies it onto ndkVersion.
    version="$(sed -n "s/.*const ndkVersion = '\\([^']*\\)'.*/\\1/p" "${gradle_utils}" | head -n 1)"
    if [[ -z "${version}" ]]; then
        echo "ci-android-ndk: could not read Flutter's NDK version from ${gradle_utils}" >&2
        exit 1
    fi
}

if [[ "$#" -eq 3 ]]; then
    install_ndk "$@"
fi

discover
install_ndk "${sdk_root}" "${sdkmanager}" "${version}"
