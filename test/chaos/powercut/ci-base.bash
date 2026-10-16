#! /usr/bin/env bash

# Points a power-cut Dockerfile that pins one Ubuntu registry at the first
# mirror that still answers. `docker pull` progress stays on stderr.
#
# The power-cut workflow runs this before `make test/chaos/powercut`. A pull
# request built from main still has
# `FROM public.ecr.aws/docker/library/ubuntu:24.04`. ECR Public's anonymous
# data cap answers that pull with HTTP 429, and the job stops before LazyFS
# is built. The next mirror is a different quota.
#
# A Dockerfile that already declares ARG BASE_IMAGE is left alone. That tree's
# Makefile pulls the mirror and passes it as the build arg. ensure-base.bash
# is the same rewrite for a Makefile that calls it from check/docker.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: test/chaos/powercut/ci-base.bash

When test/chaos/powercut/Dockerfile pins public.ecr.aws, pulls ubuntu:24.04
from the first mirror that answers and rewrites that FROM line. A Dockerfile
that takes BASE_IMAGE is unchanged.

Override the Dockerfile with POWERCUT_DOCKERFILE. Override the mirror list
with POWERCUT_BASE_IMAGES, a whitespace-separated list of image references.

The power-cut job runs this before `make test/chaos/powercut`. A mirror that
answers 429 is skipped; if every mirror refuses, the script exits 1 and names
the ones it tried.
USAGE
    exit 0
fi

if [[ $# -ne 0 ]]; then
    echo "ci-base.bash: unknown argument: $1 (try --help)" >&2
    exit 2
fi

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
dockerfile="${POWERCUT_DOCKERFILE:-$here/Dockerfile}"

if [[ ! -f "$dockerfile" ]]; then
    echo "ci-base.bash: no Dockerfile at ${dockerfile}" >&2
    echo "ci-base.bash: run it from the repo, or set POWERCUT_DOCKERFILE." >&2
    exit 1
fi

if grep -q '^ARG BASE_IMAGE=' "$dockerfile"; then
    exit 0
fi

pinned='public.ecr.aws/docker/library/ubuntu:24.04'
if ! grep -qx "FROM ${pinned}" "$dockerfile"; then
    exit 0
fi

images=(
    "$pinned"
    mirror.gcr.io/library/ubuntu:24.04
    ubuntu:24.04
)
if [[ -n "${POWERCUT_BASE_IMAGES:-}" ]]; then
    read -r -a images <<<"${POWERCUT_BASE_IMAGES}"
fi

chosen=""
for image in "${images[@]}"; do
    if docker pull "$image" >&2; then
        chosen="$image"
        break
    fi
    echo "powercut: could not pull ${image}; trying the next mirror" >&2
done

if [[ -z "$chosen" ]]; then
    echo "powercut: none of the ubuntu mirrors could be pulled: ${images[*]}" >&2
    echo "powercut: the power-cut job needs one of those registries to answer. Re-run once they do, or set POWERCUT_BASE_IMAGES to a mirror you can pull." >&2
    exit 1
fi

if [[ "$chosen" == "$pinned" ]]; then
    exit 0
fi

tmp="${dockerfile}.tmp"
: >"$tmp"
while IFS= read -r line || [[ -n "$line" ]]; do
    if [[ "$line" == "FROM ${pinned}" ]]; then
        line="FROM ${chosen}"
    fi
    printf '%s\n' "$line"
done <"$dockerfile" >"$tmp"
mv "$tmp" "$dockerfile"
echo "powercut: base image ${chosen}" >&2
