#!/usr/bin/env bash
set -euo pipefail

# Out-of-band confirmation gate for critical restore.
#
# Reads a name from stdin and exits 0 iff the input is byte-for-byte
# equal to the expected name passed as $1. No trimming, no case-folding,
# no fuzziness. The whole point is that the operator types the exact
# resource name in a channel separate from the agent — any flexibility
# here defeats Rule 5 of reflective-agent-defaults v1.3.
#
# Exit codes:
#   0  — input matched, caller may proceed
#   1  — input did not match
#   64 — usage error (missing/extra args), BSD sysexits

if [ "$#" -ne 1 ]; then
    printf 'usage: restore_confirm.sh <expected-name>\n' >&2
    exit 64
fi

expected="$1"

# `IFS= read -r` — empty IFS prevents bash from stripping leading
# whitespace, which it would otherwise do as part of word-splitting.
# Strict-comparison contract requires we see exactly what was typed.
typed=""
IFS= read -r typed || true

if [ "$typed" = "$expected" ]; then
    exit 0
fi

printf 'Confirmation rejected: typed name does not match expected.\n' >&2
exit 1
