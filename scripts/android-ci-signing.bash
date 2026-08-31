#! /usr/bin/env bash

# Prepares a CI runner to produce a signed Android App Bundle:
#   1. decodes the upload keystore out of a repository secret
#   2. writes android/key.properties pointing at it
#   3. proves all four secrets actually open the key, before any build starts
#
# Reads its inputs from the environment so nothing sensitive lands in argv or in the
# shell history of a runner log.

set -euo pipefail

usage() {
    cat <<'USAGE'
Usage: scripts/android-ci-signing.bash [--keystore PATH]

Prepares Android release signing on a CI runner.

Options:
  --keystore PATH  Where to write the decoded keystore.
                   Default: android/quark-release.jks
  -h, --help       Show this help.

Required environment variables:
  ANDROID_KEYSTORE_BASE64    base64 of the release keystore (.jks)
  ANDROID_KEYSTORE_PASSWORD  password protecting the keystore
  ANDROID_KEY_ALIAS          alias of the signing key inside it
  ANDROID_KEY_PASSWORD       password protecting that key

To produce the base64 value locally:
  base64 -i quark-release.jks | pbcopy

See docs/android-release.md for how to create the keystore.
USAGE
}

die() {
    echo "Error: $1" >&2
    shift
    for line in "$@"; do echo "  ${line}" >&2; done
    exit 1
}

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KEYSTORE="android/quark-release.jks"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --keystore) KEYSTORE="${2:?--keystore needs a path}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Error: unknown argument '$1'" >&2; echo >&2; usage >&2; exit 2 ;;
    esac
done

command -v keytool >/dev/null 2>&1 || die \
    "'keytool' is required but not installed." \
    "It ships with the JDK. On a runner, add actions/setup-java before this step."

missing=0
for var in ANDROID_KEYSTORE_BASE64 ANDROID_KEYSTORE_PASSWORD ANDROID_KEY_ALIAS ANDROID_KEY_PASSWORD; do
    if [[ -z "${!var:-}" ]]; then
        echo "Error: ${var} is not set." >&2
        missing=1
    fi
done
if [[ "${missing}" -ne 0 ]]; then
    echo >&2
    echo "These come from GitHub Actions secrets. Run with --help for how to generate them," >&2
    echo "or see docs/android-release.md." >&2
    exit 1
fi

