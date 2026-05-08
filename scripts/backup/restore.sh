#!/usr/bin/env bash
set -euo pipefail

# Restore an encrypted backup from S3 into a target Postgres database.
#
# Critical DESTRUCTIVE action: this overwrites whatever is in the target
# DB. Per reflective-agent-defaults v1.3 Rule 5, confirmation is
# out-of-band: the operator must type the EXACT bucket object name on
# stdin, gated by /backup/restore_confirm.sh. The agent that drives this
# script cannot autocomplete that input on a different channel.
#
# Args:
#   $1  bucket object name (e.g. lexis-20260508T120000Z.sql.age)
#   $2  target database URL (libpq connstring)
#
# Env:
#   S3_ENDPOINT_URL        S3-compatible endpoint of the bucket
#   S3_BUCKET              source bucket
#   AWS_ACCESS_KEY_ID      S3 read credential
#   AWS_SECRET_ACCESS_KEY  S3 read credential
#   AGE_IDENTITY_FILE      path inside the container to the age identity
#                          (private key) used to decrypt the artefact
#
# Exit codes:
#   0   restore complete
#   1   confirmation rejected, no action taken
#   2   missing required env vars
#   64  usage error

if [ "$#" -ne 2 ]; then
    printf 'usage: restore.sh <bucket-object> <target-database-url>\n' >&2
    exit 64
fi

object="$1"
target_db="$2"

required_vars=(
    S3_ENDPOINT_URL
    S3_BUCKET
    AWS_ACCESS_KEY_ID
    AWS_SECRET_ACCESS_KEY
    AGE_IDENTITY_FILE
)

missing=()
for var in "${required_vars[@]}"; do
    if [ -z "${!var:-}" ]; then
        missing+=("$var")
    fi
done

if [ "${#missing[@]}" -gt 0 ]; then
    {
        printf 'ERROR: restore cannot start — missing required env vars:\n'
        for v in "${missing[@]}"; do
            printf '  - %s\n' "$v"
        done
    } >&2
    exit 2
fi

if [ ! -r "$AGE_IDENTITY_FILE" ]; then
    printf 'ERROR: AGE_IDENTITY_FILE %s is not readable\n' \
        "$AGE_IDENTITY_FILE" >&2
    exit 2
fi

# Critical-action block — re-injection of the rule into immediate context
# (Rule 11.2). Operator reads this BEFORE typing the confirmation.
cat >&2 <<EOF
🔴 CRITICAL DESTRUCTIVE ACTION

You are about to RESTORE bucket object:
  ${object}
INTO database:
  ${target_db}

This will overwrite existing tables in the target database.

To proceed, type the EXACT bucket object name (it should match the
'You are about to RESTORE' line above) and press Enter:
EOF

if ! /backup/restore_confirm.sh "$object"; then
    printf 'Restore cancelled by confirmation gate.\n' >&2
    exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

tmp_age="${tmp_dir}/restore.age"
tmp_sql="${tmp_dir}/restore.sql"

aws --endpoint-url "$S3_ENDPOINT_URL" s3 cp \
    "s3://${S3_BUCKET}/${object}" "$tmp_age"

age -d -i "$AGE_IDENTITY_FILE" -o "$tmp_sql" "$tmp_age"

psql "$target_db" -v ON_ERROR_STOP=1 < "$tmp_sql"

printf 'Restore complete: %s → %s\n' "$object" "$target_db" >&2
