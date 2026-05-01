#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  backup-prod-db.sh <ssh-target>

Environment:
  OUTPUT_DIR=backup
  REMOTE_CONTAINER=poker-db
  REMOTE_DB=poker
  REMOTE_USER=poker

Example:
  OUTPUT_DIR=backup ./scripts/backup-prod-db.sh root@203.0.113.10
EOF
}

if [[ ${1:-} == "" || ${1:-} == "-h" || ${1:-} == "--help" ]]; then
  usage
  exit 0
fi

ssh_target="$1"
output_dir="${OUTPUT_DIR:-backup}"
remote_container="${REMOTE_CONTAINER:-poker-db}"
remote_db="${REMOTE_DB:-poker}"
remote_user="${REMOTE_USER:-poker}"
timestamp="$(date +%F_%H-%M-%S)"
output_file="${output_dir}/poker-${timestamp}.dump"
tmp_file="${output_file}.tmp"

mkdir -p "$output_dir"
trap 'rm -f "$tmp_file"' EXIT

ssh "$ssh_target" \
  "docker exec -i ${remote_container} pg_dump -U ${remote_user} -d ${remote_db} -Fc" \
  > "$tmp_file"

mv "$tmp_file" "$output_file"
trap - EXIT

printf 'backup saved to %s\n' "$output_file"
