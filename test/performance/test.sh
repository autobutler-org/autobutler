#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${AUTOBUTLER_BASE_URL:-http://127.0.0.1:8080}"
AUTH_USER="${AUTOBUTLER_USERNAME:-perf}"
AUTH_PASS="${AUTOBUTLER_PASSWORD:-perf-password}"
ACCESS_TOKEN="${AUTOBUTLER_ACCESS_TOKEN:-}"
THREADS="${TEST_THREADS:-4}"
CONCURRENCY="${TEST_CONCURRENCY:-20}"
DURATION="${TEST_DURATION:-20s}"
UPLOAD_CONCURRENCY="${TEST_UPLOAD_CONCURRENCY:-10}"
UPLOAD_COUNT="${TEST_UPLOAD_COUNT:-20}"

WORK_DIR="${WORK_DIR:-$PWD/test-results/performance}"
DATA_DIR="${DATA_DIR:-$HOME/autobutler/data}"
SCENARIO_DIR="$PWD/test/performance/wrk"

mkdir -p "$WORK_DIR" "$DATA_DIR/cirrus/perf/nested" "$WORK_DIR/upload-fixtures"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd wrk

wait_for_server() {
  local retries=60
  local status_url="$BASE_URL/api/v1/auth/status"
  for _ in $(seq 1 "$retries"); do
    if curl -sS -f "$status_url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "server did not become ready: $status_url" >&2
  return 1
}

extract_session_token() {
  local cookie_file="$1"
  # curl cookie-jar stores HttpOnly cookies with a "#HttpOnly_" prefix in
  # column 1. Those are real cookie rows, not comments, so include them.
  awk '((($0 !~ /^#/) || ($1 ~ /^#HttpOnly_/)) && $6 == "session") { print $7 }' "$cookie_file" | tail -n 1
}

auth_setup_if_needed() {
  local setup_resp
  local setup_status
  local cookie_file="$WORK_DIR/auth_cookie_setup.txt"
  setup_resp="$(curl -sS "$BASE_URL/api/v1/auth/status")"

  if echo "$setup_resp" | grep -q '"setup":true'; then
    return 0
  fi

  setup_status="$(curl -sS -o /dev/null -w "%{http_code}" \
    -c "$cookie_file" \
    -X POST "$BASE_URL/api/v1/auth/setup" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$AUTH_USER\",\"password\":\"$AUTH_PASS\"}")"

  if [[ "$setup_status" != "200" ]]; then
    echo "failed to initialize auth via /api/v1/auth/setup (status=$setup_status)." >&2
    return 1
  fi

  if [[ -z "$ACCESS_TOKEN" ]]; then
    ACCESS_TOKEN="$(extract_session_token "$cookie_file")"
    if [[ -z "$ACCESS_TOKEN" ]]; then
      echo "setup succeeded but no session token cookie was returned." >&2
      return 1
    fi
  fi
}

auth_login_and_get_token() {
  # Respect explicit token override from env for CI or local debugging.
  if [[ -n "$ACCESS_TOKEN" ]]; then
    return 0
  fi

  local login_status
  local cookie_file="$WORK_DIR/auth_cookie_login.txt"
  login_status="$(curl -sS -o /dev/null -w "%{http_code}" \
    -c "$cookie_file" \
    -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$AUTH_USER\",\"password\":\"$AUTH_PASS\"}")"

  if [[ "$login_status" != "200" ]]; then
    echo "autobutler auth mismatch: login failed for configured credentials (status=$login_status)." >&2
    echo "set AUTOBUTLER_ACCESS_TOKEN or AUTOBUTLER_USERNAME/AUTOBUTLER_PASSWORD correctly for this instance." >&2
    return 1
  fi

  ACCESS_TOKEN="$(extract_session_token "$cookie_file")"
  if [[ -z "$ACCESS_TOKEN" ]]; then
    echo "login succeeded but no session token cookie was returned." >&2
    return 1
  fi
}

