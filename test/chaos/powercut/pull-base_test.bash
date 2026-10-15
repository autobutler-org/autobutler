#! /usr/bin/env bash

# Proves pull-base.bash keeps going when a mirror answers 429. That is what
# failed the power-cut job: public.ecr.aws returned "429 Too Many Requests"
# with "Data limit exceeded", and `docker build` stopped before LazyFS was built.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: test/chaos/powercut/pull-base_test.bash

Checks that pull-base.bash prints the first mirror whose `docker pull` succeeds
and fails when every mirror refuses. Uses a fake docker on PATH, so it needs
no Docker daemon. Run `make test/chaos/powercut/base` to get the same.
USAGE
    exit 0
fi

cd "$(dirname "${BASH_SOURCE[0]}")"
script="./pull-base.bash"

fail() {
    echo "pull-base_test: $1" >&2
    exit 1
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat >"$tmp/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${DOCKER_LOG:?}"
if [[ "${1:-}" != "pull" || $# -ne 2 ]]; then
    echo "fake docker: expected: docker pull <image>; got: $*" >&2
    exit 2
fi
# A real `docker pull` writes progress to stdout. The script must not let
# that leak into the image reference it prints.
printf 'progress %s\n' "$2"
if [[ " ${DOCKER_FAIL:?} " == *" $2 "* ]]; then
    echo "Error response from daemon: 429 Too Many Requests: Data limit exceeded" >&2
    exit 1
fi
exit 0
EOF
chmod +x "$tmp/docker"

# The mirror the power-cut job hit is refused. The next one is printed, and
# nothing past it is pulled. Stderr is the pull's own noise; a passing run
# stays quiet.
: >"$tmp/log"
out="$(DOCKER_LOG="$tmp/log" DOCKER_FAIL="public.ecr.aws/docker/library/ubuntu:24.04" PATH="$tmp:$PATH" "$script" 2>"$tmp/err")"
[[ "$out" == "mirror.gcr.io/library/ubuntu:24.04" ]] || fail "expected the gcr mirror, got: ${out}"
[[ "$(cat "$tmp/err")" == *"could not pull public.ecr.aws/docker/library/ubuntu:24.04"* ]] ||
    fail "a refused mirror was not reported: $(cat "$tmp/err")"
pulled="$(cat "$tmp/log")"
[[ "$pulled" == $'pull public.ecr.aws/docker/library/ubuntu:24.04\npull mirror.gcr.io/library/ubuntu:24.04' ]] ||
    fail "pull order was: ${pulled}"

# A caller-supplied list stops at the first success.
: >"$tmp/log"
out="$(
    DOCKER_LOG="$tmp/log" DOCKER_FAIL="one.example/ubuntu:24.04" PATH="$tmp:$PATH" \
        POWERCUT_BASE_IMAGES="one.example/ubuntu:24.04 two.example/ubuntu:24.04 three.example/ubuntu:24.04" \
        "$script" 2>"$tmp/err"
)"
[[ "$out" == "two.example/ubuntu:24.04" ]] || fail "expected the second override, got: ${out}"
[[ "$(cat "$tmp/log")" != *"three.example"* ]] || fail "pulled past the first success: $(cat "$tmp/log")"

# Every mirror refused: the script fails and names the ones it tried.
: >"$tmp/log"
if out="$(
    DOCKER_LOG="$tmp/log" \
        DOCKER_FAIL="public.ecr.aws/docker/library/ubuntu:24.04 mirror.gcr.io/library/ubuntu:24.04 ubuntu:24.04" \
        PATH="$tmp:$PATH" "$script" 2>"$tmp/err"
)"; then
    fail "all mirrors refusing still exited 0: ${out}"
fi
err="$(cat "$tmp/err")"
[[ "$err" == *"none of the ubuntu mirrors"* ]] || fail "missing failure summary: ${err}"
[[ "$err" == *"public.ecr.aws/docker/library/ubuntu:24.04"* ]] || fail "summary omits the ecr mirror: ${err}"
[[ "$err" == *"mirror.gcr.io/library/ubuntu:24.04"* ]] || fail "summary omits the gcr mirror: ${err}"
[[ "$err" == *"ubuntu:24.04"* ]] || fail "summary omits Docker Hub: ${err}"
