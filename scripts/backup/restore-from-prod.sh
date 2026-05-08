#!/usr/bin/env bash
set -euo pipefail

# Operator-facing wrapper around restore.sh that refuses to run with the
# committed test age keypair. Use this entry point for real recoveries
# from production backups; smoke / integration tests continue to invoke
# restore.sh directly.
#
# Why a separate script (issue #12, P0 spec block 5.2): the original P0
# requirement asked for a distinct prod-recovery entry point so that an
# accidental swap of AGE_IDENTITY_FILE (test → prod or prod → test) is
# rejected fast instead of producing silent garbage. We compare the SHA-256
# of the supplied identity file against the committed test identity. The
# hash is embedded as a constant — recomputable from
# tests/fixtures/age/test_identity.txt with `sha256sum`.
#
# Args (forwarded verbatim to restore.sh after the guard):
#   $1  bucket object name
#   $2  target database URL
#
# Required env (passed through to restore.sh):
#   AGE_IDENTITY_FILE      path to the production age identity file
#   S3_ENDPOINT_URL, S3_BUCKET, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
#
# Exit codes:
#   0  — wrapper passed, restore.sh exec'd (contract follows restore.sh)
#   1  — confirmation gate rejected (from restore.sh)
#   2  — missing/unreadable AGE_IDENTITY_FILE (or missing env in restore.sh)
#   64 — usage error (wrong arg count), BSD sysexits
#   65 — fingerprint matches the committed test identity (guard tripped)

# SHA-256 of tests/fixtures/age/test_identity.txt — keep in sync if the
# committed test keypair is ever rotated (see tests/fixtures/age/README.md).
TEST_IDENTITY_SHA256="aba0917df258f36ad11f6460589d1f844637797c7e13694e576cb1a83c4aa9ff"

if [ "$#" -ne 2 ]; then
    printf 'usage: restore-from-prod.sh <bucket-object> <target-database-url>\n' >&2
    exit 64
fi

if [ -z "${AGE_IDENTITY_FILE:-}" ]; then
    printf 'ERROR: AGE_IDENTITY_FILE is not set.\n' >&2
    exit 2
fi

if [ ! -r "$AGE_IDENTITY_FILE" ]; then
    printf 'ERROR: AGE_IDENTITY_FILE %s is not readable.\n' \
        "$AGE_IDENTITY_FILE" >&2
    exit 2
fi

actual_hash="$(sha256sum "$AGE_IDENTITY_FILE" | awk '{print $1}')"

if [ "$actual_hash" = "$TEST_IDENTITY_SHA256" ]; then
    cat >&2 <<'EOF'
🔴 REFUSED: AGE_IDENTITY_FILE matches the committed test identity.

This script is for production recovery only and refuses to run with the
test keypair (tests/fixtures/age/test_identity.txt). For integration
or smoke tests that must use the test keypair, invoke restore.sh
directly — that path is not gated by this guard.

If you intended to run a real recovery, point AGE_IDENTITY_FILE at the
production age identity (1Password → Mac Keychain → tmpfs) and re-run.
EOF
    exit 65
fi

# Guard cleared. Delegate to the unified restore engine. We resolve
# restore.sh relative to this script so the wrapper works both inside the
# backup container (/backup/...) and during bats runs (/code/scripts/...).
script_dir="$(cd "$(dirname "$0")" && pwd)"
exec "${script_dir}/restore.sh" "$@"
