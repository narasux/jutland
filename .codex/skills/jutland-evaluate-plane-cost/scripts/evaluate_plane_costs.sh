#!/usr/bin/env bash
# evaluate_plane_costs.sh — 提取飞机战力数据并计算 fundsCost / timeCost。
#
# 用法：
#   bash .codex/skills/jutland-evaluate-plane-cost/scripts/evaluate_plane_costs.sh
#
# 输出 TSV 格式：name	type	nation	combatPower	tonnage	fundsCost	timeCost

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

# 花费公式参数（与 SKILL.md 保持同步）
SCALE_FACTOR="0.10"
FUNDS_MIN=3
FUNDS_MAX=30
TIME_MIN=3
TIME_MAX=10

# 临时文件路径
TEST_SRC="$SCRIPT_DIR/plane_cost_data_test.go"
TEST_DST="$REPO_ROOT/pkg/mission/object/initialize/plane_cost_data_test.go"

# 清理函数
cleanup() {
    rm -f "$TEST_DST"
}
trap cleanup EXIT

# 步骤 1：复制测试文件
echo "[evaluate_plane_costs] copying test template..." >&2
cp "$TEST_SRC" "$TEST_DST"

# 步骤 2：运行 Go 测试，提取战力数据
echo "[evaluate_plane_costs] running go test..." >&2
cd "$REPO_ROOT"

GO_TEST_OUTPUT="$(go test -run TestPlaneCostData -v ./pkg/mission/object/initialize/ 2>&1)" || {
    echo "[evaluate_plane_costs] ERROR: go test failed:" >&2
    echo "$GO_TEST_OUTPUT" >&2
    exit 1
}

# 步骤 3 & 4：解析 JSON 并套用花费公式
echo "[evaluate_plane_costs] calculating costs..." >&2

echo "$GO_TEST_OUTPUT" | python3 -c "
import json, sys, math

scale_factor = $SCALE_FACTOR
funds_min = $FUNDS_MIN
funds_max = $FUNDS_MAX
time_min = $TIME_MIN
time_max = $TIME_MAX

type_multipliers = {
    'fighter': 1.00,
    'dive_bomber': 1.15,
    'torpedo_bomber': 1.30,
}

records = []
for line in sys.stdin:
    line = line.strip()
    # 查找 JSON 行：格式为 '    plane_cost_data_test.go:NN: {...}'
    if 'plane_cost_data_test.go:' not in line:
        continue
    idx = line.index('plane_cost_data_test.go:')
    after = line[idx:]
    colon_idx = after.index(':') + 1
    json_str = after[colon_idx:].strip()
    try:
        rec = json.loads(json_str)
    except json.JSONDecodeError:
        continue
    records.append(rec)

if not records:
    print('ERROR: no plane data extracted from test output', file=sys.stderr)
    sys.exit(1)

records.sort(key=lambda r: r['name'])

print('name\ttype\tnation\tcombatPower\ttonnage\tfundsCost\ttimeCost')

for rec in records:
    t = rec.get('type', 'fighter')
    cp = int(rec.get('combatPower', 0))
    multi = type_multipliers.get(t, 1.0)

    raw = cp * multi * scale_factor
    funds = int(round(raw / 5) * 5)
    funds = max(funds_min, min(funds_max, funds))
    time_cost = int(round(funds * 0.35 + 2))
    time_cost = max(time_min, min(time_max, time_cost))

    print(f\"{rec['name']}\t{rec['type']}\t{rec['nation']}\t{cp}\t{rec['tonnage']}\t{funds}\t{time_cost}\")

min_cp = min(r.get('combatPower', 0) for r in records)
max_cp = max(r.get('combatPower', 0) for r in records)
print(f'[evaluate_plane_costs] combat power range: {min_cp} - {max_cp}', file=sys.stderr)
print(f'[evaluate_plane_costs] {len(records)} planes evaluated', file=sys.stderr)
"

echo "[evaluate_plane_costs] done." >&2

