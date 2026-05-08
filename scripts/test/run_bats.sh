#!/usr/bin/env bash
set -euo pipefail

# Run bats tests inside a pinned Docker image so contributors do not need
# bats installed on the host. Mounts the repo at /code and forwards args
# straight to the bats CLI.
#
# Usage:
#   scripts/test/run_bats.sh tests/backups/
#   scripts/test/run_bats.sh tests/backups/retention.bats

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"

docker run --rm \
    -v "$repo_root:/code" \
    -w /code \
    bats/bats:1.11.1 \
    "$@"
