#! /usr/bin/env bash

# Prints the next App Store Connect build number for the current marketing version.
#
# Build numbers must be unique and strictly increasing within a marketing version, and
# App Store Connect never frees one once consumed. Asking it what it already has is the
# only way to be certain; a counter or a clock can collide or run backwards.
#
# Only the number goes to stdout, so callers can capture it directly. Everything else
# goes to stderr.

set -euo pipefail

API="https://api.appstoreconnect.apple.com/v1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
    cat <<'USAGE'
Usage: scripts/ios-next-build-number.bash [--version X.Y.Z] [--bundle-id ID]

Prints the next unused App Store Connect build number for a marketing version.

Options:
  --version X.Y.Z    Marketing version to scope to. Default: the most recent git tag
  --bundle-id ID     App bundle ID. Default: read from ios/Runner.xcodeproj
  -h, --help         Show this help.

Required environment variables:
  APP_STORE_CONNECT_KEY_ID       API key ID
  APP_STORE_CONNECT_ISSUER_ID    API issuer ID

The private key is read from APP_STORE_CONNECT_PRIVATE_KEY, or from
~/.appstoreconnect/private_keys/AuthKey_<KEY_ID>.p8

See docs/ios-release.md for how to create the key.
USAGE
}

die() {
    echo "Error: $1" >&2
    shift
    for line in "$@"; do echo "  ${line}" >&2; done
    exit 1
}

VERSION=""
BUNDLE_ID=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version) VERSION="${2:?--version needs a value}"; shift 2 ;;
        --bundle-id) BUNDLE_ID="${2:?--bundle-id needs a value}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Error: unknown argument '$1'" >&2; echo >&2; usage >&2; exit 2 ;;
    esac
done

for tool in openssl curl jq xxd awk; do
    command -v "${tool}" >/dev/null 2>&1 || die "'${tool}' is required but not installed."
done

: "${APP_STORE_CONNECT_KEY_ID:=}"
: "${APP_STORE_CONNECT_ISSUER_ID:=}"
if [[ -z "${APP_STORE_CONNECT_KEY_ID}" || -z "${APP_STORE_CONNECT_ISSUER_ID}" ]]; then
    die "APP_STORE_CONNECT_KEY_ID and APP_STORE_CONNECT_ISSUER_ID must be set." \
        "See docs/ios-release.md for how to create the API key."
fi

if [[ -z "${VERSION}" ]]; then
    # Same source as the Makefile: the most recent tag, not pubspec.yaml, which carries
    # no version field precisely so the two cannot disagree.
    VERSION="$(git -C "${REPO_ROOT}" describe --tags --abbrev=0 2>/dev/null | sed -E 's/^v//')"
    [[ -n "${VERSION}" ]] || die \
        "no git tag found to derive the app version from." \
        "Fetch tags with 'git fetch --tags', or pass --version X.Y.Z."
fi

