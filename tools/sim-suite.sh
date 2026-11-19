#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -z "${JUTLAND_SIM_HOME:-}" ]]; then
  echo "JUTLAND_SIM_HOME is required" >&2
  exit 1
fi
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"

# Lightweight smoke suite; edit the list for a larger periodic matrix.
go run ./cmd/jutland-sim shot shot-f0 --runs "${SIM_SHOT_RUNS:-100}"
go run ./cmd/jutland-duel duel duel-jiujiang --ticks "${SIM_DUEL_TICKS:-1800}"
if [[ "${SIM_MACRO_PARALLEL:-0}" == "1" ]]; then
  pids=()
  logs=()
  for seed in ${SIM_MACRO_SEEDS:-1 3 5}; do
    log="${TMPDIR:-/tmp}/jutland-suite-$$-${seed}.log"
    logs+=("$log")
    go run ./cmd/jutland-macro macro macro-pearl --seed "$seed" --ticks "${SIM_MACRO_TICKS:-12000}" >"$log" 2>&1 &
    pids+=("$!")
  done
  failed=0
  for idx in "${!pids[@]}"; do
    if ! wait "${pids[$idx]}"; then
      failed=1
      cat "${logs[$idx]}" >&2
    fi
  done
  if [[ "$failed" != "0" ]]; then
    echo "parallel macro suite failed" >&2
    exit 1
  fi
else
  for seed in ${SIM_MACRO_SEEDS:-1 3 5}; do
    go run ./cmd/jutland-macro macro macro-pearl --seed "$seed" --ticks "${SIM_MACRO_TICKS:-12000}"
  done
fi

go run ./cmd/jutland-sim trend --limit "${SIM_TREND_LIMIT:-50}"
go run ./cmd/jutland-sim analyze --limit "${SIM_ANALYZE_LIMIT:-20}"
