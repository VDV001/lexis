#!/usr/bin/env bats

# restore_confirm.sh — out-of-band confirmation gate for critical restore.
#
# Refs: reflective-agent-defaults v1.3 Rule 5 (out-of-band confirmation
# for critical DESTRUCTIVE actions). The script reads a name from stdin
# and exits 0 only when the input is byte-for-byte equal to the expected
# name passed as $1. Strict comparison — no trimming, no case-folding,
# no fuzziness — because the whole point is that the operator types the
# exact resource name in a separate channel from the agent.
#
# Exit code contract:
#   0  — input matches expected, restore may proceed
#   1  — input did not match (confirmation rejected)
#   64 — usage error (missing/extra args), per BSD sysexits convention
#
# These specific codes matter: assertions like `-ne 0` would pass on
# exit 127 (script missing), giving a false GREEN. Strict equality keeps
# RED honest until restore_confirm.sh actually exists.

CONFIRM="$BATS_TEST_DIRNAME/../../scripts/backup/restore_confirm.sh"

EXPECTED="lexis-20260508T065056Z.sql.age"

@test "exact match accepted (exit 0)" {
    run bash -c "echo '$EXPECTED' | $CONFIRM '$EXPECTED'"
    [ "$status" -eq 0 ]
}

@test "single-char typo rejected (exit 1)" {
    typo="lexis-20260508T065056Z.sql.aGe"  # 'g' → 'G'
    run bash -c "echo '$typo' | $CONFIRM '$EXPECTED'"
    [ "$status" -eq 1 ]
}

@test "empty input rejected (exit 1)" {
    run bash -c "echo '' | $CONFIRM '$EXPECTED'"
    [ "$status" -eq 1 ]
}

@test "leading whitespace rejected (exit 1, strict comparison)" {
    padded=" $EXPECTED"
    run bash -c "echo '$padded' | $CONFIRM '$EXPECTED'"
    [ "$status" -eq 1 ]
}

@test "missing argument is a usage error (exit 64)" {
    run bash -c "echo '$EXPECTED' | $CONFIRM"
    [ "$status" -eq 64 ]
}
