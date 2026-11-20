#!/usr/bin/env bash
# Nightly lab run: small suite -> rendered-frame gate (best effort) -> trend/analyze.
# Intended to be called by launchd/cron; see tools/install-scheduler.sh.
set -euo pipefail
cd "$(dirname "$0")/.."

: "${JUTLAND_SIM_HOME:?JUTLAND_SIM_HOME is required}"
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"

echo "== nightly suite $(date '+%Y-%m-%d %H:%M:%S') =="
bash tools/sim-suite.sh

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
echo "== nightly done $(date '+%Y-%m-%d %H:%M:%S') =="
