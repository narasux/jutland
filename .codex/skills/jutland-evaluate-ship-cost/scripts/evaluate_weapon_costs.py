#!/usr/bin/env python3
"""Evaluate gun, torpedo-launcher, and rocket-launcher reference costs."""

import argparse
import json
import math
import re
from pathlib import Path

import json5

REPO_ROOT = Path(__file__).resolve().parents[4]

WEAPON_TYPE_FACTORS = {
    "gun": 0.50,
    "torpedo": 1.00,
    "rocket": 0.70,
}
SUSTAINED_WEIGHT = 0.70
BURST_WEIGHT = 0.30
REFERENCE_CYCLES = {
    "gun": 20.0,
    "torpedo": 60.0,
    "rocket": 60.0,
}
COST_MIN = 1
COST_MAX = 100
COST_STEP = 5
REFERENCE_SCORE_PER_FUND = 5.0

TARGETS = (
    ("GUN ", "configs/guns.json5", "guns", "bulletCount", "gun"),
    (
        "TORP",
        "configs/torpedo_launchers.json5",
        "torpedoes",
        "bulletCount",
        "torpedo",
    ),
    (
        "RKT ",
        "configs/rocket_launchers.json5",
        "rockets",
        "rocketCount",
        "rocket",
    ),
)


def round_to_step(value, step):
    return int(math.floor(value / step + 0.5) * step)


def launcher_cycle(launcher, count_key, weapon_type):
    count = launcher.get(count_key, 1)
    if weapon_type == "gun":
        return launcher.get("reloadTime", 1)
    if weapon_type == "torpedo":
        return launcher.get("reloadTime", 1) + max(
            0, count - 1
        ) * launcher.get("shotInterval", 0)

    group_count = min(max(1, launcher.get("groupCount", 1)), count)
    group_size = math.ceil(count / group_count)
    actual_groups = math.ceil(count / group_size)
    return (
        launcher.get("reloadTime", 1)
        + (count - actual_groups) * launcher.get("shotInterval", 0)
        + max(0, actual_groups - 1) * launcher.get("groupInterval", 0)
    )


def expected_damage(bullets, bullet_name):
    bullet = bullets.get(bullet_name, {})
    damage = float(bullet.get("damage", 0))
    critical_rate = float(bullet.get("criticalRate", 0))
    return damage * (1 + 2.7 * critical_rate)


def weapon_unit_cost(damage, count, cycle, weapon_type):
    if damage <= 0 or count <= 0 or cycle <= 0:
        return COST_MIN
    salvo_damage = damage * count
    effective_rate = salvo_damage * (
        SUSTAINED_WEIGHT / cycle
        + BURST_WEIGHT / REFERENCE_CYCLES[weapon_type]
    )
    # 原始输出是用于比较强弱的战力分，除以 5 后映射到游戏资金尺度。
    raw = (
        effective_rate
        * WEAPON_TYPE_FACTORS[weapon_type]
        / REFERENCE_SCORE_PER_FUND
    )
    return max(
        COST_MIN,
        min(COST_MAX, round_to_step(raw, COST_STEP)),
    )


def evaluate_target(bullets, relative_path, count_key, weapon_type):
    with (REPO_ROOT / relative_path).open() as file:
        weapons = json5.loads(file.read())

    costs = {}
    rows = []
    for weapon in weapons:
        count = weapon.get(count_key, 1)
        cycle = launcher_cycle(weapon, count_key, weapon_type)
        damage = expected_damage(bullets, weapon["bulletName"])
        # 无阵营路径的彩蛋武器保留手工定价，但同样受 $1–100 全局范围约束。
        cost = max(
            COST_MIN,
            min(COST_MAX, weapon.get("fundsCost", COST_MIN)),
        )
        if "/" in weapon["name"]:
            cost = weapon_unit_cost(damage, count, cycle, weapon_type)
        costs[weapon["name"]] = cost
        rows.append(
            (
                weapon["name"],
                weapon.get("fundsCost", COST_MIN),
                cost,
                damage,
                count,
                cycle,
            )
        )
    return costs, rows


def apply_costs(relative_path, costs):
    path = REPO_ROOT / relative_path
    with path.open() as file:
        content = file.read()

    lines = content.split("\n")
    updated = []
    changed = 0
    for index, line in enumerate(lines):
        cost_match = re.match(r"^(\s{4})fundsCost:\s*\d+(\s*,?\s*)$", line)
        if cost_match:
            for previous_index in range(index - 1, -1, -1):
                name_match = re.match(
                    r'^\s{4}name:\s*"([^"]+)"',
                    lines[previous_index],
                )
                if not name_match or name_match.group(1) not in costs:
                    continue
                new_cost = costs[name_match.group(1)]
                line = (
                    f"{cost_match.group(1)}fundsCost: "
                    f"{new_cost}{cost_match.group(2)}"
                )
                changed += 1
                break
        updated.append(line)

    with path.open("w") as file:
        file.write("\n".join(updated).rstrip() + "\n")
    return changed


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--apply",
        action="store_true",
        help="write evaluated costs to weapon configuration files",
    )
    parser.add_argument(
        "--output",
        type=Path,
        help="write the evaluated cost map to this caller-selected path",
    )
    args = parser.parse_args()
    output_path = args.output
    if output_path and not output_path.is_absolute():
        output_path = REPO_ROOT / output_path

    with (REPO_ROOT / "configs/bullets.json5").open() as file:
        bullets = {
            bullet["name"]: bullet for bullet in json5.loads(file.read())
        }

    all_costs = {}
    changed_total = 0
    for label, relative_path, key, count_key, weapon_type in TARGETS:
        costs, rows = evaluate_target(
            bullets,
            relative_path,
            count_key,
            weapon_type,
        )
        all_costs[key] = costs
        for name, old_cost, new_cost, damage, count, cycle in rows:
            print(
                f"{label} {name:30s} exp={damage:7.0f} ct={count:2d} "
                f"cyc={cycle:6.1f}s old=${old_cost:3d} new=${new_cost:3d}"
            )
        print()
        if args.apply:
            changed_total += apply_costs(relative_path, costs)

    if output_path:
        output_path.parent.mkdir(parents=True, exist_ok=True)
        with output_path.open("w") as file:
            json.dump(all_costs, file, indent=2, sort_keys=True)
        print(
            "Saved "
            f"{len(all_costs['guns'])} guns + "
            f"{len(all_costs['torpedoes'])} torpedoes + "
            f"{len(all_costs['rockets'])} rockets "
            f"to {output_path}"
        )
    if args.apply:
        print(f"Updated {changed_total} weapon costs")


if __name__ == "__main__":
    main()
