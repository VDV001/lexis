#!/usr/bin/env bash
set -euo pipefail

# Smoke test: backup image bundles postgresql-client + age + aws-cli.
#
# Why: a daily Postgres dump → age-encrypt → S3 upload pipeline cannot run
# without all three tools. We verify by building the image and running each
# tool's --version inside it. This locks the image contract so subsequent
# refactors cannot accidentally drop one of them.
#
# How: `docker compose build backup` builds the image declared under the
# backup profile, then `docker compose run --rm backup <cmd>` exercises it.

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

docker compose --profile backup build backup >/dev/null

run_backup() {
    docker compose --profile backup run --rm --no-deps --entrypoint sh backup -c "$1"
}

failures=0
check() {
    local label="$1"
    local cmd="$2"
    if run_backup "$cmd" >/dev/null 2>&1; then
        printf 'PASS: %s available in backup image\n' "$label"
    else
        printf 'FAIL: %s missing from backup image\n' "$label" >&2
        failures=$((failures + 1))
    fi
}

check "pg_dump"  "command -v pg_dump  && pg_dump  --version"
check "age"      "command -v age      && age      --version"
check "aws"      "command -v aws      && aws      --version"

if [ "$failures" -gt 0 ]; then
    printf '\n%d tool(s) missing — backup pipeline cannot run.\n' "$failures" >&2
    exit 1
fi
