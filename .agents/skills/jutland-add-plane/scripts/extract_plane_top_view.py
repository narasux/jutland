#!/usr/bin/env python3
"""Extract a transparent top-view plane PNG from a user-supplied source image.

This script intentionally performs only deterministic source-preserving edits:
crop, rotate, remove white page background, keep the largest connected component,
trim transparent borders, and optionally resize. It must not redraw, recolor, or
invent aircraft structure.
"""

from __future__ import annotations

import argparse
import math
from collections import deque
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


def flood_background(rgb: np.ndarray) -> np.ndarray:
    """Identify page background by flood fill from the image borders.

    White/silver-painted aircraft have skin pixels as bright as the page
    background, so a global white threshold would punch out the fuselage and
    wing interiors. Only white pixels connected to the page border are
    background; white enclosed by the line art is kept opaque.
    """
    span = rgb.max(axis=2) - rgb.min(axis=2)
    bg_like = (rgb.min(axis=2) > 245) & (span < 15)
    height, width = bg_like.shape
    background = np.zeros_like(bg_like)
    queue: deque[tuple[int, int]] = deque()
    for x in range(width):
        for y in (0, height - 1):
            if bg_like[y, x] and not background[y, x]:
                background[y, x] = True
                queue.append((y, x))
    for y in range(height):
        for x in (0, width - 1):
            if bg_like[y, x] and not background[y, x]:
                background[y, x] = True
                queue.append((y, x))
    while queue:
        y, x = queue.popleft()
        for ny, nx in ((y - 1, x), (y + 1, x), (y, x - 1), (y, x + 1)):
            if 0 <= ny < height and 0 <= nx < width and bg_like[ny, nx] and not background[ny, nx]:
                background[ny, nx] = True
                queue.append((ny, nx))
    return background


def remove_background(
    rgb: np.ndarray,
    background_mode: str,
    white_threshold: int,
    near_white_threshold: int,
    near_white_alpha: int,
) -> np.ndarray:
    """Build an alpha mask from page background using the selected mode."""
    span = rgb.max(axis=2) - rgb.min(axis=2)

    if background_mode == "flood":
        background = flood_background(rgb)
        height, width = background.shape
        pad = np.pad(background, 1, constant_values=False)
        adjacent_bg = (
            pad[0:height, 1 : width + 1]
            | pad[2 : height + 2, 1 : width + 1]
            | pad[1 : height + 1, 0:width]
            | pad[1 : height + 1, 2 : width + 2]
        )
        near_white = (
            (rgb[..., 0] > near_white_threshold)
            & (rgb[..., 1] > near_white_threshold)
            & (rgb[..., 2] > near_white_threshold)
            & (span < 17)
            & (~background)
            & adjacent_bg
        )
        white = background
    else:
        white = (
            (rgb[..., 0] > white_threshold)
            & (rgb[..., 1] > white_threshold)
            & (rgb[..., 2] > white_threshold)
            & (span < 13)
        )
        near_white = (
            (rgb[..., 0] > near_white_threshold)
            & (rgb[..., 1] > near_white_threshold)
            & (rgb[..., 2] > near_white_threshold)
            & (span < 17)
            & (~white)
        )

    alpha = np.full(white.shape, 255, dtype=np.uint8)
    alpha[white] = 0
    alpha[near_white] = np.uint8(near_white_alpha)
    return alpha


def crop_edge_sides(mask: np.ndarray) -> list[str]:
    """Return crop sides touched by a mask before rotation."""
    sides = []
    if mask[:, 0].any():
        sides.append("left")
    if mask[:, -1].any():
        sides.append("right")
    if mask[0, :].any():
        sides.append("top")
    if mask[-1, :].any():
        sides.append("bottom")
    return sides


def row_runs(row_mask: np.ndarray) -> list[tuple[int, int]]:
    """Contiguous opaque runs of one row."""
    idx = np.nonzero(row_mask)[0]
    if len(idx) == 0:
        return []
    runs, start, prev = [], int(idx[0]), int(idx[0])
    for i in idx[1:]:
        if i > prev + 1:
            runs.append((start, prev))
            start = int(i)
        prev = int(i)
    runs.append((start, prev))
    return runs


