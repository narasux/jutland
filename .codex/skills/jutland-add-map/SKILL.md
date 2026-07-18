---
name: jutland-add-map
description: Add or revise Jutland map resources from raw raster tiles, including tile-order analysis, 4096-pixel runtime-width normalization, explicit rectangular-map height planning, RGBA resizing and stitching, transparent canvas extension, terrain .map generation, configs/maps.json5 registration, minimap integration, coastal square-artifact repair, and runtime-style validation. Use when adding a new map, rebuilding an existing map PNG/.map pair, expanding map bounds with ocean, fixing transparent coastline cells, or diagnosing wrong map dimensions, clipped content, or misaligned thumbnails. Do not use for fleet placement or scenario design; use jutland-design-map-mission after the map itself is complete.
---

# Jutland 添加地图

## 核心边界

- 在仓库根目录工作。先阅读 `AGENTS.md` 并运行 `git status --short`，保留用户已有改动。
- 默认只修改地图资源、`configs/maps.json5` 和必要的通用地图代码。不要修改 `configs/missions.json5`、舰队位置或关卡信息，除非用户明确要求。
- 原始素材保留在 `raw/map/`；运行时资源使用：
  - `resources/images/map/abbrs/<source>.png`
  - `resources/maps/<map-name>.map`
  - `configs/maps.json5`
- 地图完成后再使用 `jutland-design-map-mission` 设计关卡。

## 运行资源尺寸不变量

- Jutland 当前运行时地图 PNG 的标准宽度是 **4096 像素**。正方形地图使用 `4096×4096`；长方形地图必须在生成前明确高度，例如珍珠港纵横比为 `1:1.5`，使用 `4096×6144`。
- 不要把原始瓦片尺寸、拼接中间图尺寸当成运行时尺寸。即使原图拼接后是 `8192×10240`，也必须先归一化到计划的 `4096×<目标高度>`，不能直接放进 `resources/images/map/abbrs/`。
- 在动手前写明四个值：目标 PNG 宽高、目标网格宽高、`cell-pixels`、计划的画布扩展方向。当前已注册地图通常为 128 格宽，因此 `4096 / 128 = 32` 源像素/格；不要沿用旧示例中的 64，必须用实际 PNG 和 `.map` 重新计算。
- PNG 宽高必须分别等于 `.map` 宽高乘以同一个 `cell-pixels`，保证每个地图格对应正方形源图区域。宽高或网格任一项变化时同步更新另一项。
- 保持地理内容纵横比：先按统一比例缩放拼接图，再通过透明画布补足目标长方形范围。不要分别拉伸宽度和高度，也不要复制岸线填充外海。
- 像素分辨率与游戏逻辑比例是两件事。PNG 降采样而 `.map` 网格不变，只改变清晰度和文件大小，不会缩小岛屿相对舰船的逻辑尺寸；地标/舰船比例问题交给 `jutland-design-map-mission` 审计。

## 地形与透明度不变量

运行时绘制规则决定地形字符不能只凭视觉猜测：

- `.` / `O`：只绘制通用浅海/深海，不叠加地图源图；对应源图格必须全透明。
- `S`：可航行浅海；先铺浅海底图，再叠加源图。
- `C`：不可航行海岸；先铺浅海底图，再叠加源图。
- `L`：陆地；只绘制源图，不铺海底图；对应源图格必须完全不透明。

因此必须满足：

1. 任何含可见像素的格子都不能标成 `.` 或 `O`。
2. 任何含透明或半透明像素的格子都不能标成 `L`，否则会出现方形黑底或暗块。
3. 半透明格按通航意图分为 `S` 或 `C`；水侧和允许舰船进入的浅滩用 `S`，明确阻挡舰船的岸线用 `C`。
4. 不要用“可见像素占比超过 5%”之类阈值决定是否叠加；只要 alpha 最大值大于 0 就必须叠加。

## 工作流程

1. **检查素材。**
   - 记录每张图的尺寸、模式、alpha 范围和文件名。
   - 素材必须能转换为 RGBA。若海域本身是不透明蓝色，不能使用 alpha 自动生成地形；先取得透明海域版本或单独地形掩码。
   - 用 `scripts/inspect_tile_edges.py` 比较左右、上下边缘，结合地标确认拼接顺序；不要只相信文件编号。

