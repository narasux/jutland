#!/usr/bin/env python3
"""Rank likely right and bottom neighbors for RGBA map tiles."""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageChops, ImageStat


Image.MAX_IMAGE_PIXELS = None


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Compare premultiplied RGBA tile edges and rank likely neighbors."
    )
    parser.add_argument("tiles", nargs="+", type=Path)
    parser.add_argument("--strip", type=int, default=8, help="edge strip width in source pixels")
    parser.add_argument("--sample-length", type=int, default=512)
    parser.add_argument("--top", type=int, default=3, help="number of candidates per direction")
    return parser.parse_args()


def edge(path: Path, side: str, strip: int, sample_length: int) -> Image.Image:
    with Image.open(path) as opened:
        image = opened.convert("RGBa")
    width, height = image.size
    strip = max(1, min(strip, width, height))
    if side == "left":
        return image.crop((0, 0, strip, height)).resize(
            (strip, sample_length), Image.Resampling.LANCZOS
        )
    if side == "right":
        return image.crop((width - strip, 0, width, height)).resize(
            (strip, sample_length), Image.Resampling.LANCZOS
        )
    if side == "top":
        return image.crop((0, 0, width, strip)).resize(
            (sample_length, strip), Image.Resampling.LANCZOS
        )
    if side == "bottom":
        return image.crop((0, height - strip, width, height)).resize(
            (sample_length, strip), Image.Resampling.LANCZOS
        )
    raise ValueError(f"unknown side: {side}")


def difference(first: Image.Image, second: Image.Image) -> float:
    means = ImageStat.Stat(ImageChops.difference(first, second)).mean
    return sum(means) / len(means)


def main() -> None:
    args = parse_args()
    for path in args.tiles:
        if not path.is_file():
            raise SystemExit(f"missing tile: {path}")

    edges = {
        path: {
            side: edge(path, side, args.strip, args.sample_length)
            for side in ("left", "right", "top", "bottom")
        }
        for path in args.tiles
    }

    for path in args.tiles:
        right = sorted(
            (
                difference(edges[path]["right"], edges[other]["left"]),
                other,
            )
            for other in args.tiles
            if other != path
        )
        bottom = sorted(
            (
                difference(edges[path]["bottom"], edges[other]["top"]),
                other,
            )
            for other in args.tiles
            if other != path
        )
        print(path)
        print("  right:")
        for score, candidate in right[: args.top]:
            print(f"    {score:8.3f}  {candidate}")
        print("  bottom:")
        for score, candidate in bottom[: args.top]:
            print(f"    {score:8.3f}  {candidate}")


if __name__ == "__main__":
    main()
