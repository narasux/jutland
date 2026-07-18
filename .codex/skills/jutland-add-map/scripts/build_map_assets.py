#!/usr/bin/env python3
"""Resize, stitch, extend, and classify transparent Jutland map tiles."""

from __future__ import annotations

import argparse
from collections import Counter
from pathlib import Path

from PIL import Image


Image.MAX_IMAGE_PIXELS = None


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--tiles",
        nargs="+",
        required=True,
        type=Path,
        help="input tiles in explicit row-major order",
    )
    parser.add_argument("--columns", required=True, type=int)
    parser.add_argument("--tile-size", type=int, required=True)
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
        help="planned runtime PNG height; declare it before generation",
    )
    parser.add_argument("--expected-grid-width", type=int, required=True)
    parser.add_argument("--expected-grid-height", type=int, required=True)
    parser.add_argument("--append-top-cells", type=int, default=0)
    parser.add_argument("--append-bottom-cells", type=int, default=0)
    parser.add_argument("--append-left-cells", type=int, default=0)
    parser.add_argument("--append-right-cells", type=int, default=0)
    parser.add_argument(
        "--partial-char",
        choices=("S", "C"),
        default="S",
        help="terrain for partially transparent cells when no threshold selects C",
    )
    parser.add_argument(
        "--coast-alpha-threshold",
        type=float,
        help="optional mean alpha ratio [0,1]; partial cells at or above it become C",
    )
    parser.add_argument("--output-image", required=True, type=Path)
    parser.add_argument("--output-map", required=True, type=Path)
    return parser.parse_args()


def resize_rgba(image: Image.Image, size: tuple[int, int]) -> Image.Image:
    # RGBa is premultiplied alpha and avoids dark RGB bleeding at transparent edges.
    return image.convert("RGBa").resize(size, Image.Resampling.LANCZOS).convert("RGBA")


def validate_args(args: argparse.Namespace) -> None:
    if args.columns <= 0 or len(args.tiles) % args.columns != 0:
        raise SystemExit("tile count must be divisible by a positive --columns value")
    if args.tile_size <= 0 or args.cell_pixels <= 0:
        raise SystemExit("--tile-size and --cell-pixels must be positive")
    if args.expected_width <= 0 or args.expected_height <= 0:
        raise SystemExit("--expected-width and --expected-height must be positive")
    if args.expected_grid_width <= 0 or args.expected_grid_height <= 0:
        raise SystemExit("--expected-grid-width and --expected-grid-height must be positive")
    if args.tile_size % args.cell_pixels != 0:
        raise SystemExit("--tile-size must be divisible by --cell-pixels")
    extensions = (
        args.append_top_cells,
        args.append_bottom_cells,
        args.append_left_cells,
        args.append_right_cells,
    )
    if any(value < 0 for value in extensions):
        raise SystemExit("canvas extension cell counts cannot be negative")
    if args.coast_alpha_threshold is not None and not (
        0 <= args.coast_alpha_threshold <= 1
    ):
        raise SystemExit("--coast-alpha-threshold must be within [0,1]")
    for tile in args.tiles:
        if not tile.is_file():
            raise SystemExit(f"missing tile: {tile}")


def classify(image: Image.Image, cell_pixels: int, args: argparse.Namespace) -> list[str]:
    alpha = image.getchannel("A")
    width, height = image.size
    rows: list[str] = []
    for y in range(height // cell_pixels):
        chars: list[str] = []
        for x in range(width // cell_pixels):
            box = (
                x * cell_pixels,
                y * cell_pixels,
                (x + 1) * cell_pixels,
                (y + 1) * cell_pixels,
            )
            cell = alpha.crop(box)
            min_alpha, max_alpha = cell.getextrema()
            if max_alpha == 0:
                chars.append(".")
            elif min_alpha == 255:
                chars.append("L")
            elif args.coast_alpha_threshold is None:
                chars.append(args.partial_char)
            else:
                histogram = cell.histogram()
                mean_alpha = sum(value * count for value, count in enumerate(histogram)) / (
                    cell_pixels * cell_pixels * 255
                )
                chars.append("C" if mean_alpha >= args.coast_alpha_threshold else args.partial_char)
        rows.append("".join(chars))
    return rows


def main() -> None:
    args = parse_args()
    validate_args(args)

    tile_rows = len(args.tiles) // args.columns
    stitched = Image.new(
        "RGBA",
        (args.columns * args.tile_size, tile_rows * args.tile_size),
        (0, 0, 0, 0),
    )
    input_details: list[str] = []
    for index, path in enumerate(args.tiles):
        with Image.open(path) as opened:
            input_details.append(f"{path}={opened.size[0]}x{opened.size[1]} {opened.mode}")
            tile = resize_rgba(opened, (args.tile_size, args.tile_size))
        x = index % args.columns
        y = index // args.columns
        stitched.paste(tile, (x * args.tile_size, y * args.tile_size))

    left = args.append_left_cells * args.cell_pixels
    top = args.append_top_cells * args.cell_pixels
    final_width = stitched.width + left + args.append_right_cells * args.cell_pixels
    final_height = stitched.height + top + args.append_bottom_cells * args.cell_pixels
    image = Image.new("RGBA", (final_width, final_height), (0, 0, 0, 0))
    image.paste(stitched, (left, top))

    declared_size = (args.expected_width, args.expected_height)
    if image.size != declared_size:
        raise SystemExit(f"final image size={image.size}, declared target={declared_size}")
    if image.width % args.cell_pixels != 0 or image.height % args.cell_pixels != 0:
        raise SystemExit("final image dimensions are not divisible by --cell-pixels")
    grid_size = (image.width // args.cell_pixels, image.height // args.cell_pixels)
    declared_grid_size = (args.expected_grid_width, args.expected_grid_height)
    if grid_size != declared_grid_size:
        raise SystemExit(f"final grid size={grid_size}, declared grid={declared_grid_size}")

    rows = classify(image, args.cell_pixels, args)
    args.output_image.parent.mkdir(parents=True, exist_ok=True)
    args.output_map.parent.mkdir(parents=True, exist_ok=True)
    image.save(args.output_image)
    args.output_map.write_text("\n".join(rows) + "\n", encoding="utf-8")

    print("inputs:")
    for detail in input_details:
        print(f"  {detail}")
    print(f"image={image.width}x{image.height} RGBA")
    print(f"declared-target={args.expected_width}x{args.expected_height}")
    print(f"declared-grid={args.expected_grid_width}x{args.expected_grid_height}")
    print(f"grid={len(rows[0])}x{len(rows)} cell-pixels={args.cell_pixels}")
    print(f"terrain={dict(sorted(Counter(''.join(rows)).items()))}")
    print(f"output-image={args.output_image}")
    print(f"output-map={args.output_map}")


if __name__ == "__main__":
    main()
