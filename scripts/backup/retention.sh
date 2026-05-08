#!/usr/bin/env bash
set -euo pipefail

# Pure 7+4+3 retention bucketing algorithm.
#
# Stdin:  one "<filename> <epoch_seconds>" per line.
# Arg 1:  current time as epoch seconds.
# Stdout: filenames to delete, one per line. Order: deletions emitted as
#         they are decided (older same-bucket entries get displaced as
#         newer ones arrive); anything older than the monthly horizon is
#         emitted immediately.
#
# Bucket math is intentionally epoch-based, not calendar-based, so the
# script needs no `date -d` and runs identically on any POSIX shell with
# bash builtins (alpine BusyBox included). The resulting buckets do not
# align to midnight UTC or ISO weeks, but they are stable and deterministic
# — the policy is "one per ~24h slice, one per ~7d slice, one per ~30d
# slice", which matches the operational intent of 7+4+3.
#
# Window thresholds:
#   age < 7d   →  daily   bucket  (epoch / 86400)
#   age < 35d  →  weekly  bucket  (epoch / 604800)
#   age < 96d  →  monthly bucket  (epoch / 2592000)
#   age ≥ 96d  →  delete unconditionally

if [ "$#" -ne 1 ]; then
    printf 'usage: retention.sh <now_epoch>\n' >&2
    exit 64
fi

now="$1"

declare -A keepers

DAY_S=86400
WEEK_S=604800
MONTH_S=2592000
DAILY_HORIZON=$((7 * DAY_S))      # 604800
WEEKLY_HORIZON=$((35 * DAY_S))    # 3024000
MONTHLY_HORIZON=$((96 * DAY_S))   # 8294400

while read -r filename epoch; do
    [ -z "${filename:-}" ] && continue
    [ -z "${epoch:-}" ] && continue

    age=$((now - epoch))

    if [ "$age" -lt 0 ]; then
        # Future timestamp — operator clock skew or fixture artefact.
        # Keep silent rather than guess intent.
        continue
    elif [ "$age" -lt "$DAILY_HORIZON" ]; then
        bucket="d$((epoch / DAY_S))"
    elif [ "$age" -lt "$WEEKLY_HORIZON" ]; then
        bucket="w$((epoch / WEEK_S))"
    elif [ "$age" -lt "$MONTHLY_HORIZON" ]; then
        bucket="m$((epoch / MONTH_S))"
    else
        printf '%s\n' "$filename"
        continue
    fi

    if [ -z "${keepers[$bucket]:-}" ]; then
        keepers[$bucket]="${filename}|${epoch}"
    else
        prev_record="${keepers[$bucket]}"
        prev_epoch="${prev_record##*|}"
        prev_file="${prev_record%|*}"

        if [ "$epoch" -gt "$prev_epoch" ]; then
            # Newer entry wins this bucket — older one is now redundant.
            printf '%s\n' "$prev_file"
            keepers[$bucket]="${filename}|${epoch}"
        else
            # Current entry is older than the bucket holder; mark it for
            # deletion immediately.
            printf '%s\n' "$filename"
        fi
    fi
done
