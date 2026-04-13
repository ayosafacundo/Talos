#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export TALOS_TEST_SOCKET="${TALOS_TEST_SOCKET:-unix://${TMPDIR:-/tmp}/talos_test_hub_$$.sock}"
echo "TALOS_TEST_SOCKET=$TALOS_TEST_SOCKET"
go test ./internal/hub/ -tags=integration -count=1 "$@"
