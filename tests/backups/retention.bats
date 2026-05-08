#!/usr/bin/env bats

# retention.sh — pure bucketing algorithm for the 7+4+3 retention policy:
#   keep one backup per day for the last 7 days
#   keep one backup per week for the next ~28 days
#   keep one backup per month for the next ~60 days
#   delete anything older than ~96 days
#
# Input (stdin): lines of "<filename> <epoch_seconds>".
# Argument: $1 = now (epoch seconds).
# Output (stdout): one filename per line — those to delete.
#
# Each test feeds a synthetic listing + frozen "now" so the algorithm is
# isolated from real time and date parsing concerns.

RETENTION="$BATS_TEST_DIRNAME/../../scripts/backup/retention.sh"

# Frozen reference timestamp: 2026-05-08T12:00:00Z
NOW=1778414400

@test "single recent backup is kept (no deletions)" {
    run bash -c "echo 'today.sql.age $((NOW - 3600))' | $RETENTION $NOW"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "two backups same day: older one is deleted, newer kept" {
    morning=$((NOW - 7200))
    evening=$((NOW - 1800))
    run bash -c "printf 'morning.sql.age %d\nevening.sql.age %d\n' $morning $evening | $RETENTION $NOW"
    [ "$status" -eq 0 ]
    [ "$output" = "morning.sql.age" ]
}

@test "backup older than 96 days is unconditionally deleted" {
    # 100 days = 8640000 seconds
    run bash -c "echo 'ancient.sql.age $((NOW - 8640000))' | $RETENTION $NOW"
    [ "$status" -eq 0 ]
    [ "$output" = "ancient.sql.age" ]
}

@test "future-dated backup is ignored, not crashed on" {
    run bash -c "echo 'future.sql.age $((NOW + 86400))' | $RETENTION $NOW"
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "weekly window: two backups in the same week, older deleted" {
    # 10 days ago and 9 days ago — both fall in the weekly window (7-34d)
    # and bucket together by epoch/604800
    older=$((NOW - 10 * 86400))
    newer=$((NOW - 9 * 86400))
    run bash -c "printf 'wk-old.sql.age %d\nwk-new.sql.age %d\n' $older $newer | $RETENTION $NOW"
    [ "$status" -eq 0 ]
    [ "$output" = "wk-old.sql.age" ]
}
