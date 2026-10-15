#! /usr/bin/env bash
# cspell:words ndk ZipFile downloadIntermediates sdkmanager Pkg

# Proves scripts/ci-android-ndk.bash retries the failure from build-android
# on #3314. sdkmanager unpacks a truncated NDK archive ("Error on ZipFile
# unknown archive"), caches it, and the next install fails the same way until
# that cache is removed. The Gradle retry in scripts/ensure-android-ndk.bash
# is not in a pull request published from main, so build-android has to run
# this script first. A fake sdkmanager stands in for the real one.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: scripts/ci-android-ndk-test.bash

Checks that the CI NDK install retries a corrupt sdkmanager archive, skips an
NDK that is already installed, stops after three failures, discovers Flutter's
NDK version, and runs before the Android build.
USAGE
    exit 0
fi

cd "$(dirname "${BASH_SOURCE[0]}")/.."

fail() {
    echo "ci-android-ndk-test: $1" >&2
    exit 1
}

SCRIPT="scripts/ci-android-ndk.bash"
[[ -f "${SCRIPT}" ]] || fail "missing ${SCRIPT}"

NDK_VERSION="28.2.13676358"
export CI_ANDROID_NDK_RETRY_DELAY=0

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

write_fake_sdkmanager() {
    local path="$1" mode="$2"
    cat >"${path}" <<EOF
#! /usr/bin/env bash
set -euo pipefail
count_file="\${FAKE_SDKMANAGER_STATE}/count"
count="\$(cat "\${count_file}" 2>/dev/null || echo 0)"
count="\$((count + 1))"
printf '%s\\n' "\${count}" >"\${count_file}"

sdk_root=""
version=""
for arg in "\$@"; do
    case "\${arg}" in
        --sdk_root=*) sdk_root="\${arg#--sdk_root=}" ;;
        ndk\\;*) version="\${arg#ndk;}" ;;
    esac
done
[[ -n "\${sdk_root}" && -n "\${version}" ]] || {
    echo "fake sdkmanager: expected --sdk_root and ndk;<version>" >&2
    exit 1
}

mode="${mode}"
if [[ "\${mode}" == "retry" && "\${count}" -lt 2 ]]; then
    mkdir -p "\${sdk_root}/.downloadIntermediates" "\${sdk_root}/ndk/\${version}"
    printf 'not a zip\\n' >"\${sdk_root}/.downloadIntermediates/bad.zip"
    echo "Warning: An error occurred while preparing SDK package NDK (Side by side) \${version}: Error on ZipFile unknown archive." >&2
    exit 1
fi
if [[ "\${mode}" == "always" ]]; then
    mkdir -p "\${sdk_root}/.downloadIntermediates"
    printf 'not a zip\\n' >"\${sdk_root}/.downloadIntermediates/bad.zip"
    echo "Warning: An error occurred while preparing SDK package NDK (Side by side) \${version}: Error on ZipFile unknown archive." >&2
    exit 1
fi
if [[ -e "\${sdk_root}/.downloadIntermediates" || -d "\${sdk_root}/ndk/\${version}" ]]; then
    echo "fake sdkmanager: corrupt archive was still cached on retry" >&2
    exit 1
fi
mkdir -p "\${sdk_root}/ndk/\${version}"
printf 'Pkg.Revision = %s\\n' "\${version}" >"\${sdk_root}/ndk/\${version}/source.properties"
EOF
    chmod +x "${path}"
}

# A sdkmanager that fails once with the CI error, leaving a corrupt archive,
# then succeeds only when that archive and the partial NDK are gone.
write_fake_sdkmanager "${work}/sdkmanager-retry" retry
export FAKE_SDKMANAGER_STATE="${work}/state-retry"
mkdir -p "${FAKE_SDKMANAGER_STATE}" "${work}/sdk-retry"
if ! bash "${SCRIPT}" "${work}/sdk-retry" "${work}/sdkmanager-retry" "${NDK_VERSION}"; then
    fail "a corrupt archive was not installed on retry"
fi
[[ -f "${work}/sdk-retry/ndk/${NDK_VERSION}/source.properties" ]] ||
    fail "retry did not leave source.properties"
[[ "$(cat "${FAKE_SDKMANAGER_STATE}/count")" == "2" ]] ||
    fail "expected two sdkmanager calls, got $(cat "${FAKE_SDKMANAGER_STATE}/count")"
[[ ! -e "${work}/sdk-retry/.downloadIntermediates" ]] ||
    fail "retry left the cached archive in place"

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

# Three failures and then stop.
write_fake_sdkmanager "${work}/sdkmanager-always" always
export FAKE_SDKMANAGER_STATE="${work}/state-always"
mkdir -p "${FAKE_SDKMANAGER_STATE}" "${work}/sdk-always"
if bash "${SCRIPT}" "${work}/sdk-always" "${work}/sdkmanager-always" "${NDK_VERSION}"; then
    fail "three corrupt installs were treated as success"
fi
[[ "$(cat "${FAKE_SDKMANAGER_STATE}/count")" == "3" ]] ||
    fail "expected three sdkmanager calls, got $(cat "${FAKE_SDKMANAGER_STATE}/count")"

# No arguments: find Flutter's pinned NDK and the SDK, then install.
# This is the path build-android runs.
flutter_sdk="${work}/flutter"
mkdir -p "${flutter_sdk}/bin" "${flutter_sdk}/packages/flutter_tools/lib/src/android"
cat >"${flutter_sdk}/bin/flutter" <<'EOF'
#! /usr/bin/env bash
echo "fake flutter: should not have been invoked" >&2
exit 99
EOF
chmod +x "${flutter_sdk}/bin/flutter"
printf "const ndkVersion = '%s';\n" "${NDK_VERSION}" \
    >"${flutter_sdk}/packages/flutter_tools/lib/src/android/gradle_utils.dart"
write_fake_sdkmanager "${work}/sdkmanager-discover" retry
mkdir -p "${work}/sdk-discover/cmdline-tools/latest/bin"
cp "${work}/sdkmanager-discover" "${work}/sdk-discover/cmdline-tools/latest/bin/sdkmanager"
chmod +x "${work}/sdk-discover/cmdline-tools/latest/bin/sdkmanager"
export FAKE_SDKMANAGER_STATE="${work}/state-discover"
mkdir -p "${FAKE_SDKMANAGER_STATE}"
if ! env \
    PATH="${flutter_sdk}/bin:${PATH}" \
    ANDROID_SDK_ROOT="${work}/sdk-discover" \
    FLUTTER_ROOT="" \
    bash "${SCRIPT}"; then
    fail "discovery did not install a corrupt archive on retry"
fi
[[ -f "${work}/sdk-discover/ndk/${NDK_VERSION}/source.properties" ]] ||
    fail "discovery did not leave source.properties"
[[ "$(cat "${FAKE_SDKMANAGER_STATE}/count")" == "2" ]] ||
    fail "discovery expected two sdkmanager calls, got $(cat "${FAKE_SDKMANAGER_STATE}/count")"

# The install has to run before Gradle, which is what calls sdkmanager once.
install_line="$(grep -n 'bash scripts/ci-android-ndk.bash' .github/workflows/build.yml | head -n 1 | cut -d: -f1)"
build_line="$(grep -n 'make build/frontend/android$' .github/workflows/build.yml | head -n 1 | cut -d: -f1)"
[[ -n "${install_line}" ]] || fail "build-android does not run scripts/ci-android-ndk.bash"
[[ -n "${build_line}" && "${install_line}" -lt "${build_line}" ]] ||
    fail "the NDK install does not run before the Android build"

echo "ci-android-ndk-test: ok"