if [[ -z "${BUNDLE_ID}" ]]; then
    # The test target's identifier is the app's plus a suffix, so exclude it.
    BUNDLE_ID="$(grep -o 'PRODUCT_BUNDLE_IDENTIFIER = [^;]*;' \
        "${REPO_ROOT}/ios/Runner.xcodeproj/project.pbxproj" \
        | sed -E 's/PRODUCT_BUNDLE_IDENTIFIER = "?([^";]*)"?;/\1/' \
        | grep -v '\.RunnerTests$' | head -1)"
    [[ -n "${BUNDLE_ID}" ]] || die "could not read PRODUCT_BUNDLE_IDENTIFIER from the Xcode project"
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

KEY_PATH="${WORK_DIR}/key.p8"
if [[ -n "${APP_STORE_CONNECT_PRIVATE_KEY:-}" ]]; then
    printf '%s' "${APP_STORE_CONNECT_PRIVATE_KEY}" > "${KEY_PATH}"
else
    DEFAULT_KEY="${HOME}/.appstoreconnect/private_keys/AuthKey_${APP_STORE_CONNECT_KEY_ID}.p8"
    [[ -f "${DEFAULT_KEY}" ]] || die \
        "no App Store Connect private key found." \
        "Set APP_STORE_CONNECT_PRIVATE_KEY, or place the .p8 at:" \
        "  ${DEFAULT_KEY}"
    cp "${DEFAULT_KEY}" "${KEY_PATH}"
fi

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

NOW="$(date +%s)"
# Apple rejects tokens with a lifetime over 20 minutes.
HEADER="$(jq -cn --arg kid "${APP_STORE_CONNECT_KEY_ID}" \
    '{alg:"ES256", kid:$kid, typ:"JWT"}' | b64url)"
PAYLOAD="$(jq -cn --arg iss "${APP_STORE_CONNECT_ISSUER_ID}" --argjson iat "${NOW}" \
    '{iss:$iss, iat:$iat, exp:($iat + 900), aud:"appstoreconnect-v1"}' | b64url)"
SIGNING_INPUT="${HEADER}.${PAYLOAD}"

if ! printf '%s' "${SIGNING_INPUT}" \
    | openssl dgst -sha256 -sign "${KEY_PATH}" -binary > "${WORK_DIR}/sig.der" 2>"${WORK_DIR}/sign.err"; then
    die "could not sign the JWT with that private key." \
        "$(cat "${WORK_DIR}/sign.err")" \
        "Is the key the full .p8, including the BEGIN/END lines?"
fi

# JWS wants the raw r||s pair, but openssl emits ASN.1 DER: SEQUENCE { INTEGER r,
# INTEGER s }. Pull both integers out and left-pad each to 32 bytes.
INTS="$(openssl asn1parse -inform DER -in "${WORK_DIR}/sig.der" \
    | awk -F: '/INTEGER/ {gsub(/[ \t]/, "", $NF); print $NF}')"
R_HEX="$(printf '%s' "${INTS}" | sed -n 1p)"
S_HEX="$(printf '%s' "${INTS}" | sed -n 2p)"
[[ -n "${R_HEX}" && -n "${S_HEX}" ]] || die "could not parse the ECDSA signature."

pad32() {
    # Drop any DER sign-padding byte, then left-pad to exactly 64 hex characters.
    local hex="${1: -64}"
    printf '%064s' "${hex}" | tr ' ' '0'
}
SIGNATURE="$(printf '%s%s' "$(pad32 "${R_HEX}")" "$(pad32 "${S_HEX}")" | xxd -r -p | b64url)"
JWT="${SIGNING_INPUT}.${SIGNATURE}"

api_get() {
    local url="$1" body status
    body="$(curl -sS -w $'\n%{http_code}' -H "Authorization: Bearer ${JWT}" "${url}")"
    status="$(printf '%s' "${body}" | tail -1)"
    body="$(printf '%s' "${body}" | sed '$d')"
    if [[ "${status}" != "200" ]]; then
        local detail
        detail="$(printf '%s' "${body}" \
            | jq -r '.errors[]? | "\(.title): \(.detail)"' 2>/dev/null || printf '%s' "${body}")"
        if [[ "${status}" == "401" ]]; then
            die "App Store Connect rejected the credentials (401)." "${detail}" \
                "Check APP_STORE_CONNECT_KEY_ID and APP_STORE_CONNECT_ISSUER_ID match the .p8."
        fi
        die "App Store Connect returned ${status}." "${detail}"
    fi
    printf '%s' "${body}"
}

APP_ID="$(api_get "${API}/apps?filter%5BbundleId%5D=${BUNDLE_ID}" | jq -r '.data[0].id // empty')"
[[ -n "${APP_ID}" ]] || die \
    "no app in App Store Connect with bundle ID '${BUNDLE_ID}'." \
    "Register the bundle ID and create the app record first."

# Scope to this marketing version: build numbers only need to be unique within one.
HIGHEST="$(api_get \
    "${API}/builds?filter%5Bapp%5D=${APP_ID}&filter%5BpreReleaseVersion.version%5D=${VERSION}&limit=200" \
    | jq -r '[.data[]?.attributes.version | select(. != null) | tonumber?] | max // 0')"

NEXT=$((HIGHEST + 1))
if [[ "${HIGHEST}" == "0" ]]; then
    echo "No builds yet for ${BUNDLE_ID} ${VERSION}; starting at ${NEXT}." >&2
else
    echo "Highest existing build for ${BUNDLE_ID} ${VERSION} is ${HIGHEST}; next is ${NEXT}." >&2
fi
echo "${NEXT}"
