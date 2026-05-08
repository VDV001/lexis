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

dump_dir="/backup/dumps"
mkdir -p "$dump_dir"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
dump_file="${dump_dir}/lexis-${timestamp}.sql"

pg_dump --format=plain --no-owner --no-privileges "$DATABASE_URL" > "$dump_file"

if [ "${BACKUP_DRY_RUN:-0}" = "1" ]; then
    printf 'dry-run: dump written to %s (%d bytes)\n' \
        "$dump_file" "$(wc -c < "$dump_file")"
    exit 0
fi

encrypted_file="${dump_file}.age"
age -r "$AGE_PUBLIC_KEY" -o "$encrypted_file" "$dump_file"
# Plaintext dump must not linger on disk — only the encrypted artefact
# leaves the container.
rm -f "$dump_file"

aws --endpoint-url "$S3_ENDPOINT_URL" s3 cp \
    "$encrypted_file" "s3://${S3_BUCKET}/$(basename "$encrypted_file")"

rm -f "$encrypted_file"

# Retention cleanup. Pulled into a function so the integration shape stays
# obvious from the main flow: list → parse → delegate to retention.sh →
# delete. retention.sh itself is the algorithm and stays pure (no aws,
# no date parsing).
apply_retention() {
    local listing files_with_epoch now deletions
    listing="$(aws --endpoint-url "$S3_ENDPOINT_URL" s3 ls "s3://${S3_BUCKET}/" 2>/dev/null || true)"
    files_with_epoch=""
    while read -r _date _time _size object; do
        [ -z "${object:-}" ] && continue
        local ts formatted epoch
        ts="$(printf '%s' "$object" | sed -nE 's/.*lexis-([0-9]{8}T[0-9]{6}Z)\.sql\.age$/\1/p')"
        [ -z "$ts" ] && continue
        formatted="${ts:0:4}-${ts:4:2}-${ts:6:2}T${ts:9:2}:${ts:11:2}:${ts:13:2}Z"
        epoch="$(date -u -d "$formatted" +%s)"
        files_with_epoch+="${object} ${epoch}"$'\n'
    done <<< "$listing"

    now="$(date -u +%s)"
    deletions="$(printf '%s' "$files_with_epoch" | /backup/retention.sh "$now")"

    [ -z "$deletions" ] && return 0

    while read -r victim; do
        [ -z "$victim" ] && continue
        aws --endpoint-url "$S3_ENDPOINT_URL" s3 rm "s3://${S3_BUCKET}/${victim}"
    done <<< "$deletions"
}

apply_retention

printf 'backup uploaded to s3://%s/%s and retention applied\n' \
    "$S3_BUCKET" "$(basename "$encrypted_file")"
