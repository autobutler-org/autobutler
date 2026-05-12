#!/usr/bin/env bash
set -euo pipefail

TARGET_DIR="${1:-${PERF_FIXTURE_TARGET_DIR:-}}"
FIXTURE_ROOT_NAME="${PERF_FIXTURE_ROOT_NAME:-perf}"
PHOTO_COUNT="${PERF_FIXTURE_PHOTO_COUNT:-120}"
TEXT_COUNT="${PERF_FIXTURE_TEXT_COUNT:-250}"
RANDOM_FILE_COUNT="${PERF_FIXTURE_RANDOM_FILE_COUNT:-24}"
RANDOM_FILE_SIZE_KB="${PERF_FIXTURE_RANDOM_FILE_SIZE_KB:-16}"

if [[ -z "$TARGET_DIR" ]]; then
  echo "missing target dir: pass PERF_FIXTURE_TARGET_DIR or the path as arg 1" >&2
  exit 1
fi

perf_dir="$TARGET_DIR/$FIXTURE_ROOT_NAME"
nested_dir="$perf_dir/nested"
mkdir -p "$nested_dir"

# Small deterministic JPEG fixture reused across photo-oriented scenarios.
jpeg_base64="/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAkGBxAQEBUQEBIVFRUVFRUVFRUVFRUVFRUWFxUVFRUYHSggGBolGxUVITEhJSkrLi4uFx8zODMtNygtLisBCgoKDg0OGhAQGi0fHyUtLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLf/AABEIAAEAAQMBIgACEQEDEQH/xAAXAAEBAQEAAAAAAAAAAAAAAAAAAQID/8QAFhABAQEAAAAAAAAAAAAAAAAAAAER/9oADAMBAAIQAxAAAAG+AH//xAAVEAEBAAAAAAAAAAAAAAAAAAABEP/aAAgBAQABBQJf/8QAFhEBAQEAAAAAAAAAAAAAAAAAARAR/9oACAEDAQE/AUf/xAAVEQEBAAAAAAAAAAAAAAAAAAABEP/aAAgBAgEBPwFH/8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQAGPwJf/8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQABPyFf/9k="
decode_flag="-d"
if base64 --help 2>/dev/null | grep -q -- '--decode'; then
  decode_flag="--decode"
fi

printf "%s" "$jpeg_base64" | base64 "$decode_flag" > "$perf_dir/sample-1.jpg"
cp "$perf_dir/sample-1.jpg" "$perf_dir/sample-2.jpg"
cp "$perf_dir/sample-1.jpg" "$nested_dir/sample-3.jpg"

for i in $(seq 1 "$PHOTO_COUNT"); do
  printf -v root_photo "%s/photo-%04d.jpg" "$perf_dir" "$i"
  printf -v nested_photo "%s/photo-%04d.jpg" "$nested_dir" "$i"
  cp "$perf_dir/sample-1.jpg" "$root_photo"
  cp "$perf_dir/sample-1.jpg" "$nested_photo"
done

for i in $(seq 1 "$TEXT_COUNT"); do
  printf "fixture-%04d\n" "$i" > "$nested_dir/file-$i.txt"
done

for i in $(seq 1 "$RANDOM_FILE_COUNT"); do
  dd if=/dev/urandom of="$perf_dir/blob-$i.bin" bs=1024 count="$RANDOM_FILE_SIZE_KB" status=none
done