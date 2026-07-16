#!/usr/bin/env python3
"""Evaluate ship funds and reinforcement time from runtime combat power."""

import argparse
import json
import math
from pathlib import Path

import json5

REPO_ROOT = Path(__file__).resolve().parents[4]

COMBAT_BASE_COST = {
    "torpedo_boat": 5,
    "destroyer": 20,
    "frigate": 15,
    "cruiser": 40,
    "battleship": 230,
    "aircraft_carrier": 80,
    "cargo": 20,
}
COMBAT_POWER_FACTOR = {
    "torpedo_boat": 1.20,
    "destroyer": 0.85,
    "frigate": 0.60,
    "cruiser": 0.65,
    "battleship": 0.55,
    "aircraft_carrier": 0.55,
    "cargo": 0.80,
}
HULL_FLOOR_MULTIPLIER = {
    "default": 0.00,
    "torpedo_boat": 0.10,
    "destroyer": 0.18,
    "frigate": 0.15,
    "cruiser": 0.35,
    "battleship": 0.80,
    "aircraft_carrier": 0.65,
    "cargo": 0.30,
    "hospital": 0.50,
}
HULL_FLOOR_SCALE = 3.0
BURST_POWER_WEIGHT = 0.25
PROJECTION_POWER_WEIGHT = 0.10
HOSPITAL_UTILITY_COST = 25
COST_STEP = 5
# 战力公式无法完整表达科技树层级时，使用少量显式战略溢价。
# 萨摩应贴近大和；江户作为日本终局舰需要明显高于大和。
STRATEGIC_COST_MULTIPLIER = {
    "satsuma": 1.10,
    "edo": 1.15,
}

DEFAULT_TIME_FACTOR = 0.35
DEFAULT_TIME_BASE = 2
DEFAULT_TIME_MAX = 130
BATTLESHIP_TIME_FACTOR = 0.13
BATTLESHIP_TIME_BASE = 40
BATTLESHIP_TIME_MIN = 85
BATTLESHIP_TIME_MAX = 180

AIR_FIT_FACTOR = 0.12
AIR_TYPE_PENALTY = 1.5
AIR_FIT_MAX = 18
NATION_AIR_FIT_MULTIPLIER = {
    "us": 0.75,
    "jp": 1.00,
    "de": 1.00,
    "uk": 0.90,
    "ru": 1.05,
    "su": 1.05,
    "cn": 0.50,
    "special": 1.00,
}


def round_to_step(value, step):
    return int(round(value / step) * step)


def aircraft_total(ship, plane_costs):
    return sum(
        group.get("maxCount", 0) * plane_costs.get(group.get("name", ""), 10)
        for group in ship.get("aircraft", {}).get("groups", [])
    )


def air_wing_fit_penalty(ship, plane_times):
    groups = [
        group
        for group in ship.get("aircraft", {}).get("groups", [])
        if group.get("maxCount", 0) > 0
    ]
    count = sum(group.get("maxCount", 0) for group in groups)
    if count <= 0:
        return 0
    weighted_time = sum(
        group.get("maxCount", 0)
        * plane_times.get(group.get("name", ""), 6)
        for group in groups
    )
    average_time = weighted_time / count
    raw = (
        math.sqrt(count) * average_time * AIR_FIT_FACTOR
        + max(0, len({group.get("name", "") for group in groups}) - 1)
        * AIR_TYPE_PENALTY
    )
    return max(0, min(AIR_FIT_MAX, int(round(raw))))


def hull_floor_cost(ship_type, effective_hp):
    raw = (
        math.pow(max(float(effective_hp), 1), 0.45)
        * HULL_FLOOR_MULTIPLIER.get(
            ship_type,
            HULL_FLOOR_MULTIPLIER["cruiser"],
        )
        * HULL_FLOOR_SCALE
    )
    return max(0, round_to_step(raw, COST_STEP))


def ship_time_cost(ship, funds, fit_penalty):
    ship_type = ship.get("type", "cruiser")
    if ship_type == "battleship":
        base_time = max(
            BATTLESHIP_TIME_MIN,
            min(
                BATTLESHIP_TIME_MAX,
                round(BATTLESHIP_TIME_BASE + BATTLESHIP_TIME_FACTOR * funds),
            ),
        )
        time_max = BATTLESHIP_TIME_MAX
    else:
        base_time = max(
            3,
            min(
                DEFAULT_TIME_MAX,
                round(DEFAULT_TIME_BASE + DEFAULT_TIME_FACTOR * funds),
            ),
        )
        time_max = DEFAULT_TIME_MAX

    fit_multiplier = NATION_AIR_FIT_MULTIPLIER.get(
        ship.get("nation", "jp"),
        1.0,
    )
    return max(
        3,
        min(time_max, int(round(base_time + fit_penalty * fit_multiplier))),
    )


