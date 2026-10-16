#! /usr/bin/env bash
# cspell:words ndk ZipFile downloadIntermediates Pkg

# Proves scripts/ensure-android-ndk.bash retries the failure from build-android
# on #3311: sdkmanager unpacks a truncated NDK archive ("Error on ZipFile
# unknown archive"), caches it, and the next install fails the same way until
# that cache is removed. A fake sdkmanager stands in for the real one.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: scripts/ensure-android-ndk-test.bash

Checks that the NDK install retries a corrupt sdkmanager archive, skips an
NDK that is already installed, and stops after three failures.
USAGE
    exit 0
fi

cd "$(dirname "${BASH_SOURCE[0]}")/.."

fail() {
    echo "ensure-android-ndk-test: $1" >&2
    exit 1
}

SCRIPT="scripts/ensure-android-ndk.bash"
[[ -f "${SCRIPT}" ]] || fail "missing ${SCRIPT}"

NDK_VERSION="28.2.13676358"
export ENSURE_ANDROID_NDK_RETRY_DELAY=0

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

# A sdkmanager that fails once with the CI error, leaving a corrupt archive,
# then succeeds only when that archive and the partial NDK are gone.
cat >"${work}/sdkmanager-retry" <<'EOF'
#! /usr/bin/env bash
set -euo pipefail
count_file="${FAKE_SDKMANAGER_STATE}/count"
count="$(cat "${count_file}" 2>/dev/null || echo 0)"
count="$((count + 1))"
printf '%s\n' "${count}" >"${count_file}"

sdk_root=""
version=""
for arg in "$@"; do
    case "${arg}" in
        --sdk_root=*) sdk_root="${arg#--sdk_root=}" ;;
        ndk\;*) version="${arg#ndk;}" ;;
    esac
done
[[ -n "${sdk_root}" && -n "${version}" ]] || {
    echo "fake sdkmanager: expected --sdk_root and ndk;<version>" >&2
    exit 1
}

if [[ "${count}" -lt 2 ]]; then
    mkdir -p "${sdk_root}/.downloadIntermediates" "${sdk_root}/ndk/${version}"
    printf 'not a zip\n' >"${sdk_root}/.downloadIntermediates/bad.zip"
    echo "Warning: An error occurred while preparing SDK package NDK (Side by side) ${version}: Error on ZipFile unknown archive." >&2
    exit 1
fi

if [[ -e "${sdk_root}/.downloadIntermediates" || -d "${sdk_root}/ndk/${version}" ]]; then
    echo "fake sdkmanager: corrupt archive was still cached on retry" >&2
    exit 1
fi
mkdir -p "${sdk_root}/ndk/${version}"
printf 'Pkg.Revision = %s\n' "${version}" >"${sdk_root}/ndk/${version}/source.properties"
EOF
chmod +x "${work}/sdkmanager-retry"

export FAKE_SDKMANAGER_STATE="${work}/state-retry"
mkdir -p "${FAKE_SDKMANAGER_STATE}" "${work}/sdk-retry"
if ! bash "${SCRIPT}" "${work}/sdk-retry" "${work}/sdkmanager-retry" "${NDK_VERSION}"; then
    fail "a corrupt archive was not installed on retry"
fi
[[ -f "${work}/sdk-retry/ndk/${NDK_VERSION}/source.properties" ]] ||
    fail "retry did not leave source.properties"
[[ "$(cat "${FAKE_SDKMANAGER_STATE}/count")" == "2" ]] ||
    fail "expected two sdkmanager calls, got $(cat "${FAKE_SDKMANAGER_STATE}/count")"

# An installed NDK is left alone.
cat >"${work}/sdkmanager-unused" <<'EOF'
#! /usr/bin/env bash
echo "fake sdkmanager: should not have been called" >&2
exit 99
EOF
chmod +x "${work}/sdkmanager-unused"
mkdir -p "${work}/sdk-present/ndk/${NDK_VERSION}"
printf 'Pkg.Revision = %s\n' "${NDK_VERSION}" >"${work}/sdk-present/ndk/${NDK_VERSION}/source.properties"
bash "${SCRIPT}" "${work}/sdk-present" "${work}/sdkmanager-unused" "${NDK_VERSION}" ||
    fail "an installed NDK was not left alone"

# Three failures and then stop. The cache stays poisoned on purpose: a retry
# that does not delete it keeps failing, and so does one that gives up.
cat >"${work}/sdkmanager-always" <<'EOF'
#! /usr/bin/env bash
set -euo pipefail
count_file="${FAKE_SDKMANAGER_STATE}/count"
count="$(cat "${count_file}" 2>/dev/null || echo 0)"
count="$((count + 1))"
printf '%s\n' "${count}" >"${count_file}"
sdk_root=""
version=""
for arg in "$@"; do
    case "${arg}" in
        --sdk_root=*) sdk_root="${arg#--sdk_root=}" ;;
        ndk\;*) version="${arg#ndk;}" ;;
    esac
done
mkdir -p "${sdk_root}/.downloadIntermediates"
printf 'not a zip\n' >"${sdk_root}/.downloadIntermediates/bad.zip"
echo "Warning: An error occurred while preparing SDK package NDK (Side by side) ${version}: Error on ZipFile unknown archive." >&2
exit 1
EOF
chmod +x "${work}/sdkmanager-always"
export FAKE_SDKMANAGER_STATE="${work}/state-always"
mkdir -p "${FAKE_SDKMANAGER_STATE}" "${work}/sdk-always"
if bash "${SCRIPT}" "${work}/sdk-always" "${work}/sdkmanager-always" "${NDK_VERSION}"; then
    fail "three corrupt installs were treated as success"
fi
[[ "$(cat "${FAKE_SDKMANAGER_STATE}/count")" == "3" ]] ||
    fail "expected three sdkmanager calls, got $(cat "${FAKE_SDKMANAGER_STATE}/count")"

echo "ensure-android-ndk-test: ok"
