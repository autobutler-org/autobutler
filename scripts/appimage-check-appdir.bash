#! /usr/bin/env bash
# cspell:words pixbuf libpixbufloader librsvg

set -euo pipefail

if [[ $# -ne 1 || "$1" == "-h" || "$1" == "--help" ]]; then
    cat <<'USAGE'
Usage: scripts/appimage-check-appdir.bash APPDIR

Checks an AppDir that appimage-builder has finished setting up, so an AppImage
that cannot start on a bare desktop session fails the build instead (#3451).
linux/AppImageBuilder.yml runs it as its after_runtime script.

  1. The SVG gdk-pixbuf loader (librsvg2-common) is bundled. The system icon
     theme is SVG, and GTK aborts when it cannot decode the icon it falls back to.
  2. Every loader loaders.cache names is in the loaders directory beside it.
     appimage-builder writes the cache from the build host's loaders, not the
     AppDir's, so a host loader the recipe does not bundle lands in it.
  3. AppRun.env points GDK_PIXBUF_MODULE_FILE at that cache.
  4. AppRun.env ends XDG_DATA_DIRS with /usr/local/share:/usr/share, the XDG
     default, so GIO finds the system MIME database when the session set none.

Exits 0 when clean, 1 with one line per violation otherwise.
USAGE
    [[ $# -eq 1 ]] && exit 0
    exit 1
fi

APPDIR="$1"
failures=0
fail() {
    echo "AppDir check: $*" >&2
    failures=$((failures + 1))
}

loaders_dir="$(find "$APPDIR/usr/lib" -type d -path '*/gdk-pixbuf-2.0/*/loaders' -print -quit 2>/dev/null || true)"
if [[ -z "$loaders_dir" ]]; then
    fail "no gdk-pixbuf-2.0/*/loaders directory under $APPDIR/usr/lib"
else
    cache="$(dirname "$loaders_dir")/loaders.cache"
    [[ -f "$loaders_dir/libpixbufloader-svg.so" ]] ||
        fail "libpixbufloader-svg.so is not in $loaders_dir (is librsvg2-common in the apt include list?)"
    if [[ ! -f "$cache" ]]; then
        fail "no loaders.cache beside $loaders_dir"
    else
        grep -q '^"libpixbufloader-svg.so"$' "$cache" || fail "$cache has no SVG loader entry"
        while read -r module; do
            [[ -f "$loaders_dir/$module" ]] || fail "$cache names $module, which is not in $loaders_dir"
        done < <(sed -n 's/^"\(libpixbufloader-[^"]*\.so\)"$/\1/p' "$cache")
    fi
fi

env_file="$APPDIR/AppRun.env"
if [[ ! -f "$env_file" ]]; then
    fail "no AppRun.env in $APPDIR"
else
    grep -q '^GDK_PIXBUF_MODULE_FILE=[$]APPDIR/.*/loaders\.cache$' "$env_file" ||
        fail "AppRun.env does not set GDK_PIXBUF_MODULE_FILE to the bundled loaders.cache"
    grep -q '^XDG_DATA_DIRS=.*:/usr/local/share:/usr/share$' "$env_file" ||
        fail "AppRun.env does not end XDG_DATA_DIRS with /usr/local/share:/usr/share"
fi

if [[ $failures -gt 0 ]]; then
    exit 1
fi
echo "AppDir check: SVG loader, loaders.cache and AppRun.env are in place"
