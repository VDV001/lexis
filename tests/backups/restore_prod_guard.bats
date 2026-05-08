#!/usr/bin/env bats

# restore-from-prod.sh — operator wrapper around restore.sh with a
# SHA-256 fingerprint guard against the committed test age keypair.
#
# Why a separate script: the original P0 spec called for a distinct
# operator-only entry point, separate from the integration restore used
# by smoke tests. Issue #12 resolved as "split with fingerprint guard"
# — accidental use of the test keypair in a real recovery would either
# produce undecryptable output (key mismatch) or, worse, succeed against
# a backup encrypted to the test pubkey if such a file ever ended up in
# a prod-flavoured bucket. The guard refuses to run when AGE_IDENTITY_FILE
# matches the committed test identity by exact bytes.
#
# Exit code contract:
#   0  — wrapper passed all checks, exec'd restore.sh (only seen in real flow)
#   1  — confirmation rejected downstream
#   2  — missing or unreadable AGE_IDENTITY_FILE / required env (early)
#   64 — usage error (wrong arg count), BSD sysexits
#   65 — fingerprint matches the test identity (guard tripped)

GUARD="$BATS_TEST_DIRNAME/../../scripts/backup/restore-from-prod.sh"
TEST_ID="$BATS_TEST_DIRNAME/../fixtures/age/test_identity.txt"

setup() {
    NON_TEST_ID="$BATS_TEST_TMPDIR/not_the_test_identity.txt"
    printf 'AGE-SECRET-KEY-1NOTAREALKEYJUSTPLACEHOLDERBYTESFORHASHING\n' \
        > "$NON_TEST_ID"
}

@test "exit 65 when AGE_IDENTITY_FILE is the committed test identity" {
    run env \
        AGE_IDENTITY_FILE="$TEST_ID" \
        S3_ENDPOINT_URL=http://x \
        S3_BUCKET=x \
        AWS_ACCESS_KEY_ID=x \
        AWS_SECRET_ACCESS_KEY=x \
        "$GUARD" obj target-db
    [ "$status" -eq 65 ]
    [[ "$output" == *"test"*"identity"* ]] || \
        [[ "$output" == *"test"*"keypair"* ]]
}

@test "exit 64 on missing args" {
    run env AGE_IDENTITY_FILE="$NON_TEST_ID" "$GUARD"
    [ "$status" -eq 64 ]
}

@test "exit 64 on extra args" {
    run env AGE_IDENTITY_FILE="$NON_TEST_ID" \
        "$GUARD" obj target-db extra
    [ "$status" -eq 64 ]
}

@test "exit 2 when AGE_IDENTITY_FILE is unset" {
    run env -i PATH="$PATH" "$GUARD" obj target-db
    [ "$status" -eq 2 ]
}

@test "exit 2 when AGE_IDENTITY_FILE points at a missing file" {
    run env AGE_IDENTITY_FILE="$BATS_TEST_TMPDIR/does-not-exist" \
        "$GUARD" obj target-db
    [ "$status" -eq 2 ]
}

@test "non-test identity passes guard, delegates to restore.sh" {
    # Guard passed → control reaches restore.sh, which validates its own
    # required env. With only AGE_IDENTITY_FILE set the missing-env branch
    # of restore.sh fires (exit 2). Strict equality on 2 keeps RED honest:
    # `-ne 65` would false-pass when the script is missing (exit 127).
    run env -i PATH="$PATH" \
        AGE_IDENTITY_FILE="$NON_TEST_ID" \
        "$GUARD" obj target-db
    [ "$status" -eq 2 ]
    [[ "$output" != *"test keypair"* ]]
    [[ "$output" != *"test identity"* ]]
}
