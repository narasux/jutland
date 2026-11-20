#!/usr/bin/env bash
# Nightly lab run: evaluation suite -> rendered-frame gate -> trend/analyze -> archive.
# Intended to be called by launchd/cron; see tools/install-scheduler.sh.
#
#   SIM_NIGHTLY_SUITE   full-evaluation (default) or "small" for tools/sim-suite.sh
#   SIM_NIGHTLY_EXTRA_SUITES  space-separated non-gating suites (default "f3-coverage")
#   SIM_NIGHTLY_ATTEMPTS      attempts for Ebiten/GUI init crashes (default 3)
#   SIM_NIGHTLY_RETRY_WAIT    seconds between crash retries (default 15)
#   SIM_NIGHTLY_FRAME   1 (default) runs the rendered-frame gate
#   SIM_NIGHTLY_ARCHIVE 1 archives runs older than SIM_KEEP_RUNS (default 100); off by default
#
# The suite runs with --fail-on-red: missing runs or smoke red items make the job
# exit non-zero, but trend/analyze still run so the failing report is complete.
set -euo pipefail
cd "$(dirname "$0")/.."

: "${JUTLAND_SIM_HOME:?JUTLAND_SIM_HOME is required}"
export GOCACHE="${GOCACHE:-/tmp/jutland-gocache}"

# Ebiten/GLFW 初始化在这台机器上会突发失败（exit>=2 的进程级崩溃），几秒后同样
# 的命令又能跑通。只对崩溃重试；门禁/评测失败是语义结果（exit 1），直接返回，
# 避免重复跑仿真产生重复 run。
run_with_retry() {
  local label="$1"
  shift
  local attempts="${SIM_NIGHTLY_ATTEMPTS:-3}"
  local wait="${SIM_NIGHTLY_RETRY_WAIT:-15}"
  local n=1 code=0
  while (( n <= attempts )); do
    code=0
    "$@" || code=$?
    if (( code == 0 )); then
      return 0
    fi
    if (( code < 2 )) || (( n == attempts )); then
      return "$code"
    fi
    echo "$label crashed (exit $code, likely Ebiten/GUI init); retry $n/$attempts in ${wait}s" >&2
    sleep "$wait"
    n=$((n + 1))
  done
  return "$code"
}

# 帧门禁先跑：它需要干净的图形会话；放在两轮 suite 之后容易被 Ebiten
# 进程占满 WindowServer 而初始化失败。
# 渲染帧门禁需要可用的图形会话；锁屏/无会话时跳过，不能影响 suite 结果。
if [[ "${SIM_NIGHTLY_FRAME:-1}" == "1" ]]; then
  echo "== rendered frame gate =="
  if run_with_retry "frame gate" bash tools/frame-gate.sh; then
    echo "frame gate: pass"
  else
    echo "frame gate: skipped (no graphical session or over budget)" >&2
  fi
fi

gate_status=0
suite="${SIM_NIGHTLY_SUITE:-full-evaluation}"
echo "== nightly suite $suite $(date '+%Y-%m-%d %H:%M:%S') =="
if [[ "$suite" == "small" ]]; then
  run_with_retry "suite $suite" bash tools/sim-suite.sh || gate_status=1
else
  run_with_retry "suite $suite" go run ./cmd/jutland-suite suite "$suite" --fail-on-red || gate_status=1
fi
if [[ "$gate_status" != "0" ]]; then
  echo "suite gate: failed (missing runs or smoke red items)" >&2
fi

# 探索性覆盖套件（例如 AI 会僵持的 test 任务）不参与门禁，失败只记录日志。
for extra in ${SIM_NIGHTLY_EXTRA_SUITES-f3-coverage}; do
  [[ -z "$extra" ]] && continue
  echo "== nightly coverage suite $extra =="
  if ! run_with_retry "coverage $extra" go run ./cmd/jutland-suite suite "$extra"; then
    echo "coverage suite $extra: failed (not gating)" >&2
  fi
done

echo "== trend/analyze =="
bash tools/sim-periodic.sh

if [[ "${SIM_NIGHTLY_ARCHIVE:-0}" == "1" ]]; then
  echo "== archive old runs =="
  go run ./cmd/jutland-sim archive --keep "${SIM_KEEP_RUNS:-100}"
fi

echo "== nightly done $(date '+%Y-%m-%d %H:%M:%S') gate_status=$gate_status =="
exit "$gate_status"
