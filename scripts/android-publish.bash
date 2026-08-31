#! /usr/bin/env bash

# Uploads an Android App Bundle to Google Play via the Play Developer API v3.
#
# The API works in "edits": open one, upload into it, point a track at the uploaded
# versionCode, then commit. Nothing is visible to testers until the commit, and an
# uncommitted edit expires on its own -- but this script abandons it explicitly on any
# failure, so a retry is never blocked by a half-finished one.
#
# Authenticates as a service account with a self-signed JWT, so there is no dependency on
# gcloud or fastlane on the machine running it.

set -euo pipefail

API="https://androidpublisher.googleapis.com/androidpublisher/v3"
UPLOAD_API="https://androidpublisher.googleapis.com/upload/androidpublisher/v3"
TOKEN_URL="https://oauth2.googleapis.com/token"
SCOPE="https://www.googleapis.com/auth/androidpublisher"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEFAULT_KEY="${HOME}/.config/quark/play-service-account.json"

usage() {
    cat <<'USAGE'
Usage: scripts/android-publish.bash [--aab PATH] [--track NAME] [--dry-run]

Uploads an Android App Bundle to a Google Play track.

Options:
  --aab PATH     The .aab to upload. Default: newest in build/android-release
  --track NAME   Play track to release on. Default: internal
                 One of: internal, alpha, beta, production
  --dry-run      Upload and stage the release, then abandon the edit instead of
                 committing. Nothing reaches testers. Proves credentials and the
                 bundle are acceptable.
  -h, --help     Show this help.

Credentials:
  The service account JSON is read from GOOGLE_PLAY_SERVICE_ACCOUNT_JSON, or from
  ~/.config/quark/play-service-account.json

  The service account needs the "Release manager" permission on the app in the Play
  Console, and the Play Developer API must be enabled on its Google Cloud project.

See docs/android-release.md for how to create it.
USAGE
}

die() {
    echo "Error: $1" >&2
    shift
    for line in "$@"; do echo "  ${line}" >&2; done
    exit 1
}

AAB=""
TRACK="internal"
DRY_RUN=0
while [[ $# -gt 0 ]]; do
    case "$1" in
        --aab) AAB="${2:?--aab needs a path}"; shift 2 ;;
        --track) TRACK="${2:?--track needs a name}"; shift 2 ;;
        --dry-run) DRY_RUN=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Error: unknown argument '$1'" >&2; echo >&2; usage >&2; exit 2 ;;
    esac
done

for tool in openssl curl jq unzip; do
    command -v "${tool}" >/dev/null 2>&1 || die "'${tool}' is required but not installed."
done

case "${TRACK}" in
    internal|alpha|beta|production) ;;
    *) die "unknown track '${TRACK}'." "Use one of: internal, alpha, beta, production." ;;
esac

# build/android-release is where the build target copies the finished AAB under its
# published name. Gradle's own output directory is deliberately not used: it accumulates
# bundles across builds, so the newest file there can be a previous version's.
if [[ -z "${AAB}" ]]; then
    AAB="$(ls -t "${REPO_ROOT}"/build/android-release/*.aab 2>/dev/null | head -1 || true)"
    [[ -n "${AAB}" ]] || die \
        "no AAB found in build/android-release." \
        "Run 'make build/frontend/android/aab' first, or pass --aab PATH."
fi
[[ -f "${AAB}" ]] || die "'${AAB}' does not exist."

# Refuse to upload a debug build. `make build/frontend/android/aab` already checks this,
# but publish can be run on its own against whatever is lying in the output directory.
if unzip -l "${AAB}" | grep -q "kernel_blob.bin"; then
    die "'${AAB}' is a DEBUG build (contains flutter_assets/kernel_blob.bin)." \
        "It would install from Play and then refuse to launch." \
        "Rebuild with 'make build/frontend/android/aab'."
fi

