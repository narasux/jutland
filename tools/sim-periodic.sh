#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -z "${JUTLAND_SIM_HOME:-}" ]]; then
  echo "JUTLAND_SIM_HOME is required" >&2
  exit 1
fi
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"

go run ./cmd/jutland-sim trend --limit "${SIM_TREND_LIMIT:-50}"
go run ./cmd/jutland-sim analyze --limit "${SIM_ANALYZE_LIMIT:-20}"
