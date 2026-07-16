#!/usr/bin/env bash
# evaluate_plane_costs.sh — 导出飞机运行时战力并计算 fundsCost / timeCost。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
DATA_FILE="$(mktemp)"
trap 'rm -f "$DATA_FILE"' EXIT

# 花费公式参数（与 SKILL.md 保持同步）
SCALE_FACTOR="0.10"
FUNDS_MIN=3
FUNDS_MAX=30
TIME_MIN=3
TIME_MAX=10

echo "[evaluate_plane_costs] exporting runtime combat power..." >&2
cd "$REPO_ROOT"
go run "$SCRIPT_DIR/../export_plane_cost_data.go" > "$DATA_FILE"

echo "[evaluate_plane_costs] calculating costs..." >&2
python3 - "$DATA_FILE" "$SCALE_FACTOR" "$FUNDS_MIN" "$FUNDS_MAX" "$TIME_MIN" "$TIME_MAX" <<'PY'
import json
import sys

data_file = sys.argv[1]
scale_factor = float(sys.argv[2])
funds_min = int(sys.argv[3])
funds_max = int(sys.argv[4])
time_min = int(sys.argv[5])
time_max = int(sys.argv[6])

type_multipliers = {
    "fighter": 1.00,
    "dive_bomber": 1.15,
    "torpedo_bomber": 1.30,
}

with open(data_file) as file:
    records = json.load(file)
if not records:
    raise SystemExit("no plane combat-power records exported")

print("name\ttype\tnation\tcombatPower\ttonnage\tfundsCost\ttimeCost")
for record in records:
    plane_type = record.get("type", "fighter")
    combat_power = int(record.get("combatPower", 0))
    multiplier = type_multipliers.get(plane_type, 1.0)
    funds = round(combat_power * multiplier * scale_factor)
    funds = max(funds_min, min(funds_max, funds))
    time_cost = round(funds * 0.35 + 2)
    time_cost = max(time_min, min(time_max, time_cost))
    print(
        f"{record['name']}\t{record['type']}\t{record['nation']}\t"
        f"{combat_power}\t{record['tonnage']}\t{funds}\t{time_cost}"
    )

values = [record.get("combatPower", 0) for record in records]
print(
    f"[evaluate_plane_costs] combat power range: {min(values)} - {max(values)}",
    file=sys.stderr,
)
print(f"[evaluate_plane_costs] {len(records)} planes evaluated", file=sys.stderr)
PY

echo "[evaluate_plane_costs] done." >&2