# The keystore must outlive this script -- Gradle reads it later in the job -- so it
# cannot live in a mktemp directory removed by an EXIT trap. android/.gitignore already
# covers **/*.jks, and the workflow's cleanup step deletes it.
KEYSTORE_PATH="${KEYSTORE}"
[[ "${KEYSTORE_PATH}" = /* ]] || KEYSTORE_PATH="${REPO_ROOT}/${KEYSTORE_PATH}"
mkdir -p "$(dirname "${KEYSTORE_PATH}")"

if ! printf '%s' "${ANDROID_KEYSTORE_BASE64}" | base64 --decode > "${KEYSTORE_PATH}" 2>/dev/null; then
    die "ANDROID_KEYSTORE_BASE64 is not valid base64." \
        "Set it from the keystore's base64, not the keystore itself:" \
        "  base64 -i quark-release.jks | gh secret set ANDROID_KEYSTORE_BASE64 --repo <owner>/<repo>"
fi
if [[ ! -s "${KEYSTORE_PATH}" ]]; then
    die "ANDROID_KEYSTORE_BASE64 decoded to an empty keystore."
fi
chmod 600 "${KEYSTORE_PATH}"

# Prove each secret separately, so a failure names the one that is wrong. Discovering
# this during the Gradle signing task instead produces "Failed to read key from store",
# which does not say which of the four is at fault.
ERR="$(mktemp)"
trap 'rm -f "${ERR}"' EXIT

# keytool reports its failures on stdout, not stderr, so capture both streams -- a
# `2>` alone yields an empty "keytool said:" and hides the actual cause.
if ! keytool -list -keystore "${KEYSTORE_PATH}" \
    -storepass "${ANDROID_KEYSTORE_PASSWORD}" >"${ERR}" 2>&1; then
    # macOS base64 accepts garbage silently and emits a short file, so a bad secret
    # arrives here rather than at the decode above. Distinguish the two: blaming the
    # password for what is really a truncated or mis-pasted secret sends whoever is
    # debugging this to the wrong Actions setting.
    if grep -qiE 'unrecognized keystore format|invalid keystore format|not a valid keystore' "${ERR}"; then
        die "ANDROID_KEYSTORE_BASE64 did not decode to a keystore." \
            "keytool said:" \
            "$(sed 's/^/    /' "${ERR}")" \
            "Decoded $(wc -c < "${KEYSTORE_PATH}" | tr -d ' ') bytes." \
            "Regenerate the secret from the keystore file itself:" \
            "  base64 -i quark-release.jks | gh secret set ANDROID_KEYSTORE_BASE64 --repo <owner>/<repo>"
    fi
    die "ANDROID_KEYSTORE_PASSWORD does not open the keystore." \
        "keytool said:" \
        "$(sed 's/^/    /' "${ERR}")"
fi

if ! keytool -list -keystore "${KEYSTORE_PATH}" \
    -storepass "${ANDROID_KEYSTORE_PASSWORD}" \
    -alias "${ANDROID_KEY_ALIAS}" >"${ERR}" 2>&1; then
    die "ANDROID_KEY_ALIAS '${ANDROID_KEY_ALIAS}' is not in the keystore." \
        "Aliases it does contain:" \
        "$(keytool -list -keystore "${KEYSTORE_PATH}" -storepass "${ANDROID_KEYSTORE_PASSWORD}" \
            2>/dev/null | sed -n 's/^\([^,]*\),.*PrivateKeyEntry.*$/    \1/p')"
fi

# -certreq needs the private key itself, which is what actually exercises the key
# password. `keytool -list` does not, so it would pass with a wrong ANDROID_KEY_PASSWORD
# and fail later inside Gradle.
if ! keytool -certreq -keystore "${KEYSTORE_PATH}" \
    -storepass "${ANDROID_KEYSTORE_PASSWORD}" \
    -alias "${ANDROID_KEY_ALIAS}" \
    -keypass "${ANDROID_KEY_PASSWORD}" >"${ERR}" 2>&1; then
    die "ANDROID_KEY_PASSWORD does not unlock key '${ANDROID_KEY_ALIAS}'." \
        "keytool said:" \
        "$(sed 's/^/    /' "${ERR}")" \
        "For a keystore created by 'keytool -genkeypair' with no -keypass, this is the" \
        "same value as ANDROID_KEYSTORE_PASSWORD."
fi

# PKCS12 -- keytool's default format since JDK 9 -- holds a single password. keytool
# ignores a differing -keypass with only a warning, so the check above passes. AGP does
# not ignore it: it hands keyPassword to KeyStore.getKey and the signing task dies with
# "Failed to read key ... from store", after the build has already run.
if grep -qi 'Different store and key passwords not supported' "${ERR}" \
    && [[ "${ANDROID_KEY_PASSWORD}" != "${ANDROID_KEYSTORE_PASSWORD}" ]]; then
    die "this is a PKCS12 keystore, which supports only one password, but ANDROID_KEY_PASSWORD differs from ANDROID_KEYSTORE_PASSWORD." \
        "keytool ignores the mismatch; Gradle will not, and fails at the signing task." \
        "Fix: set ANDROID_KEY_PASSWORD to the same value as ANDROID_KEYSTORE_PASSWORD."
fi

# storeFile is resolved by android/app/build.gradle.kts through rootProject.file(), so a
# relative path here is relative to android/, not to the repo root.
STORE_FILE="${KEYSTORE_PATH}"
if [[ "${STORE_FILE}" == "${REPO_ROOT}/android/"* ]]; then
    STORE_FILE="${STORE_FILE#"${REPO_ROOT}/android/"}"
fi

KEY_PROPERTIES="${REPO_ROOT}/android/key.properties"
cat > "${KEY_PROPERTIES}" <<PROPERTIES
# Generated by scripts/android-ci-signing.bash. Do not commit.
storeFile=${STORE_FILE}
storePassword=${ANDROID_KEYSTORE_PASSWORD}
keyAlias=${ANDROID_KEY_ALIAS}
keyPassword=${ANDROID_KEY_PASSWORD}
PROPERTIES
chmod 600 "${KEY_PROPERTIES}"

FINGERPRINT="$(keytool -list -v -keystore "${KEYSTORE_PATH}" \
    -storepass "${ANDROID_KEYSTORE_PASSWORD}" \
    -alias "${ANDROID_KEY_ALIAS}" 2>/dev/null \
    | sed -n 's/^[[:space:]]*SHA256:[[:space:]]*\(.*\)$/\1/p' | head -1)"

echo "Signing ready." >&2
echo "  Keystore:    ${KEYSTORE_PATH#"${REPO_ROOT}/"}" >&2
echo "  Alias:       ${ANDROID_KEY_ALIAS}" >&2
echo "  SHA-256:     ${FINGERPRINT}" >&2
echo "  Properties:  android/key.properties" >&2
echo >&2
echo "  Play must show this same SHA-256 under Release > Setup > App integrity." >&2
echo "  A different one means the app was signed by a key users cannot upgrade from." >&2
