#! /usr/bin/env bash

# This script enforces the Go layout conventions from AGENTS.md that a general-purpose
# linter cannot see: the interface file every package must have, version prefixes that
# disagree with their directory, and handlers reaching for low-level packages.

set -euo pipefail

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    cat <<'USAGE'
Usage: scripts/check-go-structure.bash

Checks the Go layout conventions documented in AGENTS.md:

  1. Every package under pkg/ and every router package under internal/server/api/
     has an interface file named after its own directory (albums/albums.go).
  2. No interface file declares a private func or type. The exported surface
     lives there; privates belong in types.go or helpers.go.
  3. No file carries a v<N>_ prefix that disagrees with the api version directory
     it sits in (v1_files.go inside v0/files).
  4. Handler packages under internal/server/api/ do not import the low-level
     packages that belong in pkg/util or internal/db.

Exits 0 when the tree is clean, 1 with one line per violation otherwise.
Run `make check/lint/go` to get this plus golangci-lint.
USAGE
    exit 0
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${REPO_ROOT}"

if ! command -v go >/dev/null 2>&1; then
    echo "go is not on PATH. Install Go (see the version pinned in go.mod) and try again."
    exit 1
fi

API_ROOT="internal/server/api"

# Packages a handler must not reach for directly. Shelling out, poking at the OS, and
# opening a database driver are all somebody else's job -- pkg/util wraps the first two,
# internal/db owns the third. Deliberately short: every entry is at zero use today, so
# the list stays honest and can be tightened as the layers separate further.
FORBIDDEN_HANDLER_IMPORTS=(
    "os/exec"
    "syscall"
    "golang.org/x/sys/unix"
    "database/sql/driver"
    "modernc.org/sqlite"
    "github.com/mattn/go-sqlite3"
)

violations=0

fail() {
    echo "$1"
    violations=$((violations + 1))
}

# Collected up front rather than piped straight into the loops: a `go list` that fails
# on the far side of a pipe would leave the loop with nothing to read, and the script
# would report a clean tree for code that does not even compile.
if ! PACKAGE_DIRS="$(go list -f '{{.Dir}}' ./pkg/... "./${API_ROOT}/...")"; then
    echo "go list failed -- the tree does not build, so its structure cannot be checked."
    exit 1
fi
if ! API_IMPORTS="$(go list -f '{{.ImportPath}} {{join .Imports " "}}' "./${API_ROOT}/...")"; then
    echo "go list failed -- the tree does not build, so its structure cannot be checked."
    exit 1
fi

# 1. Every directory holding Go code needs the interface file named after itself.
#    `go list` is the source of truth for "is this a package" -- a directory of only
#    _test.go files or of only subdirectories (pkg/util) is not one.
#
# 2. And that file is the package's public face, so nothing private may sit in it.
#    grep rather than a parser: a private top-level declaration always opens its line
#    with `func x` or `type x`, an exported one with a capital, and a method reads
#    `func (r *router)`. Consts and vars are deliberately not checked -- a private
#    tuning value next to the exported thing it tunes is not the sprawl this catches,
#    and flagging every one of them would drown the signal.
while read -r dir; do
    pkg="$(basename "${dir}")"
    if [[ ! -f "${dir}/${pkg}.go" ]]; then
        fail "${dir}: missing interface file ${pkg}.go (see AGENTS.md, API package layout)"
        continue
    fi
    while read -r decl; do
        [[ -z "${decl}" ]] && continue
        fail "${dir}/${pkg}.go: private '${decl}' in the interface file; move it to types.go or helpers.go (see AGENTS.md, API package layout)"
    done < <(grep -oE '^(func|type) [a-z][A-Za-z0-9_]*' "${dir}/${pkg}.go" || true)
done < <(echo "${PACKAGE_DIRS}" | sed "s|^${REPO_ROOT}/||" | sort)

# 3. A v<N>_ filename prefix that disagrees with the version directory it lives in.
#    v1_files.go inside v0/files is a rename someone started and did not finish.
while read -r file; do
    prefix="$(basename "${file}" | sed -E 's/^(v[0-9]+)_.*/\1/')"
    version="$(echo "${file}" | sed -E "s|^${API_ROOT}/(v[0-9]+)/.*|\1|")"
    if [[ "${prefix}" != "${version}" ]]; then
        fail "${file}: ${prefix}_ prefix disagrees with its ${version}/ directory"
    fi
done < <(find "${API_ROOT}" -name 'v[0-9]*_*.go' | sort)

# 4. Handlers route and validate; they do not do the work themselves. Enforced as an
#    import deny-list, because "business logic" is not something grep can recognize.
while read -r pkg_imports; do
    pkg="${pkg_imports%% *}"
    for forbidden in "${FORBIDDEN_HANDLER_IMPORTS[@]}"; do
        if [[ " ${pkg_imports#* } " == *" ${forbidden} "* ]]; then
            fail "${pkg}: handler package imports ${forbidden}; move that work into pkg/util or internal/db"
        fi
    done
done < <(echo "${API_IMPORTS}")

if [[ "${violations}" -gt 0 ]]; then
    echo ""
    echo "${violations} Go structure violation(s). The conventions are documented in AGENTS.md."
    exit 1
fi

echo "Go structure OK"
