#!/usr/bin/env bash
set -euo pipefail

# Smoke test: restore.sh round-trips a backup into a separate target DB.
#
# Flow:
#   1. Up postgres + minio + api (api applies migrations to source DB)
#   2. Run a full backup → produces an encrypted object in the bucket
#   3. Up the isolated postgres-restore-target service
#   4. Pipe the exact bucket object name into restore.sh as confirmation
#   5. restore.sh downloads, decrypts (test identity), and psql-restores
#      into the target DB
#   6. Assert the target DB now has the "users" table from the migrations
#
# Refs: reflective-agent-defaults v1.3 Rules 4 + 5 (out-of-band confirmation
# for critical DESTRUCTIVE — restore here is the prime example).

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

COMPOSE=(docker compose
    -f docker-compose.yml
    -f docker-compose.test.yml
    --profile backup
    --profile backup-test)

BUCKET="lexis-backup-test"
PUBKEY="$(tr -d '\n' < tests/fixtures/age/test_pubkey.txt)"
RESTORE_DB_URL="postgres://langtutor:langtutor@postgres-restore-target:5432/langtutor_restore?sslmode=disable"

cleanup() {
    "${COMPOSE[@]}" stop postgres-restore-target minio >/dev/null 2>&1 || true
    "${COMPOSE[@]}" rm -f postgres-restore-target minio >/dev/null 2>&1 || true
}
trap cleanup EXIT

"${COMPOSE[@]}" build backup >/dev/null
"${COMPOSE[@]}" up -d --wait postgres minio >/dev/null

# api populates the source DB with the project migrations on first start.
"${COMPOSE[@]}" up -d api >/dev/null
# Allow migrations to run before snapshotting. api has no healthcheck for
# "migrations applied"; a few seconds is enough for the embedded migrator.
sleep 6

# Idempotent bucket create.
"${COMPOSE[@]}" run --rm --no-deps \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    --entrypoint sh backup -c \
    "aws --endpoint-url http://minio:9000 s3 mb s3://${BUCKET} 2>/dev/null || true" >/dev/null

# Full backup of the migrated source DB.
"${COMPOSE[@]}" run --rm --no-deps \
    -e DATABASE_URL="postgres://langtutor:langtutor@postgres:5432/langtutor?sslmode=disable" \
    -e S3_ENDPOINT_URL="http://minio:9000" \
    -e S3_BUCKET="${BUCKET}" \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    -e AGE_PUBLIC_KEY="${PUBKEY}" \
    --entrypoint /backup/backup.sh backup >/dev/null

# Find the most recent .sql.age in the bucket.
object="$("${COMPOSE[@]}" run --rm --no-deps \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    --entrypoint sh backup -c \
    "aws --endpoint-url http://minio:9000 s3 ls s3://${BUCKET}/ | awk '/\\.sql\\.age\$/ {print \$NF}' | tail -1" \
    2>/dev/null | tr -d '\r')"

if [ -z "$object" ]; then
    printf 'FAIL: no backup object in bucket — backup stage broken.\n' >&2
    exit 1
fi

# Bring up the target DB after the backup is sealed.
"${COMPOSE[@]}" up -d --wait postgres-restore-target >/dev/null

# Run restore. Pipe the exact object name as the confirmation typed by
# the operator — in a real recovery this comes from an out-of-band channel.
set +e
restore_output="$("${COMPOSE[@]}" run --rm --no-deps \
    -e S3_ENDPOINT_URL="http://minio:9000" \
    -e S3_BUCKET="${BUCKET}" \
    -e AWS_ACCESS_KEY_ID=minioadmin \
    -e AWS_SECRET_ACCESS_KEY=minioadmin \
    -e AGE_IDENTITY_FILE=/keys/test_identity.txt \
    -v "${repo_root}/tests/fixtures/age:/keys:ro" \
    --entrypoint sh backup -c \
    "echo '${object}' | /backup/restore.sh '${object}' '${RESTORE_DB_URL}'" 2>&1)"
restore_exit=$?
set -e

if [ "$restore_exit" -ne 0 ]; then
    printf 'FAIL: restore.sh exited %d.\n' "$restore_exit" >&2
    printf 'Output:\n%s\n' "$restore_output" >&2
    exit 1
fi

# Verify the target DB now contains the migrated schema.
verify_output="$("${COMPOSE[@]}" exec -T postgres-restore-target \
    psql -U langtutor -d langtutor_restore -t -A \
    -c "SELECT to_regclass('public.users') IS NOT NULL" 2>&1)"

if ! printf '%s\n' "$verify_output" | grep -q '^t$'; then
    printf 'FAIL: "users" table not present in restore target after restore.\n' >&2
    printf 'psql output:\n%s\n' "$verify_output" >&2
    exit 1
fi

printf 'PASS: restore.sh round-trips backup into isolated target DB (object=%s).\n' "$object"