2. **确定几何。**
   - 明确素材行列顺序、每块目标尺寸、地图格源像素尺寸和最终网格宽高。
   - 检查 `resources/images/map/abbrs/*.png`、目标 `.map` 和 `configs/maps.json5`，确认运行时目标宽度为 4096，而不是从最大原图猜测尺寸。
   - 最终 PNG 宽高必须分别整除网格宽高，且横纵 `cell-pixels` 必须相同。
   - 扩展海域时增加透明画布，而不是复制岸线。例如从 `128×128` 扩为 `128×192`，应在指定方向增加 64 格；按 32 像素/格计算，PNG 从 `4096×4096` 变为 `4096×6144`。

3. **生成候选资源。**
   - 脚本依赖 Pillow。若系统 `python3` 无法导入 `PIL`，先调用工作区依赖加载工具取得 bundled Python 路径，并用该解释器执行；不要为此修改项目依赖。
   - 按明确的行优先顺序运行：
     ```bash
     python3 .codex/skills/jutland-add-map/scripts/build_map_assets.py \
       --tiles raw/map/tl.png raw/map/tr.png raw/map/bl.png raw/map/br.png \
       --columns 2 \
       --tile-size 2048 \
       --cell-pixels 32 \
       --expected-width 4096 \
       --expected-height 4096 \
       --expected-grid-width 128 \
       --expected-grid-height 128 \
       --output-image /tmp/new_map.png \
       --output-map /tmp/new_map.map
     ```
   - 需要扩展底部海域时添加 `--append-bottom-cells <N>`。
   - `--tile-size`、`--cell-pixels`、目标高度和目标网格宽高必须显式填写；生成脚本默认要求运行时宽度为 4096，候选 PNG 或网格尺寸不符时在写文件前失败。
   - 默认将所有半透明格标为 `S`，确保浅海底图存在。需要阻挡通行的岸线，审图后人工改为 `C`，或使用 `--coast-alpha-threshold` 生成候选 `C` 格；不要未经审查直接接受阈值结果。
   - 先输出到临时路径并检查，确认后再写入运行时资源路径。

4. **审查地形。**
   - 生成整图预览并放大检查所有港湾、岛屿和外海岸，不只检查用户截图区域。
   - 重点寻找沿实际 `cell-pixels` 网格呈阶梯状的黑框、暗框、缺口和海面色差。
   - 检查 `S` 是否过度侵入内陆，`C/L` 是否切断应可通航的港池和航道。
   - 若计划停放舰船，保证舰体中心及合理舰体范围落在 `.`、`O` 或 `S`，而不是 `C/L`。

5. **接入游戏。**
   - 将确认后的 PNG 和 `.map` 放入运行时路径，并向 `configs/maps.json5` 添加地图名称、四种语言显示名和 `source`。
   - 若这是首张非正方形地图，检查 [references/map-runtime.md](references/map-runtime.md) 中列出的所有矩形地图消费者；不要只修主战场或一个缩略图入口。
   - 不要顺手添加或修改任务。

6. **验证并交付。**
   - 运行：
     ```bash
     python3 .codex/skills/jutland-add-map/scripts/validate_map_assets.py \
       --image resources/images/map/abbrs/<source>.png \
       --map resources/maps/<map-name>.map \
       --cell-pixels 32 \
       --expected-width 4096 \
       --expected-height <计划高度> \
       --expected-grid-width <计划列数> \
       --expected-grid-height <计划行数> \
       --preview /tmp/<map-name>-preview.png
     ```
   - 目标高度和目标网格宽高必须显式填写，禁止根据已经生成的错误图片反推。验证器默认要求宽度 4096，PNG 或网格尺寸不符时必须失败。
   - 必须得到正确的 `declared-target` 和 `declared-grid`，并满足：`visible-skipped=0`、`partial-land=0`、`empty-overlay=0`。
   - 运行 `git diff --check`、地图配置相关测试和 `go build ./...`。受无关工作区改动阻塞时，明确报告阻塞文件，不改动它们。
   - 完全退出旧游戏进程并重新启动；地图配置和场景块缓存不会被旧进程自动刷新。
   - 最终说明素材顺序、PNG/网格尺寸、扩展方向和格数、字符统计、验证结果、未修改的关卡范围及手动视觉验证方法。

## 经验与故障定位

遇到拼接、岸线方框、矩形地图裁切、缩略图比例或坐标错位时，读取 [references/map-runtime.md](references/map-runtime.md)。
