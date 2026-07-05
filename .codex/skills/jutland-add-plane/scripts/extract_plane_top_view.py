#!/usr/bin/env python3
"""Extract a transparent top-view plane PNG from a user-supplied source image.

This script intentionally performs only deterministic source-preserving edits:
crop, rotate, remove white page background, keep the largest connected component,
trim transparent borders, and optionally resize. It must not redraw, recolor, or
invent aircraft structure.
"""

from __future__ import annotations

import argparse
from pathlib import Path

import numpy as np
from PIL import Image


def parse_crop(value: str) -> tuple[int, int, int, int]:
    parts = [int(part.strip()) for part in value.split(",")]
    if len(parts) != 4:
        raise argparse.ArgumentTypeError("crop must be left,top,right,bottom")
    left, top, right, bottom = parts
    if right <= left or bottom <= top:
        raise argparse.ArgumentTypeError("crop right/bottom must be greater than left/top")
    return left, top, right, bottom


def largest_component(mask: np.ndarray) -> np.ndarray:
    height, width = mask.shape
    seen = np.zeros_like(mask, dtype=bool)
    best: list[tuple[int, int]] = []

    for y in range(height):
        for x0 in np.where(mask[y] & (~seen[y]))[0]:
            if seen[y, x0] or not mask[y, x0]:
                continue
            stack = [(y, int(x0))]
            seen[y, x0] = True
            comp: list[tuple[int, int]] = []
            while stack:
                cy, cx = stack.pop()
                comp.append((cy, cx))
                for ny in (cy - 1, cy, cy + 1):
                    for nx in (cx - 1, cx, cx + 1):
                        if ny == cy and nx == cx:
                            continue
                        if 0 <= ny < height and 0 <= nx < width and mask[ny, nx] and not seen[ny, nx]:
                            seen[ny, nx] = True
                            stack.append((ny, nx))
            if len(comp) > len(best):
                best = comp

    keep = np.zeros_like(mask, dtype=bool)
    for y, x in best:
        keep[y, x] = True
    return keep


def dilate(mask: np.ndarray, radius: int) -> np.ndarray:
    if radius <= 0:
        return mask
    height, width = mask.shape
    padded = np.pad(mask, radius, constant_values=False)
    result = mask.copy()
    size = radius * 2 + 1
    for dy in range(size):
        for dx in range(size):
            result |= padded[dy : dy + height, dx : dx + width]
    return result


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", required=True, type=Path, help="Source image path.")
    parser.add_argument("--output", required=True, type=Path, help="Output PNG path.")
    parser.add_argument("--crop", required=True, type=parse_crop, help="Crop box: left,top,right,bottom.")
    parser.add_argument(
        "--rotate",
        required=True,
        type=float,
        help="Rotation in Pillow degrees. Clockwise 45 degrees is -45.",
    )
    resize_group = parser.add_mutually_exclusive_group()
    resize_group.add_argument("--target-width", type=int, help="Optional output width after trimming.")
    resize_group.add_argument("--target-height", type=int, help="Optional output height after trimming.")
    parser.add_argument("--white-threshold", type=int, default=246, help="RGB threshold for full transparency.")
    parser.add_argument("--near-white-threshold", type=int, default=236, help="RGB threshold for soft edge alpha.")
    parser.add_argument("--near-white-alpha", type=int, default=105, help="Alpha for near-white antialias pixels.")
    parser.add_argument("--dilate", type=int, default=1, help="Pixel dilation after largest-component selection.")
    args = parser.parse_args()

    img = Image.open(args.input).convert("RGBA")
    crop = img.crop(args.crop)
    rotated = crop.rotate(
        args.rotate,
        resample=Image.Resampling.BICUBIC,
        expand=True,
        fillcolor=(255, 255, 255, 255),
    )

    arr = np.array(rotated)
    rgb = arr[..., :3].astype(np.int16)
    span = rgb.max(axis=2) - rgb.min(axis=2)

    white = (
        (rgb[..., 0] > args.white_threshold)
        & (rgb[..., 1] > args.white_threshold)
        & (rgb[..., 2] > args.white_threshold)
        & (span < 13)
    )
    near_white = (
        (rgb[..., 0] > args.near_white_threshold)
        & (rgb[..., 1] > args.near_white_threshold)
        & (rgb[..., 2] > args.near_white_threshold)
        & (span < 17)
        & (~white)
    )

    alpha = np.full(white.shape, 255, dtype=np.uint8)
    alpha[white] = 0
    alpha[near_white] = np.uint8(args.near_white_alpha)

    keep = largest_component(alpha > 30)
    keep = dilate(keep, args.dilate)
    arr[..., 3] = np.where(keep, alpha, 0).astype(np.uint8)

    out = Image.fromarray(arr, "RGBA")
    out_alpha = np.array(out.getchannel("A"))
    ys, xs = np.where(out_alpha > 0)
    if len(xs) == 0 or len(ys) == 0:
        raise SystemExit("no visible aircraft pixels remained after background removal")
    out = out.crop((xs.min(), ys.min(), xs.max() + 1, ys.max() + 1))

    if args.target_width:
        if args.target_width <= 0:
            raise SystemExit("--target-width must be positive")
        width, height = out.size
        out = out.resize((args.target_width, round(height * args.target_width / width)), Image.Resampling.LANCZOS)
    elif args.target_height:
        if args.target_height <= 0:
            raise SystemExit("--target-height must be positive")
        width, height = out.size
        out = out.resize((round(width * args.target_height / height), args.target_height), Image.Resampling.LANCZOS)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    out.save(args.output)
    print(f"{args.output} {out.size}")


if __name__ == "__main__":
    main()