seed_files() {
  local cirrus_dir="$DATA_DIR/cirrus"
  mkdir -p "$cirrus_dir/perf/nested"

  # 1x1 white JPEG (small deterministic fixture)
  local jpeg_base64="/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAkGBxAQEBUQEBIVFRUVFRUVFRUVFRUVFRUWFxUVFRUYHSggGBolGxUVITEhJSkrLi4uFx8zODMtNygtLisBCgoKDg0OGhAQGi0fHyUtLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLf/AABEIAAEAAQMBIgACEQEDEQH/xAAXAAEBAQEAAAAAAAAAAAAAAAAAAQID/8QAFhABAQEAAAAAAAAAAAAAAAAAAAER/9oADAMBAAIQAxAAAAG+AH//xAAVEAEBAAAAAAAAAAAAAAAAAAABEP/aAAgBAQABBQJf/8QAFhEBAQEAAAAAAAAAAAAAAAAAARAR/9oACAEDAQE/AUf/xAAVEQEBAAAAAAAAAAAAAAAAAAABEP/aAAgBAgEBPwFH/8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQAGPwJf/8QAFBABAAAAAAAAAAAAAAAAAAAAEP/aAAgBAQABPyFf/9k="
  if base64 --help 2>/dev/null | grep -q -- '--decode'; then
    printf "%s" "$jpeg_base64" | base64 --decode > "$cirrus_dir/perf/sample-1.jpg"
    printf "%s" "$jpeg_base64" | base64 --decode > "$cirrus_dir/perf/sample-2.jpg"
    printf "%s" "$jpeg_base64" | base64 --decode > "$cirrus_dir/perf/nested/sample-3.jpg"
  else
    printf "%s" "$jpeg_base64" | base64 -d > "$cirrus_dir/perf/sample-1.jpg"
    printf "%s" "$jpeg_base64" | base64 -d > "$cirrus_dir/perf/sample-2.jpg"
    printf "%s" "$jpeg_base64" | base64 -d > "$cirrus_dir/perf/nested/sample-3.jpg"
  fi

  for i in $(seq 1 250); do
    printf "file-%04d\n" "$i" > "$cirrus_dir/perf/nested/file-$i.txt"
  done

  for i in $(seq 1 "$UPLOAD_COUNT"); do
    dd if=/dev/zero of="$WORK_DIR/upload-fixtures/upload-$i.bin" bs=1024 count=64 status=none
  done
}

seed_albums() {
  curl -sS -X POST "$BASE_URL/api/v1/albums" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name":"perf-root"}' >/dev/null || true

  for i in $(seq 1 25); do
    curl -sS -X POST "$BASE_URL/api/v1/albums" \
      -H "Authorization: Bearer $ACCESS_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"name\":\"perf-album-$i\"}" >/dev/null || true
  done
}

run_wrk() {
  local name="$1"
  local script="$2"

  echo "Running $name"
  wrk \
    -t"$THREADS" \
    -c"$CONCURRENCY" \
    -d"$DURATION" \
    --latency \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -s "$script" \
    "$BASE_URL" | tee "$WORK_DIR/$name.txt"
}

run_upload_stress() {
  echo "Running upload stress"
  ls "$WORK_DIR/upload-fixtures"/*.bin | \
    xargs -I{} -P "$UPLOAD_CONCURRENCY" \
      curl -sS -o /dev/null -w "%{http_code}\n" \
        -X POST "$BASE_URL/api/v1/cirrus/upload" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -F "file=@{}" > "$WORK_DIR/upload_status_codes.txt"

  local failures
  failures="$(grep -vc '^200$' "$WORK_DIR/upload_status_codes.txt" || true)"
  if [[ "$failures" != "0" ]]; then
    echo "upload stress had non-200 responses: $failures" >&2
    return 1
  fi
}

main() {
  wait_for_server
  auth_setup_if_needed
  auth_login_and_get_token
  seed_files
  seed_albums

  run_wrk "cirrus_list" "$SCENARIO_DIR/cirrus_list.lua"
  run_wrk "photos_list" "$SCENARIO_DIR/photos_list.lua"
  run_wrk "thumbnails" "$SCENARIO_DIR/thumbnails.lua"
  run_wrk "albums_list" "$SCENARIO_DIR/albums_list.lua"
  run_wrk "photos_metadata" "$SCENARIO_DIR/photos_metadata.lua"
  run_upload_stress
}

main "$@"
