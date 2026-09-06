#!/usr/bin/env python3
"""Apply a caller-selected evaluated cost map to ships.json5."""

import argparse
import json
import re
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[4]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--costs",
        type=Path,
        required=True,
        help="evaluated ship cost map to apply",
    )
    args = parser.parse_args()
    costs_path = args.costs
    if not costs_path.is_absolute():
        costs_path = REPO_ROOT / costs_path

    with costs_path.open() as file:
        cost_map = json.load(file)

    ships_path = REPO_ROOT / "configs/ships.json5"
    with ships_path.open() as file:
        content = file.read()

    lines = content.split("\n")
    updated = []
    changed = 0
    for index, line in enumerate(lines):
        if line.strip() in (
            "// TODO 确认资金",
            "// TODO 确认时间",
            "// TODO",
            "// FIXME 费用待定",
        ):
            continue

        funds_match = re.match(
            r"^(\s{4})fundsCost:\s*\d+(\s*,?\s*)$",
            line,
        )
        time_match = re.match(
            r"^(\s{4})timeCost:\s*\d+(\s*,?\s*)$",
            line,
        )
        if funds_match or time_match:
            for previous_index in range(index - 1, -1, -1):
                name_match = re.match(
                    r'^\s{4}name:\s*"([^"]+)"',
                    lines[previous_index],
                )
                if not name_match or name_match.group(1) not in cost_map:
                    continue
                result = cost_map[name_match.group(1)]
                if funds_match:
                    line = (
                        f"{funds_match.group(1)}fundsCost: "
                        f"{result['funds']}{funds_match.group(2)}"
                    )
                else:
                    line = (
                        f"{time_match.group(1)}timeCost: "
                        f"{result['time']}{time_match.group(2)}"
                    )
                changed += 1
                break
        updated.append(line)

    with ships_path.open("w") as file:
        file.write("\n".join(updated).rstrip() + "\n")

    print(f"Updated {changed} ship cost fields across {len(cost_map)} ships")


if __name__ == "__main__":
    main()