def fuselage_axis_tilt(alpha: np.ndarray) -> dict | None:
    """Measure the fuselage-axis tilt of a plan-view sprite, in degrees.

    Positive means the nose leans left. Only narrow runs close to the sprite
    centreline are followed, so wings, tailplanes, propeller blades and outboard
    engine nacelles cannot bias the fit (a plain centroid of every row is pulled
    off by engine nacelles on multi-engine aircraft). Returns None when the
    fuselage cannot be traced.
    """
    mask = alpha > 128
    height, width = mask.shape
    ys, xs = np.nonzero(mask)
    if len(xs) < 200:
        return None
    top, bot = int(ys.min()), int(ys.max())
    cx = float((xs.min() + xs.max()) / 2)
    max_width = max(8, int(width * 0.10))
    centre_step = max(4.0, width * 0.045)

    def collect(predict, limit: float) -> list[tuple[int, float]]:
        points = []
        for y in range(top, bot + 1):
            for start, end in row_runs(mask[y]):
                if end - start + 1 > max_width:
                    continue
                weights = alpha[y, start : end + 1]
                if weights.sum() <= 0:
                    continue
                centre = float((weights * np.arange(start, end + 1)).sum() / weights.sum())
                if abs(centre - cx) > limit:
                    continue
                if predict is not None and abs(centre - predict(y)) > centre_step:
                    continue
                points.append((y, centre))
        return points

    def fit(points: list[tuple[int, float]]) -> tuple[float, float, float, int]:
        ys_arr = np.array([p[0] for p in points], dtype=float)
        xs_arr = np.array([p[1] for p in points], dtype=float)
        keep = np.ones(len(ys_arr), dtype=bool)
        for _ in range(4):
            slope, intercept = np.polyfit(ys_arr[keep], xs_arr[keep], 1)
            residual = np.abs(xs_arr - (slope * ys_arr + intercept))
            next_keep = residual < max(1.5, 2.5 * residual[keep].std())
            if next_keep.sum() < 15 or (next_keep == keep).all():
                break
            keep = next_keep
        slope, intercept = np.polyfit(ys_arr[keep], xs_arr[keep], 1)
        residual = np.abs(xs_arr - (slope * ys_arr + intercept))[keep]
        return slope, intercept, float(residual.std()), int(keep.sum())

    points = collect(None, width * 0.18)
    if len(points) < 15:
        return None
    slope, intercept, _, _ = fit(points)
    for _ in range(2):
        points = collect(lambda y, s=slope, i=intercept: s * y + i, width * 0.25)
        if len(points) < 15:
            return None
        slope, intercept, residual_std, rows = fit(points)
    angle = math.degrees(math.atan(slope))
    if math.isnan(angle):
        return None
    return {"angle": angle, "residual": residual_std, "rows": rows}


def wing_row_fraction(alpha: np.ndarray) -> float | None:
    """Where the widest row sits along the sprite: 0 = nose end, 1 = tail end.

    Plan-view fighters and bombers carry their wing forward of mid-span, so a
    value near 1 hints that the sprite is nose-down (or that the planform is
    tailless and the check does not apply).
    """
    mask = alpha > 128
    ys, _ = np.nonzero(mask)
    if len(ys) == 0:
        return None
    top, bot = int(ys.min()), int(ys.max())
    if bot <= top:
        return None
    widest_y, widest = top, -1
    for y in range(top, bot + 1):
        cols = np.nonzero(mask[y])[0]
        if len(cols) == 0:
            continue
        width = int(cols.max() - cols.min()) + 1
        if width > widest:
            widest_y, widest = y, width
    return (widest_y - top) / (bot - top)


