#!/usr/bin/env bash
set -euo pipefail

# Smoke test: backup.sh refuses to run without all required env vars
# and reports each missing one by name.
#
# Why: silent fall-through on missing config is a foot-gun. A backup that
# uploads to "" with empty creds is worse than no backup — it would log
# "success" while the dump never reaches S3. Fail loud at process start.
#
# Required envs (Rule 4 v1.3 — backup pipeline must verify its inputs):
#   DATABASE_URL           — Postgres connection string for pg_dump
#   S3_ENDPOINT_URL        — non-default endpoint (Selectel ru-1 / MinIO test)
#   S3_BUCKET              — target bucket name
#   AWS_ACCESS_KEY_ID      — S3 credential
#   AWS_SECRET_ACCESS_KEY  — S3 credential
#   AGE_PUBLIC_KEY         — recipient public key for encryption
#
# How: run the backup container with NO env vars and assert non-zero exit
# plus stderr that names every missing var.

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

required_vars=(
    DATABASE_URL
    S3_ENDPOINT_URL
    S3_BUCKET
    AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY
    AGE_PUBLIC_KEY
)

set +e
output="$(docker compose --profile backup run --rm --no-deps \
    --entrypoint /backup/backup.sh backup 2>&1)"
exit_code=$?
set -e

if [ "$exit_code" -eq 0 ]; then
    printf 'FAIL: backup.sh exited 0 with no env vars set — should have refused.\n' >&2
    printf 'Output:\n%s\n' "$output" >&2
    exit 1
fi

failures=0
for var in "${required_vars[@]}"; do
    if ! printf '%s\n' "$output" | grep -q "$var"; then
        printf 'FAIL: missing-env error did not mention %s\n' "$var" >&2
        failures=$((failures + 1))
    fi
done

if [ "$failures" -gt 0 ]; then
    printf '\nObserved output:\n%s\n' "$output" >&2
    exit 1
fi

printf 'PASS: backup.sh validates all %d required env vars on startup.\n' \
    "${#required_vars[@]}"