def calculate_ship_cost(ship, power, plane_costs, plane_times):
    aircraft = aircraft_total(ship, plane_costs)
    if ship.get("nation") == "special":
        funds = int(ship.get("fundsCost", 0))
        return {
            "hull_floor": 0,
            "economic_power": 0,
            "combat_cost": 0,
            "aircraft": aircraft,
            "funds": funds,
            "total": funds + aircraft,
            "fit_penalty": 0,
            "time": int(ship.get("timeCost", 3)),
        }

    ship_type = ship.get("type", "cruiser")
    hull_floor = hull_floor_cost(ship_type, power.get("effectiveHP", 0))
    if ship_type == "hospital":
        economic_power = 0
        combat_cost = 0
        funds = round_to_step(
            hull_floor + HOSPITAL_UTILITY_COST,
            COST_STEP,
        )
    else:
        economic_power = float(power.get("hullPower", 0))
        # 含舰载机单位的 Burst / Projection 已混入航空贡献，避免重复计价。
        if int(power.get("aviation", 0)) <= 0:
            economic_power += (
                BURST_POWER_WEIGHT * float(power.get("burst", 0))
                + PROJECTION_POWER_WEIGHT
                * float(power.get("projection", 0))
            )
        combat_cost = round_to_step(
            COMBAT_BASE_COST.get(ship_type, 20)
            + COMBAT_POWER_FACTOR.get(ship_type, 0.65) * economic_power,
            COST_STEP,
        )
        funds = max(5, hull_floor, combat_cost)

    strategic_multiplier = STRATEGIC_COST_MULTIPLIER.get(ship["name"], 1.0)
    funds = round_to_step(funds * strategic_multiplier, COST_STEP)
    fit_penalty = air_wing_fit_penalty(ship, plane_times)
    return {
        "hull_floor": hull_floor,
        "economic_power": economic_power,
        "combat_cost": combat_cost,
        "strategic_multiplier": strategic_multiplier,
        "aircraft": aircraft,
        "funds": funds,
        "total": funds + aircraft,
        "fit_penalty": fit_penalty,
        "time": ship_time_cost(ship, funds, fit_penalty),
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--combat-power",
        type=Path,
        required=True,
        help="runtime ship combat-power JSON exported by the Go helper",
    )
    parser.add_argument(
        "--output",
        type=Path,
        required=True,
        help="write the evaluated ship cost map to this path",
    )
    args = parser.parse_args()
    combat_power_path = args.combat_power
    if not combat_power_path.is_absolute():
        combat_power_path = REPO_ROOT / combat_power_path
    output_path = args.output
    if not output_path.is_absolute():
        output_path = REPO_ROOT / output_path

    with combat_power_path.open() as file:
        powers = {
            record["name"]: record for record in json.load(file)
        }
    with (REPO_ROOT / "configs/ships.json5").open() as file:
        ships = json5.loads(file.read())

    plane_costs = {}
    plane_times = {}
    with (REPO_ROOT / "configs/planes.json5").open() as file:
        for plane in json5.loads(file.read()):
            plane_costs[plane["name"]] = plane.get("fundsCost", 10)
            plane_times[plane["name"]] = plane.get("timeCost", 6)

    records = []
    cost_map = {}
    for ship in ships:
        power = powers.get(ship["name"])
        if power is None:
            raise KeyError(f"missing runtime combat power for {ship['name']}")
        result = calculate_ship_cost(
            ship,
            power,
            plane_costs,
            plane_times,
        )
        records.append((ship, power, result))
        cost_map[ship["name"]] = result

    records.sort(key=lambda record: record[2]["total"])
    print(
        f"{'name':20s} {'type':16s} {'nat':3s} {'CP':>5s} {'burst':>5s} "
        f"{'floor':>5s} {'combat':>6s} {'funds old>new':>15s} "
        f"{'air':>6s} {'total':>6s} {'time old>new':>13s}"
    )
    print("-" * 116)
    for ship, power, result in records:
        print(
            f"{ship['name']:20s} {ship.get('type','?'):16s} "
            f"{ship.get('nation','jp'):3s} {power.get('hullPower',0):5d} "
            f"{power.get('burst',0):5d} {result['hull_floor']:4d}$ "
            f"{result['combat_cost']:5d}$ "
            f"{ship.get('fundsCost',0):5d}>{result['funds']:<5d}$ "
            f"{result['aircraft']:5d}$ {result['total']:5d}$ "
            f"{ship.get('timeCost',0):4d}>{result['time']:<4d}s"
        )

    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w") as file:
        json.dump(cost_map, file, indent=2, sort_keys=True)
    print(f"\nSaved {len(cost_map)} ship costs to {output_path}")


if __name__ == "__main__":
    main()
