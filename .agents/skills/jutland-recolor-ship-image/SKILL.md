---
name: jutland-recolor-ship-image
description: Recolor Jutland ship PNGs (舰船图片、配色修改、换色、recolor) while preserving geometry, line art, size, transparency, labels and flags; covers hull paint, camouflage removal, deck and deck seam fixes, dense black fills, torpedo bulges, weapon recolor fixes, and validation.
---

# Jutland 战舰图片配色修改

## 范围

- 来源图和候选输出统一放在 `raw_resources/` 下（例如 `raw_resources/<stem>.recolored.png`），只有正式图片写入 `resources/`；候选默认独立输出，只有用户明确要求时才更新已有候选文件。
- 只修改用户指定的涂装或材质颜色，保留舰型、视图布局、尺寸、比例、线稿和透明通道；不裁剪、不旋转、不缩放、不重绘设备，也不更新正式游戏资源和配置。
- 默认保留红色水下舰体、黑色轮廓、旗帜、文字、比例尺、舷号、舷窗和来源署名。
- 完成候选图和验证后停止，等待人工确认；不要在同一轮继续执行 `jutland-add-ship`。
- 飞机低清俯视图换涂装走 `jutland-redraw-plane-livery`：按原图结构重绘，不要做多轮像素换色。

## 核心原则

1. 把参考图作为配色依据，不作为几何重绘依据。
2. 最终候选必须从原始清晰图重新生成；旧候选只可用于取色、比较或提供材质掩码，不能反复叠加换色。
3. 精确素材修改优先使用确定性像素处理；图像生成模型只能用于快速配色预览，不能直接作为最终游戏素材。
4. 不把“同一种 RGB”默认当成“同一种材质”：甲板、炮塔、防空炮和防鱼雷突出部可能共用原图颜色。
5. 默认保护纯黑线稿；只有确认是大面积迷彩填充时，才按组件填充率或像素邻域处理其内部。
6. 使用颜色、位置、连通组件形状、填充率和左右对称性共同判断材质。
7. 每轮只处理一种材质或一组明确区域，检查后再继续。

需要判断材质、选取参考色或处理误染时，阅读
[`references/recoloring-lessons.md`](references/recoloring-lessons.md)。

## 工作流程

### 1. 检查输入

- 用 `sips` 记录目标图和参考图的尺寸、格式与透明通道。
- 用 `view_image` 检查侧视图和俯视图，明确水线上舰体和上层建筑、木甲板、主炮/副炮/防空炮与炮座、左右防鱼雷突出部和外侧金属带、水下舰体及旗帜文字等必须保留的区域。
- 旧候选发虚时，比较原图与候选中的纯黑像素改动；线稿被改灰通常是主要原因。

### 2. 建立材质与颜色表

- 从参考图相应位置取样，不只看全图高频色。
- 用分析脚本查看区域高频颜色，或统计指定颜色的连通组件：

```bash
# 区域高频色
go run .agents/skills/jutland-recolor-ship-image/scripts/analyze_ship_palette.go \
  -input <reference.png> -box <minX,minY,maxX,maxY> -top 30
# 指定颜色的连通组件
go run .agents/skills/jutland-recolor-ship-image/scripts/analyze_ship_palette.go \
  -input <target.png> -box <minX,minY,maxX,maxY> -color <R,G,B> -components -min-area 4
```

- 记录每个材质的源色、目标色、限定区域、需要保留的相邻内容和预期改动范围。
- `fill` 接近 `1` 的宽高组件更像实心涂装块；极低填充率、细长或一像素高的组件更像线稿、栏杆或甲板板缝。

### 3. 先做大面积材质，再做设备纠色

顺序：水线上舰体与上层建筑 → 俯视木甲板 → 炮塔和大型金属设备 → 防鱼雷突出部 → 博福斯等小型防空炮与炮座 → 局部阴影、高光和误染修复。

银白舰体不要使用单一灰色。推荐把层次绑定到结构：

- 主舰体和大面积舱壁：`193` 或 `203`；
- 高光、上层塔体和浅色平台：`213`、`219` 或 `226`；
- 平台底部、设备侧面和内凹阴影：`174` 或 `181`；
- `145` 或 `164` 只用于明确的深凹部位，不能保留成跨越同一舱壁的迷彩色块。

同一块连续舱壁不应因为原迷彩的斜线或矩形边界突然改变灰阶：先统一材质大面，再从结构位置恢复少量阴影。

**统一俯视图炮塔顶部颜色**

- 同批主炮塔可能一部分是浅色顶、一部分是深灰顶；先统计深色顶的精确颜色和连通组件包围盒，再逐座框选处理。
- 常见配色：深灰顶 `125,125,125`，其斜面/装饰线 `105,105,105`；浅色顶 `228,228,228`，细节线 `193,193,193`。在每座深色顶的 `-include-box` 内用 `-mode pixels` 把 `125->228`、`105->193`，即可与浅色顶统一。
- 框选要完整覆盖炮塔顶面及其内部细节线，但不要延伸到旁边甲板、副炮或炮管，避免连带改色。
- 不要全局替换 `125`/`105`：同色还用于侧舷副炮、平台和设备阴影。改完放大对比同型炮塔确认一致。
- 侧舷副炮等小型炮座若统一为深灰顶，要先确认这是整批一致的设计，还是与主炮塔不一致的残留，再决定是否改；不要默认把深灰顶一律改浅。

