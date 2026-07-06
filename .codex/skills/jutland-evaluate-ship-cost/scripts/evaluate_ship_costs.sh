#!/usr/bin/env bash
# evaluate_ship_costs.sh — compute weapon costs, then ship costs.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== Step 1: Weapon costs ==="
python3 "$SCRIPT_DIR/evaluate_weapon_costs.py"

echo ""
echo "=== Step 2: Ship costs ==="
python3 "$SCRIPT_DIR/evaluate_ship_costs.py"

echo ""
echo "=== Done ==="
echo "Run: python3 $SCRIPT_DIR/apply_ship_costs.py   to write costs to ships.json5"
