#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# Starts Kanso with the dev login on fresh temporary data for e2e tests.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
export KANSO_ADDR=127.0.0.1:18091
export KANSO_BASE_URL=http://127.0.0.1:18091
export KANSO_DATA_DIR="$tmp/data"
export KANSO_STATE_DIR="$tmp/state"
export KANSO_DEV_LOGIN=true
cd "$root" && go run ./cmd/kanso
