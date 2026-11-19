#!/usr/bin/env bash
# Rendered-frame p99 gate. Requires a real window/graphics session.
set -euo pipefail

: "${JUTLAND_SIM_HOME:?JUTLAND_SIM_HOME is required}"

frames="${SIM_FRAME_FRAMES:-300}"
budget="${SIM_FRAME_BUDGET_MS:-16.7}"
out="$JUTLAND_SIM_HOME/reports/frame/latest.json"
mkdir -p "$(dirname "$out")"

exec go run ./cmd/jutland-frame \
  --frames "$frames" \
  --budget-ms "$budget" \
  --out "$out"
