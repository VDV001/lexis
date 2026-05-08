#!/usr/bin/env bash
set -euo pipefail

# Smoke test: backup cron service is declared in docker-compose under profile "backup".
#
# Why: reflective-agent-defaults v1.3 Rule 4 (infrastructure-side enforcement)
# requires automated backups outside blast radius. The service must live in
# the project compose file but stay opt-in (profile gate) so it does not run
# on every `docker compose up` during local dev.
#
# How: use `docker compose config` (real YAML parser) instead of grep — grep
# would match comments or unrelated tokens.

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

services="$(docker compose --profile backup config --services 2>&1)"

if ! printf '%s\n' "$services" | grep -qx "backup"; then
    printf 'FAIL: service "backup" not found under compose profile "backup".\n' >&2
    printf 'Services discovered:\n%s\n' "$services" >&2
    exit 1
fi

printf 'PASS: backup service declared under profile "backup".\n'
