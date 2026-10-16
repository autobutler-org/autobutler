#! /usr/bin/env bash

# Proves ci-pull.bash rewrites a Dockerfile that pins one Ubuntu registry
# when that registry answers 429. That is what failed the power-cut job on a
# pull request whose Dockerfile still says FROM public.ecr.aws/...: docker
# build stopped before LazyFS was built.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: test/chaos/powercut/ci-pull_test.bash

Checks that ci-pull.bash leaves a Dockerfile that takes BASE_IMAGE alone,
and rewrites a pinned FROM to the first mirror whose `docker pull` succeeds.
Uses a fake docker on PATH, so it needs no Docker daemon.

`make test/chaos/powercut` runs ci-pull.bash before `docker build`. A pull
request based on an older layer still pins the registry, and that recipe is
what hits HTTP 429.
USAGE
    exit 0
fi

cd "$(dirname "${BASH_SOURCE[0]}")"
script="./ci-pull.bash"
root="$(cd ../../.. && pwd)"

fail() {
    echo "ci-pull_test: $1" >&2
    exit 1
}

[[ -f "$script" ]] || fail "missing $script"

# The power-cut job runs `make test/chaos/powercut`. The two lines around this
# call are the same on a tree that still pins the registry and on one that
# passes a base image build arg. Dropping the call puts the 429 back on the
# pinned tree.
makefile="$root/Makefile"
grep -q 'test/chaos/powercut/ci-pull.bash' "$makefile" || fail "power-cut recipe does not run ci-pull.bash"
awk '
    /^test\/chaos\/powercut: / { in_recipe = 1; next }
    in_recipe && /^[^[:space:]#]/ { exit }
    in_recipe && /test\/chaos\/powercut\/ci-pull\.bash/ && $0 !~ /_test\.bash/ { seen = 1 }
    in_recipe && /docker build/ {
        if (!seen) exit 1
        found = 1
    }
    END { if (!found) exit 1 }
' "$makefile" || fail "ci-pull.bash does not run before docker build"

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
printf 'progress %s\n' "$2"
# An empty DOCKER_FAIL means every pull succeeds. ${VAR:?} treats empty as unset.
fail_list="${DOCKER_FAIL-}"
if [[ -n "$fail_list" && " $fail_list " == *" $2 "* ]]; then
    echo "Error response from daemon: 429 Too Many Requests: Data limit exceeded" >&2
    exit 1
fi
exit 0
EOF
chmod +x "$tmp/docker"

pinned='FROM public.ecr.aws/docker/library/ubuntu:24.04'
write_pinned() {
    cat >"$tmp/Dockerfile" <<EOF
# fixture
${pinned}

ARG LAZYFS_COMMIT=abc
EOF
}

# A Dockerfile that already takes BASE_IMAGE is the tree make already pulls
# for. The script must not call docker and must not rewrite it.
cat >"$tmp/arg.Dockerfile" <<'EOF'
ARG BASE_IMAGE=public.ecr.aws/docker/library/ubuntu:24.04
FROM ${BASE_IMAGE}
EOF
: >"$tmp/log"
DOCKER_LOG="$tmp/log" DOCKER_FAIL="" PATH="$tmp:$PATH" \
    POWERCUT_DOCKERFILE="$tmp/arg.Dockerfile" "$script" 2>"$tmp/err"
[[ ! -s "$tmp/log" ]] || fail "docker was called for a BASE_IMAGE Dockerfile: $(cat "$tmp/log")"
grep -qx 'ARG BASE_IMAGE=public.ecr.aws/docker/library/ubuntu:24.04' "$tmp/arg.Dockerfile" ||
    fail "BASE_IMAGE Dockerfile was rewritten"

# The mirror the power-cut job hit is refused. FROM becomes the next one,
# and nothing past it is pulled.
write_pinned
: >"$tmp/log"
DOCKER_LOG="$tmp/log" DOCKER_FAIL="public.ecr.aws/docker/library/ubuntu:24.04" PATH="$tmp:$PATH" \
    POWERCUT_DOCKERFILE="$tmp/Dockerfile" "$script" 2>"$tmp/err"
grep -qx 'FROM mirror.gcr.io/library/ubuntu:24.04' "$tmp/Dockerfile" ||
    fail "expected the gcr mirror, Dockerfile is: $(cat "$tmp/Dockerfile")"
[[ "$(cat "$tmp/err")" == *"could not pull public.ecr.aws/docker/library/ubuntu:24.04"* ]] ||
    fail "a refused mirror was not reported: $(cat "$tmp/err")"
pulled="$(cat "$tmp/log")"
[[ "$pulled" == $'pull public.ecr.aws/docker/library/ubuntu:24.04\npull mirror.gcr.io/library/ubuntu:24.04' ]] ||
    fail "pull order was: ${pulled}"

# The pin already answers: leave FROM where it is.
write_pinned
: >"$tmp/log"
DOCKER_LOG="$tmp/log" DOCKER_FAIL="" PATH="$tmp:$PATH" \
    POWERCUT_DOCKERFILE="$tmp/Dockerfile" "$script" 2>"$tmp/err"
grep -qx "$pinned" "$tmp/Dockerfile" || fail "a successful pin was rewritten: $(cat "$tmp/Dockerfile")"
[[ "$(cat "$tmp/log")" == "pull public.ecr.aws/docker/library/ubuntu:24.04" ]] ||
    fail "pulled past the pin: $(cat "$tmp/log")"

# Every mirror refused: the script fails and the pin stays, so the build does
# not proceed against a registry that already said no.
write_pinned
: >"$tmp/log"
if DOCKER_LOG="$tmp/log" \
    DOCKER_FAIL="public.ecr.aws/docker/library/ubuntu:24.04 mirror.gcr.io/library/ubuntu:24.04 ubuntu:24.04" \
    PATH="$tmp:$PATH" POWERCUT_DOCKERFILE="$tmp/Dockerfile" "$script" 2>"$tmp/err"; then
    fail "all mirrors refusing still exited 0"
fi
err="$(cat "$tmp/err")"
[[ "$err" == *"none of the ubuntu mirrors"* ]] || fail "missing failure summary: ${err}"
[[ "$err" == *"mirror.gcr.io/library/ubuntu:24.04"* ]] || fail "summary omits the gcr mirror: ${err}"
grep -qx "$pinned" "$tmp/Dockerfile" || fail "a failed run rewrote FROM: $(cat "$tmp/Dockerfile")"
