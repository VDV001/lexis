#!/usr/bin/env bash
set -euo pipefail

# Smoke test for the weekly backup-restore CI workflow:
#   1. file exists at the expected path
#   2. actionlint accepts it (no syntax/typing/expression errors)
#   3. it carries the structural elements we depend on:
#        - schedule trigger (so the test runs weekly without nudging)
#        - workflow_dispatch (so an operator can re-run on demand)
#        - invocation of scripts/test/smoke_backup_restore.sh
#
# actionlint runs inside a pinned Docker image — the host does not need
# Go or actionlint installed.

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$repo_root"

WORKFLOW=".github/workflows/backup-restore-test.yml"

if [ ! -f "$WORKFLOW" ]; then
    printf 'FAIL: workflow file missing: %s\n' "$WORKFLOW" >&2
    exit 1
fi

if ! docker run --rm \
        -v "$repo_root:/repo:ro" \
        -w /repo \
        rhysd/actionlint:1.7.7 -color "$WORKFLOW"; then
    printf 'FAIL: actionlint reported issues in %s\n' "$WORKFLOW" >&2
    exit 1
fi

failures=0
check() {
    local label="$1"
    local pattern="$2"
    if ! grep -qE "$pattern" "$WORKFLOW"; then
        printf 'FAIL: workflow does not contain %s (regex: %s)\n' \
            "$label" "$pattern" >&2
        failures=$((failures + 1))
    fi
}

check "schedule trigger"      '^[[:space:]]*schedule:'
check "workflow_dispatch"     '^[[:space:]]*workflow_dispatch:'
check "smoke_backup_restore"  'smoke_backup_restore'

if [ "$failures" -gt 0 ]; then
    exit 1
fi

printf 'PASS: workflow exists, actionlint clean, key elements present.\n'
