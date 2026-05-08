#!/usr/bin/env bash
set -euo pipefail

# Smoke test: backup.sh in dry-run mode writes a non-empty pg_dump to a
# mounted volume, against the project's postgres service.
#
# Why dry-run mode: at this point in the implementation, age + S3 stages do
# not yet exist. We test pg_dump in isolation by setting BACKUP_DRY_RUN=1
# which short-circuits the pipeline after the dump and writes the SQL into
# /backup/dumps/<timestamp>.sql inside the container. The volume mount
# captures that file on the host so we can inspect it.
#
# How: bring postgres up healthy, run backup container with
# BACKUP_DRY_RUN=1 + DATABASE_URL pointing at the compose-internal hostname
# "postgres", mount /tmp/lexis-backup-test as /backup/dumps, then assert
# the resulting .sql file is non-empty and begins with the PostgreSQL dump
# header.

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

dump_host_dir="$(mktemp -d -t lexis-backup-smoke-XXXXXX)"
trap 'rm -rf "$dump_host_dir"' EXIT

docker compose --profile backup build backup >/dev/null
docker compose up -d --wait postgres >/dev/null

set +e
output="$(docker compose --profile backup run --rm --no-deps \
    -e BACKUP_DRY_RUN=1 \
    -e DATABASE_URL="postgres://langtutor:langtutor@postgres:5432/langtutor?sslmode=disable" \
    -e S3_ENDPOINT_URL="http://stub" \
    -e S3_BUCKET="stub" \
    -e AWS_ACCESS_KEY_ID="stub" \
    -e AWS_SECRET_ACCESS_KEY="stub" \
    -e AGE_PUBLIC_KEY="age1stubpublickey00000000000000000000000000000000000000000" \
    -v "$dump_host_dir:/backup/dumps" \
    --entrypoint /backup/backup.sh backup 2>&1)"
exit_code=$?
set -e

if [ "$exit_code" -ne 0 ]; then
    printf 'FAIL: backup.sh exited %d in dry-run mode.\n' "$exit_code" >&2
    printf 'Output:\n%s\n' "$output" >&2
    exit 1
fi

dump_file="$(find "$dump_host_dir" -maxdepth 1 -name '*.sql' -type f | head -n1)"

if [ -z "$dump_file" ]; then
    printf 'FAIL: no .sql file produced in %s\n' "$dump_host_dir" >&2
    printf 'Output:\n%s\n' "$output" >&2
    ls -la "$dump_host_dir" >&2
    exit 1
fi

if [ ! -s "$dump_file" ]; then
    printf 'FAIL: dump file %s exists but is empty\n' "$dump_file" >&2
    exit 1
fi

if ! head -3 "$dump_file" | grep -q 'PostgreSQL database dump'; then
    printf 'FAIL: dump file does not look like a pg_dump output\n' >&2
    printf 'First lines:\n' >&2
    head -3 "$dump_file" >&2
    exit 1
fi

printf 'PASS: backup.sh produced a non-empty pg_dump (%s, %d bytes).\n' \
    "$(basename "$dump_file")" "$(wc -c < "$dump_file")"
