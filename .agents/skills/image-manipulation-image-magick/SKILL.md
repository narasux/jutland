---
name: image-manipulation-image-magick
description: Process and manipulate images with ImageMagick. Supports resize, format conversion, batch processing, and image metadata. Use when working with images, creating thumbnails, resizing wallpapers, or running batch image operations.
compatibility: Requires ImageMagick installed and available as `magick` on PATH.
---

# Image Manipulation with ImageMagick

```bash
command -v magick >/dev/null 2>&1 && echo "magick available" \
  || { echo "ImageMagick required: brew install imagemagick"; exit 1; }
```

ImageMagick 6.x ships no `magick` binary: use `convert` (and `identify`) instead.

## Dimensions and metadata

```bash
magick identify -format "%wx%h" path/to/image.jpg       # width x height
magick identify -verbose path/to/image.jpg               # format, color space, ...
for img in path/to/images/*; do
    magick identify -format "%f: %wx%h\n" "$img"
done
```

## Resize

```bash
magick input.jpg -resize 427x240 output.jpg
```

The aspect ratio is preserved by default; `^` fits the smaller side, `!` forces exact dimensions.

## Batch resize

```bash
mkdir -p path/to/output
for img in path/to/images/*; do
    filename=$(basename "$img")
    magick "$img" -resize 427x240 "path/to/output/thumb_$filename"
done
```

Large batches are memory-intensive; some operations need extra ImageMagick delegates.
