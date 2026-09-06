#!/usr/bin/env python3
"""Validate a Jutland PNG/.map pair and optionally render a runtime-style preview."""

from __future__ import annotations

import argparse
import hashlib
from collections import Counter
from pathlib import Path

from PIL import Image


Image.MAX_IMAGE_PIXELS = None
VALID_CHARS = frozenset(".OSCL")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", required=True, type=Path)
    parser.add_argument("--map", required=True, dest="map_path", type=Path)
    parser.add_argument("--cell-pixels", type=int, required=True)
    parser.add_argument(
        "--expected-width",
        type=int,
        default=4096,
        help="required runtime PNG width (default: Jutland standard 4096)",
    )
    parser.add_argument(
        "--expected-height",
        type=int,
        required=True,
        help="planned runtime PNG height; declare it explicitly for every map",
    )
    parser.add_argument("--expected-grid-width", type=int, required=True)
    parser.add_argument("--expected-grid-height", type=int, required=True)
    parser.add_argument("--preview", type=Path)
    parser.add_argument("--preview-cell-size", type=int, default=16)
    parser.add_argument(
        "--sea-dir",
        type=Path,
        default=Path("resources/images/map/blocks/sea"),
    )
    parser.add_argument(
        "--deep-sea-dir",
        type=Path,
        default=Path("resources/images/map/blocks/deep_sea"),
    )
    return parser.parse_args()


def load_base_tiles(path: Path, size: int) -> list[Image.Image]:
    files = sorted(path.glob("64_*.png"), key=lambda item: int(item.stem.split("_")[-1]))
    return [
        Image.open(file).convert("RGBA").resize((size, size), Image.Resampling.NEAREST)
        for file in files
    ]


def render_preview(
    source: Image.Image,
    rows: list[str],
    args: argparse.Namespace,
) -> None:
    size = args.preview_cell_size
    if size <= 0:
        raise SystemExit("--preview-cell-size must be positive")
    sea = load_base_tiles(args.sea_dir, size)
    deep_sea = load_base_tiles(args.deep_sea_dir, size)
    if not sea:
        raise SystemExit(f"no shallow-sea tiles found in {args.sea_dir}")
    if any("O" in row for row in rows) and not deep_sea:
        raise SystemExit(f"no deep-sea tiles found in {args.deep_sea_dir}")

    preview = Image.new("RGBA", (len(rows[0]) * size, len(rows) * size), (0, 0, 0, 255))
    for y, row in enumerate(rows):
        for x, char in enumerate(row):
            cell = Image.new("RGBA", (size, size), (0, 0, 0, 255))
            digest = hashlib.md5(f"{x}:{y}".encode("utf-8")).digest()[0]
            if char in ".SC":
                cell.alpha_composite(sea[digest % len(sea)])
            elif char == "O":
                cell.alpha_composite(deep_sea[digest % len(deep_sea)])
            if char in "SCL":
                box = (
                    x * args.cell_pixels,
                    y * args.cell_pixels,
                    (x + 1) * args.cell_pixels,
                    (y + 1) * args.cell_pixels,
                )
                overlay = source.crop(box).resize((size, size), Image.Resampling.NEAREST)
                cell.alpha_composite(overlay)
            preview.alpha_composite(cell, (x * size, y * size))

    args.preview.parent.mkdir(parents=True, exist_ok=True)
    preview.convert("RGB").save(args.preview)
    print(f"preview={args.preview}")


def main() -> None:
    args = parse_args()
    if args.cell_pixels <= 0:
        raise SystemExit("--cell-pixels must be positive")
    if args.expected_width <= 0 or args.expected_height <= 0:
        raise SystemExit("--expected-width and --expected-height must be positive")
    if args.expected_grid_width <= 0 or args.expected_grid_height <= 0:
        raise SystemExit("--expected-grid-width and --expected-grid-height must be positive")
    if not args.image.is_file() or not args.map_path.is_file():
        raise SystemExit("--image and --map must both exist")

    with Image.open(args.image) as opened:
        source_mode = opened.mode
        source = opened.convert("RGBA")
    rows = args.map_path.read_text(encoding="utf-8").splitlines()
    errors: list[str] = []
    if source_mode != "RGBA":
        errors.append(f"image mode is {source_mode}, expected RGBA")
    declared_size = (args.expected_width, args.expected_height)
    if source.size != declared_size:
        errors.append(f"image size={source.size}, declared target={declared_size}")
    if not rows:
        errors.append("map is empty")
        width = 0
    else:
        width = len(rows[0])
        for y, row in enumerate(rows):
            if len(row) != width:
                errors.append(f"row {y} width={len(row)}, expected {width}")
            unknown = sorted(set(row) - VALID_CHARS)
            if unknown:
                errors.append(f"row {y} has invalid characters: {unknown}")

    declared_grid_size = (args.expected_grid_width, args.expected_grid_height)
    actual_grid_size = (width, len(rows))
    if actual_grid_size != declared_grid_size:
        errors.append(f"map grid={actual_grid_size}, declared grid={declared_grid_size}")

    expected_size = (width * args.cell_pixels, len(rows) * args.cell_pixels)
    if source.size != expected_size:
        errors.append(f"image size={source.size}, expected={expected_size}")

    visible_skipped: list[tuple[int, int]] = []
    partial_land: list[tuple[int, int]] = []
    empty_overlay: list[tuple[int, int]] = []
    if not errors:
        alpha = source.getchannel("A")
        for y, row in enumerate(rows):
            for x, char in enumerate(row):
                box = (
                    x * args.cell_pixels,
                    y * args.cell_pixels,
                    (x + 1) * args.cell_pixels,
                    (y + 1) * args.cell_pixels,
                )
                min_alpha, max_alpha = alpha.crop(box).getextrema()
                if char in ".O" and max_alpha > 0:
                    visible_skipped.append((x, y))
                elif char == "L" and min_alpha < 255:
                    partial_land.append((x, y))
                elif char in "SC" and max_alpha == 0:
                    empty_overlay.append((x, y))

    print(f"image={source.size[0]}x{source.size[1]} mode={source_mode}")
    print(f"declared-target={args.expected_width}x{args.expected_height}")
    print(f"declared-grid={args.expected_grid_width}x{args.expected_grid_height}")
    print(f"grid={width}x{len(rows)} cell-pixels={args.cell_pixels}")
    print(f"terrain={dict(sorted(Counter(''.join(rows)).items()))}")
    print(f"visible-skipped={len(visible_skipped)}")
    print(f"partial-land={len(partial_land)}")
    print(f"empty-overlay={len(empty_overlay)}")

    for label, points in (
        ("visible-skipped", visible_skipped),
        ("partial-land", partial_land),
        ("empty-overlay", empty_overlay),
    ):
        if points:
            errors.append(f"{label} sample={points[:20]}")

    if args.preview is not None and rows and width > 0 and source.size == expected_size:
        render_preview(source, rows, args)

    if errors:
        for error in errors:
            print(f"ERROR: {error}")
        raise SystemExit(1)
    print("validation=ok")


if __name__ == "__main__":
    main()