只包含一种材质的区域可按精确 RGB 直接改像素；同一源色同时出现在甲板和金属设备上时必须按连通组件筛选：

```bash
go run .agents/skills/jutland-recolor-ship-image/scripts/recolor_ship_components.go \
  -input <input.png> -output <output.png> \
  -source-color <R,G,B> -target-color <R,G,B> \
  -mode components -selection intersect \
  -include-box <minX,minY,maxX,maxY> \
  -min-area <pixels> -min-width <pixels> -min-height <pixels>
```

- 单材质区域用 `-mode pixels`；`-include-box` 和 `-exclude-box` 均可重复。
- 多个源色需要分多次处理，每次写入由 `mktemp -d` 动态创建的临时目录；最终只保留一个候选 PNG。
- 防鱼雷突出部通常是两条长而对称的连通带，可用面积、宽度、纵向位置筛选。
- 防空炮通常是成对或成组重复的小组件，应逐组框选，不要把附近甲板板缝整体改灰。

### 4. 线稿、甲板板缝与对称结构

- 大面积纯黑迷彩填充与纯黑线稿连通时不要整组件替换：只修改同色邻居足够多（约 7 个）的内部像素，用中灰表达凹部，并保留一像素黑色边界。
- 木甲板缩小后出现大片黑影时，先判断是否为高密度一像素板缝：不要用纯黑（`0,0,0`）板缝，改用低对比度暖棕 `198,177,124`；把每 2 像素一条的板缝降为每 4 像素一条（删除其中一条、相邻一条改为柔化色），被删除的板缝恢复为相邻木色（`239,228,176`、`236,220,160`），只处理上下均为木色、左右连续的一像素水平线。
- 主甲板和二层甲板可使用接近的浅蜡黄色底色，通过较低对比度板缝区分层次。
- 炮管、设备轮廓和甲板外缘仍保留纯黑。
- 俯视图的防鱼雷突出部或金属带一侧正确、另一侧误染时，围绕舰体中轴做镜像材质修复：以正确侧的金属灰（`174,174,174`、`193,193,193`）为材质真值，只处理误染侧的像素，并验证目标带内不再残留木色。

### 5. 检查并修复误染

- 放大查看俯视图外缘、炮座、炮身和上层建筑交界，对照参考图相同材质：木甲板保留暖黄底色与较深板缝；炮、炮座和防鱼雷突出部使用浅灰至中灰金属层次；不能因为金属设备位于甲板上就把它染成木色。
- 发现误染时，从修改前候选重新分析相关颜色组件，不要凭矩形逐像素猜测。
- 每次纠色后重新运行颜色统计，确认目标区域内不再残留错误色组件。

### 6. 验证

- 比较修改前后图像：

```bash
go run .agents/skills/jutland-recolor-ship-image/scripts/compare_ship_images.go \
  -before <before.png> -after <after.png> -protected-color 0,0,0 \
  -allowed-box <minX,minY,maxX,maxY> -protected-box <flag-or-label-box> \
  -max-alpha-changes 0 -max-protected-changes 0
```

- `allowed-box` 存在时，脚本会在发现框外改动时返回失败；`protected-color` 保护原图中的精确颜色，可重复保护黑色线稿、水线或关键设备颜色；`protected-box` 保护旗帜、文字、飞机或署名区域，可重复。
- 确认尺寸、方向和比例完全一致；输出仍为 PNG 并保留透明像素；改动只位于预期区域；线稿、旗帜、文字、舷号和红色水下舰体未改变。
- 对称修复后，单独统计目标防雷带内是否仍残留木色；大面积迷彩清理后，检查同一舱壁是否仍存在无结构依据的矩形或斜向色块。
- 用 `view_image` 检查原图，并在默认透明、黑色和品红背景上检查透明边缘。
- 即使所有像素均不透明，也要保留 RGBA PNG 编码：

```bash
go run .agents/skills/jutland-recolor-ship-image/scripts/ensure_rgba_png.go \
  -input <candidate.png> -output <candidate.png>
sips -g hasAlpha <candidate.png>
```

- 如仍有白底或索具围住的封闭白块，调用 `jutland-clean-ship-image` 的清理脚本；只处理明确的背景组件。

### 7. 输出

- 报告候选文件完整路径、尺寸、透明通道、各材质改色像素数及任何仍需判断的区域，显示候选图并请求用户确认。

## 脚本

- `scripts/analyze_ship_palette.go`：统计区域高频颜色，并列出指定颜色的连通组件。
- `scripts/recolor_ship_components.go`：按精确 RGB、区域和组件尺寸进行确定性换色。
- `scripts/ensure_rgba_png.go`：强制保存为 RGBA PNG，避免全不透明图片被编码成 RGB。
- `scripts/compare_ship_images.go`：比较尺寸、透明度、边界和受保护颜色或区域。
