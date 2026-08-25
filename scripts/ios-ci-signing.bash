#! /usr/bin/env bash

# Prepares a macOS CI runner to produce a signed App Store build:
#   1. imports the Apple Distribution certificate into a throwaway keychain
#   2. installs the App Store provisioning profile
#   3. writes an ExportOptions plist wired for manual signing
#
# Reads its inputs from the environment so nothing sensitive lands in argv or
# in the shell history of a runner log.

set -euo pipefail

usage() {
    cat <<'USAGE'
Usage: scripts/ios-ci-signing.bash [--out PATH]

Prepares iOS App Store code signing on a CI runner. macOS only.

Options:
  --out PATH   Where to write the generated ExportOptions plist.
               Default: ios/ExportOptions.ci.plist
  -h, --help   Show this help.

Required environment variables:
  IOS_DIST_CERT_P12_BASE64        base64 of the Apple Distribution .p12
  IOS_DIST_CERT_PASSWORD          password protecting that .p12
  IOS_PROVISIONING_PROFILE_BASE64 base64 of the App Store .mobileprovision

To produce the base64 values locally:
  base64 -i Certificates.p12 | pbcopy
  base64 -i Quark_App_Store.mobileprovision | pbcopy

See docs/ios-release.md for how to obtain the certificate and profile.
USAGE
}

OUT="ios/ExportOptions.ci.plist"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --out) OUT="${2:?--out needs a path}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Error: unknown argument '$1'"; echo; usage; exit 2 ;;
    esac
done

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "Error: iOS code signing requires macOS (found $(uname -s))."
    echo "  Fix: run this job on a macos-* runner."
    exit 1
fi

missing=0
for var in IOS_DIST_CERT_P12_BASE64 IOS_DIST_CERT_PASSWORD IOS_PROVISIONING_PROFILE_BASE64; do
    if [[ -z "${!var:-}" ]]; then
        echo "Error: ${var} is not set."
        missing=1
    fi
done
if [[ "${missing}" -ne 0 ]]; then
    echo
    echo "These come from GitHub Actions secrets. Run with --help for how to generate them,"
    echo "or see docs/ios-release.md."
    exit 1
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

CERT_PATH="${WORK_DIR}/dist.p12"
PROFILE_PATH="${WORK_DIR}/profile.mobileprovision"
echo "${IOS_DIST_CERT_P12_BASE64}" | base64 --decode > "${CERT_PATH}"
echo "${IOS_PROVISIONING_PROFILE_BASE64}" | base64 --decode > "${PROFILE_PATH}"

# A dedicated keychain keeps the runner's login keychain untouched and lets the
# job leave no credentials behind. The password is ephemeral and never leaves
# this process.
KEYCHAIN="${WORK_DIR}/quark-signing.keychain-db"
KEYCHAIN_PASSWORD="$(uuidgen)"

security create-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN}"
security set-keychain-settings -lut 21600 "${KEYCHAIN}"
security unlock-keychain -p "${KEYCHAIN_PASSWORD}" "${KEYCHAIN}"
security import "${CERT_PATH}" \
    -k "${KEYCHAIN}" \
    -P "${IOS_DIST_CERT_PASSWORD}" \
    -T /usr/bin/codesign \
    -T /usr/bin/security \
    -f pkcs12 \
    -A >/dev/null
# Without this, codesign blocks on a GUI prompt that no runner can answer.
security set-key-partition-list \
    -S apple-tool:,apple:,codesign: \
    -s -k "${KEYCHAIN_PASSWORD}" "${KEYCHAIN}" >/dev/null
security list-keychains -d user -s "${KEYCHAIN}" $(security list-keychains -d user | tr -d '"')

if ! security find-identity -v -p codesigning "${KEYCHAIN}" | grep -q "Apple Distribution"; then
    echo "Error: no 'Apple Distribution' identity found after importing the .p12."
    echo "  The certificate is probably an Apple Development cert, or the export"
    echo "  omitted its private key. Re-export from Keychain Access selecting BOTH"
    echo "  the certificate and its private key."
    exit 1
fi

# Read the profile's own metadata rather than hardcoding a name that Apple may
# regenerate. The profile is a CMS-signed plist.
PROFILE_PLIST="${WORK_DIR}/profile.plist"
security cms -D -i "${PROFILE_PATH}" > "${PROFILE_PLIST}" 2>/dev/null

PROFILE_NAME="$(/usr/libexec/PlistBuddy -c 'Print :Name' "${PROFILE_PLIST}")"
PROFILE_UUID="$(/usr/libexec/PlistBuddy -c 'Print :UUID' "${PROFILE_PLIST}")"
TEAM_ID="$(/usr/libexec/PlistBuddy -c 'Print :TeamIdentifier:0' "${PROFILE_PLIST}")"
APP_ID="$(/usr/libexec/PlistBuddy -c 'Print :Entitlements:application-identifier' "${PROFILE_PLIST}")"
BUNDLE_ID="${APP_ID#"${TEAM_ID}."}"

if /usr/libexec/PlistBuddy -c 'Print :ProvisionedDevices' "${PROFILE_PLIST}" >/dev/null 2>&1; then
    echo "Error: '${PROFILE_NAME}' is a development or ad-hoc profile (it lists devices)."
    echo "  App Store distribution needs a profile of type 'App Store Connect'."
    echo "  Create one at https://developer.apple.com/account/resources/profiles"
    exit 1
fi

# Xcode 16 moved the profile directory. xcodebuild still reads the legacy path, but
# which one wins depends on the toolchain version on the runner, so install to both.
for PROFILE_DIR in \
    "${HOME}/Library/MobileDevice/Provisioning Profiles" \
    "${HOME}/Library/Developer/Xcode/UserData/Provisioning Profiles"; do
    mkdir -p "${PROFILE_DIR}"
    cp "${PROFILE_PATH}" "${PROFILE_DIR}/${PROFILE_UUID}.mobileprovision"
done

cat > "${OUT}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<!-- Generated by scripts/ios-ci-signing.bash. Do not commit. -->
	<key>method</key>
	<string>app-store-connect</string>
	<key>teamID</key>
	<string>${TEAM_ID}</string>
	<key>signingStyle</key>
	<string>manual</string>
	<key>signingCertificate</key>
	<string>Apple Distribution</string>
	<key>provisioningProfiles</key>
	<dict>
		<key>${BUNDLE_ID}</key>
		<string>${PROFILE_NAME}</string>
	</dict>
	<key>destination</key>
	<string>export</string>
	<key>uploadSymbols</key>
	<true/>
	<key>stripSwiftSymbols</key>
	<true/>
	<key>manageAppVersionAndBuildNumber</key>
	<false/>
</dict>
</plist>
PLIST

plutil -lint "${OUT}" >/dev/null

echo "Signing ready."
echo "  Team:    ${TEAM_ID}"
echo "  Bundle:  ${BUNDLE_ID}"
echo "  Profile: ${PROFILE_NAME} (${PROFILE_UUID})"
echo "  Export:  ${OUT}"