def rotate_transparent(img: Image.Image, angle: float) -> Image.Image:
    """Rotate an RGBA sprite in premultiplied-alpha space and trim borders.

    Plain RGBA rotation bleeds the black RGB of transparent pixels into the
    anti-aliased outline, which shows up as a dark fringe around the aircraft.
    """
    arr = np.array(img).astype(np.float64)
    alpha = arr[:, :, 3]
    premultiplied = arr[:, :, :3] * (alpha[:, :, None] / 255.0)

    def rotate_channel(channel: np.ndarray) -> np.ndarray:
        image = Image.fromarray(channel.astype(np.float32), "F")
        return np.array(
            image.rotate(angle, resample=Image.Resampling.BICUBIC, expand=True, fillcolor=0.0)
        ).astype(np.float64)

    red, green, blue, out_alpha = (
        rotate_channel(premultiplied[:, :, 0]),
        rotate_channel(premultiplied[:, :, 1]),
        rotate_channel(premultiplied[:, :, 2]),
        rotate_channel(alpha),
    )
    np.clip(out_alpha, 0, 255, out=out_alpha)
    unpremultiply = np.where(out_alpha > 0, 255.0 / np.maximum(out_alpha, 1e-6), 0.0)
    rgb = np.clip(np.dstack([red, green, blue]) * unpremultiply[:, :, None], 0, 255)
    return trim_transparent(Image.fromarray(np.dstack([rgb, out_alpha]).astype(np.uint8), "RGBA"))


def trim_transparent(img: Image.Image) -> Image.Image:
    """Crop fully transparent borders."""
    alpha = np.array(img.getchannel("A"))
    ys, xs = np.nonzero(alpha > 0)
    if len(xs) == 0 or len(ys) == 0:
        return img
    return img.crop((int(xs.min()), int(ys.min()), int(xs.max()) + 1, int(ys.max()) + 1))


