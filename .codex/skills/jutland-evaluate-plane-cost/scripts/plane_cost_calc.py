#!/usr/bin/env python3
"""
plane_cost_calc.py — reads planes.json5 and calculates fundsCost / timeCost using
a combat-power estimate derived from configurable plane attributes.

Output TSV: name	type	nation	tonnage	fundsCost	timeCost	cpEstimate
"""

import math
import re
import sys

import json5

# Fallback formula parameters (kept in sync with SKILL.md)
SCALE_FACTOR = 0.30
FUNDS_MIN = 3
FUNDS_MAX = 30
TIME_MIN = 3
TIME_MAX = 10

TYPE_MULTIPLIERS = {
    'fighter': 1.00,
    'dive_bomber': 1.15,
    'torpedo_bomber': 1.30,
}


def parse_planes_json5(path: str) -> list:
    """Parse planes.json5 and return plane entries."""
    with open(path, 'r') as f:
        return json5.loads(f.read())


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


def calc_cost(plane: dict) -> tuple[int, int, float]:
    cp_est = estimate_combat_power(plane)
    multi = TYPE_MULTIPLIERS.get(plane.get('type', 'fighter'), 1.0)

    raw = cp_est * multi * SCALE_FACTOR
    funds = int(round(raw))
    funds = max(FUNDS_MIN, min(FUNDS_MAX, funds))
    time_cost = int(round(funds * 0.35 + 2))
    time_cost = max(TIME_MIN, min(TIME_MAX, time_cost))
    return funds, time_cost, cp_est


def apply_costs(path: str, records: list[tuple[str, int, int]]) -> None:
    cost_map = {name: (funds, time_cost) for name, funds, time_cost in records}
    with open(path, 'r') as f:
        content = f.read()

    lines = content.split('\n')
    result = []
    for i, line in enumerate(lines):
        if line.strip() in ('// TODO 确认资金', '// TODO 确认时间', '// TODO', '// FIXME 费用待定'):
            continue
        mf = re.match(r'^(\s{4})fundsCost:\s*\d+(\s*,?\s*)$', line)
        mt = re.match(r'^(\s{4})timeCost:\s*\d+(\s*,?\s*)$', line)
        if mf:
            for j in range(i - 1, -1, -1):
                nm = re.match(r'^\s{4}name:\s*"([^"]+)"', lines[j])
                if nm and nm.group(1) in cost_map:
                    line = f'{mf.group(1)}fundsCost: {cost_map[nm.group(1)][0]}{mf.group(2)}'
                    break
        elif mt:
            for j in range(i - 1, -1, -1):
                nm = re.match(r'^\s{4}name:\s*"([^"]+)"', lines[j])
                if nm and nm.group(1) in cost_map:
                    line = f'{mt.group(1)}timeCost: {cost_map[nm.group(1)][1]}{mt.group(2)}'
                    break
        result.append(line)

    with open(path, 'w') as f:
        f.write('\n'.join(result).rstrip() + '\n')


def main():
    if len(sys.argv) < 2:
        print("Usage: python3 plane_cost_calc.py <planes.json5> [--apply]", file=sys.stderr)
        sys.exit(1)

    path = sys.argv[1]
    apply = '--apply' in sys.argv[2:]
    planes = parse_planes_json5(path)

    records = []
    apply_records = []
    for plane in planes:
        name = plane.get('name', '')
        ptype = plane.get('type', 'fighter')
        nation = plane.get('nation', '')
        tonnage = float(plane.get('tonnage', 0))

        funds, time_cost, cp_est = calc_cost(plane)

        records.append((name, ptype, nation, tonnage, funds, time_cost, cp_est))
        apply_records.append((name, funds, time_cost))

    records.sort(key=lambda r: r[0])
    apply_records.sort(key=lambda r: r[0])

    print('name\ttype\tnation\ttonnage\tfundsCost\ttimeCost\tcpEstimate')
    for rec in records:
        name, ptype, nation, tonnage, funds, time_cost, cp_est = rec
        print(f'{name}\t{ptype}\t{nation}\t{tonnage}\t{funds}\t{time_cost}\t{cp_est:.1f}')

    min_cp = min(r[6] for r in records)
    max_cp = max(r[6] for r in records)
    print(f'[plane_cost_calc] estimated cp range: {min_cp:.1f} – {max_cp:.1f}', file=sys.stderr)
    print(f'[plane_cost_calc] {len(records)} planes evaluated', file=sys.stderr)
    if apply:
        apply_costs(path, apply_records)
        print(f'[plane_cost_calc] updated {len(apply_records)} planes in {path}', file=sys.stderr)


if __name__ == '__main__':
    main()
