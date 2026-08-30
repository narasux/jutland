---
name: jutland-redraw-plane-livery
description: Redraw a Jutland plane top-view sprite from a low-resolution or off-style plan drawing while changing livery and keeping geometry. Use when the user supplies a yellow/trainer/low-clarity top view, asks to match existing military green or game paint, wants details preserved, or says 换涂装, 重新画, 低清俯视图, or 保留细节.
---

# Jutland 低清俯视图换涂装重绘

用户给一张**低清、黄底、训练机、线稿或非游戏风格**的俯视图，要求换成现有军绿/迷彩等游戏涂装，并保留细节时，用本 skill。

不要先走像素换色。那条路慢，而且成品更差。

## 和相邻 skill 的分工

| 情况 | 用哪个 |
|---|---|
| 低清/异风格俯视图 + 换涂装 + 保留细节 | **本 skill：按原图结构重绘** |
| 用户只要裁剪、旋转、去白底、缩放，不改涂装 | `jutland-add-plane` 的 `extract_plane_top_view.py` |
| 已经是游戏风格、调色板干净的舰船图改色 | `jutland-recolor-ship-image` 确定性像素处理 |
| 新飞机配置、武器链、航母编组 | `jutland-add-plane`；顶视图若属本表第一行，先完成本 skill |

“保留细节”指保留**几何和结构**（翼型、机身、座舱格、发动机位置、桨叶数、翼肋、浮筒、比例），不是保留原图像素或水印。

“参考已有军绿色涂装”指取样现有同国飞机的颜色、迷彩和游戏画风，不是把旧图轮廓套到新比例上。

## 核心教训（Sea Otter）

1. 黄底训练机线稿做 HSV/亮度映射，会留下 COPYRIGHT 水印、发灰的绿、以及和仓库飞机完全不像的画风。用户会觉得更垃圾。
2. 图像生成可以一次画出游戏风格顶视图。先判定路线，不要花很多轮分析调色板、抠水印、再换色。
3. `jutland-add-plane` 的“默认不重绘”只适用于用户要把**这张原图像素**当最终素材。换涂装 + 低清参考图 = 授权按原图结构重绘。
4. 生成图必须用用户俯视图当几何真值，用现有同国 PNG 当涂装真值。两张都作为 `reference_image_paths`。
5. 生成后立刻量包围盒宽高比。和用户原图相差超过约 8% 就重画，不要用大幅非等比拉伸凑数。

## 工作流程

在仓库根目录工作。先 `git status --short`，保留无关改动。

### 1. 确认输入

- 把用户图存到 `raw/<Name>-src.*`，需要时先用 `extract_plane_top_view.py` 抽出透明俯视图，只为看清结构和量比例，不当最终成品。
- 记录原图包围盒宽高比。机头朝上后，宽度/高度应接近翼展/机长。
- 从 `configs/planes.json5` 读 `length`，目标高度 = `length * 30`。宽度跟原图比例，不压成配置 `width`。
- 列出必须保留的结构：座舱格数/形状、发动机位置与桨叶数、翼肋、副翼、浮筒、尾翼形状、机头舱盖。
- 选 1–2 张同国现有飞机 PNG 作为涂装参考（如英国用 `Fulmar.png`、`Seafire.png`）。

### 2. 直接重绘

用图像生成，同时传入：

- 用户俯视图（几何）
- 1 张现有游戏飞机 PNG（涂装与线稿风格）

提示必须写明：

- 正上正交 2D 精灵，机头朝上，纯白底
- **描用户黄图/线稿的轮廓和部件，不要发明另一架飞机**
- 飞机包围盒宽高比写成具体数字（例如 `1.18`）；画布可更宽，但机翼不要铺满 4:3
- 桨叶数、座舱、发动机、翼肋、浮筒按原图
- 涂装按参考精灵：橄榄绿 + 深灰迷彩、青色座舱玻璃、黑色描边、该国常规圆标
- 不要水印、不要 COPYRIGHT、不要 RIGHT、不要文字

画布比例：原图宽高比约 1.0–1.2 用 `1:1`；约 1.15–1.3 用 `4:3`，并要求左右留白，避免机翼被拉得过宽。

### 3. 量比例，不对就重画

```bash
# 对生成图量非白包围盒
# aspect = bboxWidth / bboxHeight
# 和用户原图 aspect 比，偏差 > 8% 则重画，不要横向硬压超过 5%
```

常见失败：

- 1:1 画布把飞机画成正方形（翼展不够）
- 4:3 画布把机翼铺满画布（翼展过宽）
- 桨叶变成 3/4 叶（黄图是 2 叶就写死 TWO blades）
- 发明下单翼、额外武器或改座舱

最多试 3 版，留下最接近原图比例且结构没编造的一版。

### 4. 提取并写入资源

```bash
python .codex/skills/jutland-add-plane/scripts/extract_plane_top_view.py \
  --input "raw/<Name>-redraw.png" \
  --output "raw/<Name>-extract.png" \
  --crop <left,top,right,bottom> \
  --rotate 0 \
  --target-height <length*30>
```

- 用 `view_image` 看提取结果，并在黑底、品红底上检查白边和破洞。
- 宽度与原图比例差在 5% 内可以轻微缩放；超过就回到第 2 步。
- 用户要求替换现有飞机时，写入 `resources/images/planes/<type>/<name>.png`。
- 用 `ensure_rgba_png.go` 保证 RGBA。高度必须等于 `length * 30`。

### 5. 说明

写清：几何来自哪张用户图、涂装参考哪架现有飞机、生成了几版、最终宽高、是否做过小于 5% 的等比外微调。

## 不要做

- 不要对黄底/低清图做多轮调色板统计、连通域换色、水印像素修补。
- 不要用 `jutland-recolor-ship-image` 的舰船脚本处理飞机。
- 不要为了“风格统一”改翼型、机长翼展比、发动机布局。
- 不要把旧版游戏图（另一套比例）当几何参考。
- 不要把原图水印、RIGHT、COPYRIGHT 画进成品。
