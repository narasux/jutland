#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -z "${JUTLAND_SIM_HOME:-}" ]]; then
  echo "JUTLAND_SIM_HOME is required" >&2
  exit 1
fi
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"
exec go run ./cmd/jutland-sim perf "$@"