# Read the package name from the build file rather than hardcoding it, so the two can
# never disagree about which listing is being published to.
PACKAGE_NAME="$(sed -n 's/^[[:space:]]*applicationId[[:space:]]*=[[:space:]]*"\([^"]*\)".*$/\1/p' \
    "${REPO_ROOT}/android/app/build.gradle.kts" | head -1)"
[[ -n "${PACKAGE_NAME}" ]] || die \
    "could not read applicationId from android/app/build.gradle.kts."

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

SA_JSON="${WORK_DIR}/service-account.json"
if [[ -n "${GOOGLE_PLAY_SERVICE_ACCOUNT_JSON:-}" ]]; then
    printf '%s' "${GOOGLE_PLAY_SERVICE_ACCOUNT_JSON}" > "${SA_JSON}"
else
    [[ -f "${DEFAULT_KEY}" ]] || die \
        "no Google Play service account credentials found." \
        "Set GOOGLE_PLAY_SERVICE_ACCOUNT_JSON, or place the JSON at:" \
        "  ${DEFAULT_KEY}" \
        "See docs/android-release.md for how to create it."
    cp "${DEFAULT_KEY}" "${SA_JSON}"
fi

jq -e . "${SA_JSON}" >/dev/null 2>&1 || die \
    "the service account credentials are not valid JSON." \
    "Pass the whole downloaded key file, not just the private key."

CLIENT_EMAIL="$(jq -r '.client_email // empty' "${SA_JSON}")"
[[ -n "${CLIENT_EMAIL}" ]] || die \
    "the service account JSON has no 'client_email'." \
    "This is a service account key, not an OAuth client secret -- create one under" \
    "IAM & Admin > Service Accounts > Keys."

jq -r '.private_key // empty' "${SA_JSON}" > "${WORK_DIR}/key.pem"
[[ -s "${WORK_DIR}/key.pem" ]] || die "the service account JSON has no 'private_key'."

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

NOW="$(date +%s)"
HEADER="$(jq -cn '{alg:"RS256", typ:"JWT"}' | b64url)"
PAYLOAD="$(jq -cn \
    --arg iss "${CLIENT_EMAIL}" --arg scope "${SCOPE}" --arg aud "${TOKEN_URL}" \
    --argjson iat "${NOW}" \
    '{iss:$iss, scope:$scope, aud:$aud, iat:$iat, exp:($iat + 3600)}' | b64url)"
SIGNING_INPUT="${HEADER}.${PAYLOAD}"

# RS256 is PKCS#1 v1.5 over SHA-256, which is exactly what `openssl dgst -sign` emits for
# an RSA key -- no DER unwrapping, unlike the ECDSA signatures the App Store API wants.
if ! printf '%s' "${SIGNING_INPUT}" \
    | openssl dgst -sha256 -sign "${WORK_DIR}/key.pem" -binary > "${WORK_DIR}/sig.bin" 2>"${WORK_DIR}/sign.err"; then
    die "could not sign the JWT with the service account private key." \
        "$(sed 's/^/    /' "${WORK_DIR}/sign.err")"
fi
JWT="${SIGNING_INPUT}.$(b64url < "${WORK_DIR}/sig.bin")"

TOKEN_RESPONSE="$(curl -sS -X POST "${TOKEN_URL}" \
    --data-urlencode 'grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer' \
    --data-urlencode "assertion=${JWT}")"
ACCESS_TOKEN="$(printf '%s' "${TOKEN_RESPONSE}" | jq -r '.access_token // empty')"
if [[ -z "${ACCESS_TOKEN}" ]]; then
    die "Google rejected the service account credentials." \
        "$(printf '%s' "${TOKEN_RESPONSE}" | jq -r '"\(.error // "?"): \(.error_description // "")"')" \
        "Check the system clock is correct, and that the Play Developer API is enabled" \
        "for the service account's Google Cloud project."
fi

# Every API response goes through here so a failure reports Google's own message rather
# than an empty jq result twenty lines later.
api() {
    local method="$1" url="$2" what="$3"; shift 3
    local response status body
    response="$(curl -sS -w $'\n%{http_code}' -X "${method}" \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" "$@" "${url}")"
    status="$(printf '%s' "${response}" | tail -1)"
    body="$(printf '%s' "${response}" | sed '$d')"
    if [[ "${status}" != 2* ]]; then
        local detail
        detail="$(printf '%s' "${body}" | jq -r '.error.message // empty' 2>/dev/null)"
        [[ -n "${detail}" ]] || detail="${body}"
        echo "Error: ${what} failed (HTTP ${status})." >&2
        echo "  ${detail}" >&2
        case "${status}" in
            401) echo "  The access token was rejected. Check the system clock." >&2 ;;
            403) echo "  The service account lacks permission on '${PACKAGE_NAME}'." >&2
                 echo "  Grant it 'Release manager' in Play Console > Users and permissions." >&2 ;;
            404) echo "  No app '${PACKAGE_NAME}' in this Play account, or the API is not enabled." >&2
                 echo "  The listing must exist and have had one manual upload before the API" >&2
                 echo "  will accept bundles for it." >&2 ;;
        esac
        return 1
    fi
    printf '%s' "${body}"
}

