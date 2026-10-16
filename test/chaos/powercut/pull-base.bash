#! /usr/bin/env bash

# Pulls the power-cut image's Ubuntu base from the first mirror that answers
# and prints that image reference on stdout. `docker pull` progress stays on
# stderr, so `docker build --build-arg BASE_IMAGE=$(this script)` can use the
# printed line as-is.
#
# GitHub-hosted runners share source IPs. Docker Hub's anonymous pull limit
# and ECR Public's anonymous data cap both answer this pull with HTTP 429, and
# either one alone has failed the power-cut job. The next mirror is a different
# quota, so a 429 from one is not a failed build.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: test/chaos/powercut/pull-base.bash

Pulls ubuntu:24.04 from the first mirror that answers and prints the reference
to pass as `docker build --build-arg BASE_IMAGE=`. Override the list with
POWERCUT_BASE_IMAGES, a whitespace-separated list of image references.

Run `make test/chaos/powercut` rather than calling this directly. A mirror that
answers 429 is skipped; if every mirror refuses, the script exits 1 and names
the ones it tried.
USAGE
    exit 0
fi

if [[ $# -ne 0 ]]; then
    echo "pull-base.bash: unknown argument: $1 (try --help)" >&2
    exit 2
fi

images=(
    public.ecr.aws/docker/library/ubuntu:24.04
    mirror.gcr.io/library/ubuntu:24.04
    ubuntu:24.04
)
if [[ -n "${POWERCUT_BASE_IMAGES:-}" ]]; then
    read -r -a images <<<"${POWERCUT_BASE_IMAGES}"
fi

for image in "${images[@]}"; do
    if docker pull "$image" >&2; then
        printf '%s\n' "$image"
        exit 0
    fi
    echo "powercut: could not pull ${image}; trying the next mirror" >&2
done

echo "powercut: none of the ubuntu mirrors could be pulled: ${images[*]}" >&2
echo "powercut: the power-cut job needs one of those registries to answer. Re-run once they do, or set POWERCUT_BASE_IMAGES to a mirror you can pull." >&2
exit 1