def straighten_sprite(img: Image.Image, max_angle: float) -> tuple[Image.Image, dict | None, float | None, dict | None]:
    """Rotate the sprite so its fuselage axis is vertical.

    Returns (image, tilt_before, applied_angle, tilt_after). A tilt larger than
    max_angle is left untouched: that normally means --rotate points the nose the
    wrong way, which must stay a visible, explicit decision instead of a silent
    90/180 degree correction.
    """
    alpha = np.array(img.getchannel("A")).astype(np.float64)
    before = fuselage_axis_tilt(alpha)
    if before is None or not 0.05 < abs(before["angle"]) <= max_angle:
        return img, before, None, before
    applied = -before["angle"]
    rotated = rotate_transparent(img, applied)
    after = fuselage_axis_tilt(np.array(rotated.getchannel("A")).astype(np.float64))
    return rotated, before, applied, after


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
    parser.add_argument(
        "--background",
        choices=["white", "flood"],
        default="white",
        help=(
            "Background removal mode. 'white' matches bright pixels globally (default). "
            "'flood' flood-fills from the image borders, keeping white aircraft skin "
            "enclosed by the line art; use for white/silver-painted aircraft."
        ),
    )
    parser.add_argument("--dilate", type=int, default=1, help="Pixel dilation after largest-component selection.")
    parser.add_argument(
        "--no-straighten",
        dest="straighten",
        action="store_false",
        help="Skip the automatic fuselage-axis straightening step.",
    )
    parser.add_argument(
        "--max-straighten-angle",
        type=float,
        default=5.0,
        help="Largest tilt (degrees) the script corrects on its own; bigger tilts are reported, not rotated.",
    )
    parser.add_argument(
        "--tilt-tolerance",
        type=float,
        default=0.15,
        help="Report a warning when the sprite still tilts more than this after straightening.",
    )
    parser.add_argument(
        "--allow-crop-edge",
        action="store_true",
        help="Allow the aircraft component to touch a crop edge; use only when the source itself is clipped.",
    )
    args = parser.parse_args()

    img = Image.open(args.input).convert("RGBA")
    crop = img.crop(args.crop)
    crop_arr = np.array(crop)
    crop_alpha = remove_background(
        crop_arr[..., :3].astype(np.int16),
        args.background,
        args.white_threshold,
        args.near_white_threshold,
        args.near_white_alpha,
    )
    crop_keep = largest_component(crop_alpha > 30)
    touched_sides = crop_edge_sides(crop_keep)
    if touched_sides and not args.allow_crop_edge:
        sides = ", ".join(touched_sides)
        raise SystemExit(
            f"crop clips the largest aircraft component at: {sides}. "
            "Expand --crop to leave a white margin around the whole aircraft; "
            "use --allow-crop-edge only if the source image itself clips the aircraft."
        )

    rotated = crop.rotate(
        args.rotate,
        resample=Image.Resampling.BICUBIC,
        expand=True,
        fillcolor=(255, 255, 255, 255),
    )

    arr = np.array(rotated)
    alpha = remove_background(
        arr[..., :3].astype(np.int16),
        args.background,
        args.white_threshold,
        args.near_white_threshold,
        args.near_white_alpha,
    )

    keep = largest_component(alpha > 30)
    keep = dilate(keep, args.dilate)
    arr[..., 3] = np.where(keep, alpha, 0).astype(np.uint8)

    out = Image.fromarray(arr, "RGBA")
    out_alpha = np.array(out.getchannel("A"))
    ys, xs = np.where(out_alpha > 0)
    if len(xs) == 0 or len(ys) == 0:
        raise SystemExit("no visible aircraft pixels remained after background removal")
    out = out.crop((xs.min(), ys.min(), xs.max() + 1, ys.max() + 1))

    # 自动摆正机身轴线：量出倾角后反向旋转，保证机头正对画布上方
    tilt_before, applied, tilt_after = None, None, None
    if args.straighten:
        out, tilt_before, applied, tilt_after = straighten_sprite(out, args.max_straighten_angle)
    else:
        tilt_before = fuselage_axis_tilt(np.array(out.getchannel("A")).astype(np.float64))
        tilt_after = tilt_before
    if args.straighten and tilt_before is not None and applied is None and abs(tilt_before["angle"]) > args.max_straighten_angle:
        print(
            f"warning: 机身轴倾角 {tilt_before['angle']:+.2f}° 超过 --max-straighten-angle "
            f"{args.max_straighten_angle}°，未自动摆正；请检查 --rotate 是否把机头转反了"
        )

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

    final_alpha = np.array(out.getchannel("A")).astype(np.float64)
    if tilt_after is None:
        tilt_after = fuselage_axis_tilt(final_alpha)
    if tilt_before is None:
        print("机身轴：无法量测（连通域过小），请人工确认机头朝向与倾斜")
    elif applied is not None:
        after_text = f"{tilt_after['angle']:+.3f}°" if tilt_after else "无法复测"
        print(
            f"机身轴倾角 {tilt_before['angle']:+.3f}° -> 旋转 {applied:+.3f}° -> {after_text}"
            f"（resid {tilt_before['residual']:.2f}px）"
        )
    elif abs(tilt_before["angle"]) <= 0.05:
        print(f"机身轴倾角 {tilt_before['angle']:+.3f}°（resid {tilt_before['residual']:.2f}px），已竖直，无需摆正")
    elif abs(tilt_before["angle"]) <= args.max_straighten_angle:
        print(f"机身轴倾角 {tilt_before['angle']:+.3f}°（resid {tilt_before['residual']:.2f}px），未摆正（--no-straighten）")
    else:
        print(
            f"机身轴倾角 {tilt_before['angle']:+.3f}°（resid {tilt_before['residual']:.2f}px），"
            "超过自动摆正上限，未处理"
        )
    if applied is not None and tilt_after is not None and abs(tilt_after["angle"]) > args.tilt_tolerance:
        print(
            f"warning: 摆正后机身轴仍有 {tilt_after['angle']:+.3f}°，超过容差 {args.tilt_tolerance}°；"
            "可能是画稿本身机翼与机身不一致，需重新裁剪原图而不是继续旋转"
        )

    wing_fraction = wing_row_fraction(final_alpha)
    if wing_fraction is not None and wing_fraction > 0.55:
        print(
            f"warning: 最宽一行位于画布 {wing_fraction:.2f} 处（0=机头端），机头可能朝下；"
            "飞翼/无尾布局可忽略，其余必须用 view_image 确认机头朝上"
        )


if __name__ == "__main__":
    main()
