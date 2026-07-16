#!/usr/bin/env bash
# evaluate_ship_costs.sh — audit weapons, export runtime ship power, and propose costs.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
OUTPUT_DIR="${1:-$(mktemp -d)}"
if [[ "$OUTPUT_DIR" != /* ]]; then
  OUTPUT_DIR="$REPO_ROOT/$OUTPUT_DIR"
fi
WEAPON_COSTS_FILE="$OUTPUT_DIR/weapon_costs.json"
SHIP_COMBAT_POWER_FILE="$OUTPUT_DIR/ship_combat_power.json"
SHIP_COSTS_FILE="$OUTPUT_DIR/ship_costs.json"

mkdir -p "$OUTPUT_DIR"

echo "=== Step 1: Weapon reference costs ==="
python3 "$SCRIPT_DIR/evaluate_weapon_costs.py" \
  --output "$WEAPON_COSTS_FILE"

echo ""
echo "=== Step 2: Runtime ship combat power ==="
cd "$REPO_ROOT"
go run "$SCRIPT_DIR/../export_ship_cost_data.go" > "$SHIP_COMBAT_POWER_FILE"
python3 - "$SHIP_COMBAT_POWER_FILE" <<'PY'
import json
import sys

data_file = sys.argv[1]
with open(data_file) as file:
    records = json.load(file)
if not records:
    raise SystemExit("no ship combat-power records exported")
print(f"Exported {len(records)} ships to {data_file}")
PY

echo ""
echo "=== Step 3: Ship cost proposal ==="
python3 "$SCRIPT_DIR/evaluate_ship_costs.py" \
  --combat-power "$SHIP_COMBAT_POWER_FILE" \
  --output "$SHIP_COSTS_FILE"

echo ""
echo "=== Done: review before applying ==="
echo "Weapons: python3 $SCRIPT_DIR/evaluate_weapon_costs.py --apply"
echo "Ships:   python3 $SCRIPT_DIR/apply_ship_costs.py --costs $SHIP_COSTS_FILE"
echo "Audit files: $OUTPUT_DIR"
