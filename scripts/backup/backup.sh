#!/usr/bin/env bash
set -euo pipefail

# Daily Postgres backup pipeline:
#   pg_dump → age encrypt → aws s3 cp → retention cleanup
#
# Stage 1 (this commit): env-var validation only. The dump/encrypt/upload
# stages land in subsequent RED→GREEN pairs.
#
# Refs: reflective-agent-defaults v1.3 Rule 4 (infrastructure-side enforcement).

required_vars=(
    DATABASE_URL
    S3_ENDPOINT_URL
    S3_BUCKET
    AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY
    AGE_PUBLIC_KEY
)

missing=()
for var in "${required_vars[@]}"; do
    if [ -z "${!var:-}" ]; then
        missing+=("$var")
    fi
done

if [ "${#missing[@]}" -gt 0 ]; then
    {
        printf 'ERROR: backup pipeline cannot start — missing required env vars:\n'
        for v in "${missing[@]}"; do
            printf '  - %s\n' "$v"
        done
        printf '\nSet all of the above before invoking backup.sh.\n'
    } >&2
    exit 2
fi

printf 'env validation OK (backup logic not yet implemented)\n'
