#!/usr/bin/env python3
"""
plane_cost_calc.py — reads planes.json5 and calculates fundsCost / timeCost using
a combat-power estimate derived from configurable plane attributes.

Output TSV: name	type	nation	tonnage	fundsCost	timeCost	cpEstimate
"""

import json
import math
import re
import sys

# Cost formula parameters (kept in sync with SKILL.md)
SCALE_FACTOR = 0.10
FUNDS_MIN = 3
FUNDS_MAX = 30
TIME_MIN = 3
TIME_MAX = 10

TYPE_MULTIPLIERS = {
    'fighter': 1.00,
    'dive_bomber': 1.15,
    'torpedo_bomber': 1.30,
}


def strip_json5_comments(text: str) -> str:
    """Remove // line comments and /* */ block comments from JSON5."""
    text = re.sub(r'/\*.*?\*/', '', text, flags=re.DOTALL)
    lines = []
    for line in text.split('\n'):
        in_string = False
        result = []
        i = 0
        while i < len(line):
            ch = line[i]
            if ch == '"' and (i == 0 or line[i - 1] != '\\'):
                in_string = not in_string
                result.append(ch)
            elif ch == '/' and i + 1 < len(line) and line[i + 1] == '/' and not in_string:
                break
            else:
                result.append(ch)
            i += 1
        lines.append(''.join(result))
    return '\n'.join(lines)


def parse_planes_json5(path: str) -> list:
    """Parse planes.json5 and return plane entries."""
    with open(path, 'r') as f:
        raw = f.read()
    clean = strip_json5_comments(raw)
    return json.loads(clean)


def estimate_combat_power(plane: dict) -> float:
    """Estimate relative combat power from configurable attributes.

    Used for cost stratification, not expected to match combatpower.CalculatePlane exactly.
    Formula: HP * speedFactor * rangeFactor * weaponFactor
    """
    hp = float(plane.get('totalHP', 0))
    speed = float(plane.get('maxSpeed', 1))
    plane_range = float(plane.get('range', 1))

    speed_factor = math.sqrt(max(speed / 1500, 0.1))
    range_factor = math.sqrt(max(plane_range / 2000, 0.1))

    weapon = plane.get('weapon', {})
    gun_count = len(weapon.get('guns', []))
    bomb_count = len(weapon.get('bombs', []))
    torpedo_count = len(weapon.get('torpedoes', []))
    rocket_count = len(weapon.get('rockets', []))
    weapon_factor = 1.0 + 0.1 * gun_count + 0.3 * bomb_count + 0.4 * torpedo_count + 0.2 * rocket_count

    return hp * speed_factor * range_factor * weapon_factor


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 plane_cost_calc.py <planes.json5>", file=sys.stderr)
        sys.exit(1)

    path = sys.argv[1]
    planes = parse_planes_json5(path)

    records = []
    for plane in planes:
        name = plane.get('name', '')
        ptype = plane.get('type', 'fighter')
        nation = plane.get('nation', '')
        tonnage = float(plane.get('tonnage', 0))

        cp_est = estimate_combat_power(plane)
        multi = TYPE_MULTIPLIERS.get(ptype, 1.0)

        raw = cp_est * multi * SCALE_FACTOR
        funds = int(round(raw / 5) * 5)
        funds = max(FUNDS_MIN, min(FUNDS_MAX, funds))
        time_cost = int(round(funds * 0.35 + 2))
        time_cost = max(TIME_MIN, min(TIME_MAX, time_cost))

        records.append((name, ptype, nation, tonnage, funds, time_cost, cp_est))

    records.sort(key=lambda r: r[0])

    print('name\ttype\tnation\ttonnage\tfundsCost\ttimeCost\tcpEstimate')
    for rec in records:
        name, ptype, nation, tonnage, funds, time_cost, cp_est = rec
        print(f'{name}\t{ptype}\t{nation}\t{tonnage}\t{funds}\t{time_cost}\t{cp_est:.1f}')

    min_cp = min(r[6] for r in records)
    max_cp = max(r[6] for r in records)
    print(f'[plane_cost_calc] estimated cp range: {min_cp:.1f} – {max_cp:.1f}', file=sys.stderr)
    print(f'[plane_cost_calc] {len(records)} planes evaluated', file=sys.stderr)


if __name__ == '__main__':
    main()

