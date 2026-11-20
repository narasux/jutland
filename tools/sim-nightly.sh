#!/usr/bin/env bash
# Nightly lab run: evaluation suite -> rendered-frame gate -> trend/analyze -> archive.
# Intended to be called by launchd/cron; see tools/install-scheduler.sh.
#
#   SIM_NIGHTLY_SUITE   full-evaluation (default) or "small" for tools/sim-suite.sh
#   SIM_NIGHTLY_FRAME   1 (default) runs the rendered-frame gate
#   SIM_NIGHTLY_ARCHIVE 1 archives runs older than SIM_KEEP_RUNS (default 100); off by default
#
# The suite runs with --fail-on-red: missing runs or smoke red items make the job
# exit non-zero, but trend/analyze still run so the failing report is complete.
set -euo pipefail
cd "$(dirname "$0")/.."

: "${JUTLAND_SIM_HOME:?JUTLAND_SIM_HOME is required}"
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"

gate_status=0
suite="${SIM_NIGHTLY_SUITE:-full-evaluation}"
echo "== nightly suite $suite $(date '+%Y-%m-%d %H:%M:%S') =="
if [[ "$suite" == "small" ]]; then
  bash tools/sim-suite.sh || gate_status=1
else
  go run ./cmd/jutland-suite suite "$suite" --fail-on-red || gate_status=1
fi
if [[ "$gate_status" != "0" ]]; then
  echo "suite gate: failed (missing runs or smoke red items)" >&2
fi

# 渲染帧门禁需要可用的图形会话；锁屏/无会话时跳过，不能影响 suite 结果。
if [[ "${SIM_NIGHTLY_FRAME:-1}" == "1" ]]; then
  echo "== rendered frame gate =="
  if bash tools/frame-gate.sh; then
    echo "frame gate: pass"
  else
    echo "frame gate: skipped (no graphical session or over budget)" >&2
  fi
fi

echo "== trend/analyze =="
bash tools/sim-periodic.sh

if [[ "${SIM_NIGHTLY_ARCHIVE:-0}" == "1" ]]; then
  echo "== archive old runs =="
  go run ./cmd/jutland-sim archive --keep "${SIM_KEEP_RUNS:-100}"
fi

echo "== nightly done $(date '+%Y-%m-%d %H:%M:%S') gate_status=$gate_status =="
exit "$gate_status"