EDIT_ID="$(api POST "${API}/applications/${PACKAGE_NAME}/edits" "opening an edit" \
    -H 'Content-Length: 0' | jq -r '.id')"
[[ -n "${EDIT_ID}" && "${EDIT_ID}" != "null" ]] || die "the Play API returned no edit ID."

# An abandoned edit leaves the app untouched. Without this, a failure mid-upload leaves a
# dangling edit that has to time out before the next attempt can be reasoned about.
abandon_edit() {
    curl -sS -o /dev/null -X DELETE \
        -H "Authorization: Bearer ${ACCESS_TOKEN}" \
        "${API}/applications/${PACKAGE_NAME}/edits/${EDIT_ID}" 2>/dev/null || true
}
trap 'abandon_edit; rm -rf "${WORK_DIR}"' EXIT

echo "Uploading $(basename "${AAB}") ($(du -h "${AAB}" | cut -f1)) to ${PACKAGE_NAME}..." >&2

VERSION_CODE="$(api POST \
    "${UPLOAD_API}/applications/${PACKAGE_NAME}/edits/${EDIT_ID}/bundles?uploadType=media" \
    "uploading the bundle" \
    -H 'Content-Type: application/octet-stream' \
    --data-binary "@${AAB}" | jq -r '.versionCode')"
[[ -n "${VERSION_CODE}" && "${VERSION_CODE}" != "null" ]] || die "the upload returned no versionCode."

# status "completed" releases to the track's testers on commit. There is deliberately no
# "draft" handling: Play forces draft only on an app that has never published a release,
# and Quark's first release is created by hand in the Play Console -- Play requires an AAB
# uploaded there for a new app. By the time this script runs, the app is not a draft app.
api PUT "${API}/applications/${PACKAGE_NAME}/edits/${EDIT_ID}/tracks/${TRACK}" \
    "assigning the ${TRACK} track" \
    -H 'Content-Type: application/json' \
    --data "$(jq -cn --arg track "${TRACK}" --arg code "${VERSION_CODE}" \
        '{track:$track, releases:[{versionCodes:[$code], status:"completed"}]}')" >/dev/null

if [[ "${DRY_RUN}" -eq 1 ]]; then
    echo "Dry run: versionCode ${VERSION_CODE} uploaded and staged on '${TRACK}'." >&2
    echo "Abandoning the edit; nothing was published." >&2
    exit 0
fi

api POST "${API}/applications/${PACKAGE_NAME}/edits/${EDIT_ID}:commit" "committing the edit" \
    -H 'Content-Length: 0' >/dev/null

# Committed: there is no edit left to abandon, and calling DELETE on it would report an
# error on the way out of a successful run.
trap 'rm -rf "${WORK_DIR}"' EXIT

echo "Published versionCode ${VERSION_CODE} to the '${TRACK}' track of ${PACKAGE_NAME}." >&2
echo "Next: it appears for testers after Play finishes processing (usually minutes)." >&2
