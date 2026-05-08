#!/usr/bin/env bash
set -euo pipefail

# Smoke test: backup.sh runs the full pipeline end-to-end against a real
# postgres + a MinIO target. Verifies that:
#   1. backup.sh exits 0 with all six env vars set and no BACKUP_DRY_RUN
#   2. an encrypted .sql.age object lands in the test bucket
#   3. the object decrypts with the paired test identity
#   4. the decrypted plaintext is a valid pg_dump (header line)
#
# The test mounts the test-only age keypair from tests/fixtures/age into
# the backup container so the verification stage has the private identity
# file without it ever touching the production code path.
#
# Refs: reflective-agent-defaults v1.3 Rule 4.

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

COMPOSE=(docker compose
    -f docker-compose.yml
    -f docker-compose.test.yml
    --profile backup
    --profile backup-test)

BUCKET="lexis-backup-test"
PUBKEY="$(tr -d '\n' < tests/fixtures/age/test_pubkey.txt)"

cleanup() {
    "${COMPOSE[@]}" stop minio >/dev/null 2>&1 || true
    "${COMPOSE[@]}" rm -f minio >/dev/null 2>&1 || true
}
trap cleanup EXIT

"${COMPOSE[@]}" build backup >/dev/null
"${COMPOSE[@]}" up -d --wait postgres minio >/dev/null

# Create bucket (ignore "already exists" — smoke must be re-runnable).
"${COMPOSE[@]}" run --rm --no-deps \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    --entrypoint sh backup -c \
    "aws --endpoint-url http://minio:9000 s3 mb s3://${BUCKET} 2>/dev/null || true" >/dev/null

# Run the full backup pipeline (no BACKUP_DRY_RUN).
set +e
backup_output="$("${COMPOSE[@]}" run --rm --no-deps \
    -e DATABASE_URL="postgres://langtutor:langtutor@postgres:5432/langtutor?sslmode=disable" \
    -e S3_ENDPOINT_URL="http://minio:9000" \
    -e S3_BUCKET="${BUCKET}" \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    -e AGE_PUBLIC_KEY="${PUBKEY}" \
    --entrypoint /backup/backup.sh backup 2>&1)"
backup_exit=$?
set -e

if [ "$backup_exit" -ne 0 ]; then
    printf 'FAIL: backup.sh exited %d in full-pipeline mode.\n' "$backup_exit" >&2
    printf 'Output:\n%s\n' "$backup_output" >&2
    exit 1
fi

# Verify a .sql.age object exists in the bucket and decrypts to a pg_dump.
set +e
verify_output="$("${COMPOSE[@]}" run --rm --no-deps \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    -v "${repo_root}/tests/fixtures/age:/keys:ro" \
    --entrypoint sh backup -c '
        set -eu
        latest="$(aws --endpoint-url http://minio:9000 s3 ls s3://'"${BUCKET}"'/ \
                  | awk "/\\.sql\\.age\$/ {print \$NF}" | tail -1)"
        if [ -z "$latest" ]; then
            echo "no .sql.age object in bucket"; exit 1
        fi
        aws --endpoint-url http://minio:9000 s3 cp \
            "s3://'"${BUCKET}"'/$latest" /tmp/dl.age >/dev/null
        age -d -i /keys/test_identity.txt -o /tmp/dl.sql /tmp/dl.age
        head -3 /tmp/dl.sql
    ' 2>&1)"
verify_exit=$?
set -e

if [ "$verify_exit" -ne 0 ]; then
    printf 'FAIL: verification step failed (exit %d).\n' "$verify_exit" >&2
    printf 'Output:\n%s\n' "$verify_output" >&2
    exit 1
fi

if ! printf '%s\n' "$verify_output" | grep -q 'PostgreSQL database dump'; then
    printf 'FAIL: decrypted object does not look like a pg_dump.\n' >&2
    printf 'First lines after decrypt:\n%s\n' "$verify_output" >&2
    exit 1
fi

printf 'PASS: full pipeline (pg_dump → age → s3) round-trips through MinIO.\n'
