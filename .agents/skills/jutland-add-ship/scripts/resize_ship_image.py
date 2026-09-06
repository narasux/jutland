#!/usr/bin/env python3
"""等比例缩放 Jutland 战舰透明 PNG，并保留缩放前备份。"""

from __future__ import annotations

import argparse
import filecmp
import shutil
from pathlib import Path

from PIL import Image, ImageChops, ImageFilter


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="使用预乘 alpha 等比例缩放战舰 PNG；不执行背景清理。",
    )
    parser.add_argument("input", type=Path, help="已确认的透明 PNG")
    parser.add_argument("output", type=Path, help="缩放后的正式或候选 PNG")
    parser.add_argument("--backup", type=Path, required=True, help="缩放前原图备份路径")
    target = parser.add_mutually_exclusive_group(required=True)
    target.add_argument("--target-width", type=int)
    target.add_argument("--target-height", type=int)
    target.add_argument("--target-long-axis", type=int)
    parser.add_argument(
        "--mode",
        choices=("plain", "line-art"),
        default="line-art",
        help="plain=仅 Lanczos；line-art=缩放后做温和线稿锐化（默认）",
    )
    parser.add_argument("--preview-dir", type=Path, help="输出黑底和洋红底预览")
    parser.add_argument("--allow-upscale", action="store_true", help="允许放大低分辨率输入")
    parser.add_argument("--overwrite-output", action="store_true", help="允许覆盖输出文件")
    return parser.parse_args()


def target_size(size: tuple[int, int], args: argparse.Namespace) -> tuple[int, int]:
    width, height = size
    if args.target_width is not None:
        target_width = args.target_width
        target_height = round(height * target_width / width)
    elif args.target_height is not None:
        target_height = args.target_height
        target_width = round(width * target_height / height)
    else:
        if width >= height:
            target_width = args.target_long_axis
            target_height = round(height * target_width / width)
        else:
            target_height = args.target_long_axis
            target_width = round(width * target_height / height)
    if target_width <= 0 or target_height <= 0:
        raise ValueError("目标尺寸必须为正整数")
    return target_width, target_height


def preserve_backup(source: Path, backup: Path) -> None:
    if backup.exists():
        if not filecmp.cmp(source, backup, shallow=False):
            raise FileExistsError(f"备份已存在且内容不同，拒绝覆盖: {backup}")
        return
    backup.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, backup)


def resize(image: Image.Image, size: tuple[int, int], mode: str) -> Image.Image:
    # 在预乘 alpha 空间缩放，避免透明像素中隐藏的白色 RGB 渗入边缘。
    premultiplied = image.convert("RGBa").resize(size, Image.Resampling.LANCZOS)
    if mode == "line-art":
        # 温和提高蓝图细线对比度；参数固定，避免不同任务产生不可比结果。
        premultiplied = premultiplied.filter(
            ImageFilter.UnsharpMask(radius=0.6, percent=70, threshold=2),
        )
        # UnsharpMask 可能使颜色通道略高于 alpha；重新钳制以维持合法预乘颜色。
        red, green, blue, alpha = premultiplied.split()
        premultiplied = Image.merge(
            "RGBa",
            (
                ImageChops.darker(red, alpha),
                ImageChops.darker(green, alpha),
                ImageChops.darker(blue, alpha),
                alpha,
            ),
        )
    return premultiplied.convert("RGBA")


def save_previews(image: Image.Image, output: Path, preview_dir: Path) -> None:
    preview_dir.mkdir(parents=True, exist_ok=True)
    for suffix, color in (("black", (0, 0, 0, 255)), ("magenta", (255, 0, 255, 255))):
        canvas = Image.new("RGBA", image.size, color)
        canvas.alpha_composite(image)
        canvas.convert("RGB").save(preview_dir / f"{output.stem}.{suffix}.png")


def main() -> None:
    args = parse_args()
    source = args.input.resolve()
    output = args.output.resolve()
    backup = args.backup.resolve()
    if not source.is_file():
        raise FileNotFoundError(source)
    if source.suffix.lower() != ".png" or output.suffix.lower() != ".png" or backup.suffix.lower() != ".png":
        raise ValueError("输入、输出和备份都必须是 PNG")
    if len({source, output, backup}) != 3:
        raise ValueError("输入、输出和备份路径必须互不相同")
    if output.exists() and not args.overwrite_output:
        raise FileExistsError(f"输出已存在；确认后使用 --overwrite-output: {output}")

    with Image.open(source) as opened:
        if opened.format != "PNG" or "A" not in opened.getbands():
            raise ValueError("输入必须是带 alpha 通道的 PNG")
        image = opened.convert("RGBA")
    if image.getchannel("A").getextrema()[0] == 255:
        raise ValueError("输入没有透明像素；请先使用 jutland-clean-ship-image")

    size = target_size(image.size, args)
    scale = size[0] / image.width
    if scale > 1 and not args.allow_upscale:
        raise ValueError("默认拒绝放大输入；确认必要后使用 --allow-upscale")

    preserve_backup(source, backup)
    result = resize(image, size, args.mode)
    output.parent.mkdir(parents=True, exist_ok=True)
    result.save(output, format="PNG", compress_level=9)
    if args.preview_dir:
        save_previews(result, output, args.preview_dir.resolve())

    source_pixels_per_output_pixel = 1 / scale
    print(f"input={source} size={image.width}x{image.height}")
    print(f"output={output} size={result.width}x{result.height} mode={args.mode}")
    print(f"backup={backup}")
    print(f"scale={scale:.6f}")
    print(
        "detail-risk: source features thinner than "
        f"about {source_pixels_per_output_pixel:.2f}px may collapse below one output pixel",
    )
    if scale < 0.5:
        print("warning: aggressive downscale (<0.5); compare plain and line-art previews before replacing resources")


if __name__ == "__main__":
    main()
