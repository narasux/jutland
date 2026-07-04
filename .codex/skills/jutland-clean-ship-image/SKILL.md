---
name: jutland-clean-ship-image
description: Remove white or near-white backgrounds from ship drawings and PNG assets for Jutland while preserving image dimensions, orientation, scale, and artwork. Also supports user-requested bounded recoloring such as replacing ship camouflage with a uniform navy blue. Use before jutland-add-ship when source images still have a white background, or when a user asks only for transparent-background cleanup and a reviewable preview.
---

# Jutland 清理战舰图片白底

## 范围

- 只把白色或近白色背景变为透明；保留原始画布尺寸、方向、比例和图中内容。
- 不裁剪、不拆分视图、不判断俯视/侧视、不旋转、不缩放、不更新配置，也不覆盖正式游戏资源。
- 不重绘、补画、风格化或删除飞机、标注等非背景内容。边界不确定时停止并请用户判断。
- 只有用户明确要求时，才允许对限定区域做颜色统一，例如把侧视图舰体迷彩替换为统一海军蓝；必须保留线稿、红色水下船体、旗帜、飞机和文字。
- 始终保留输入文件，输出独立的 `<stem>.cleaned.png` 候选文件；除用户明确要求外，不在仓库中保留预览、备份或其他中间版本。
- 完成预览后停止，等待用户人工确认；不要在同一轮继续执行 `jutland-add-ship`。

## 工作流程

1. 检查输入。
   - 在仓库根目录阅读 `AGENTS.md`，执行 `git status --short`，保留无关用户改动。
   - 用 `sips` 或等效工具记录尺寸和色彩模式，并用 `view_image` 检查白底、浅灰结构、白色标线和细线区域。

2. 清理外部连通背景。
   - 转为 RGBA，但不得改变像素尺寸。
   - 优先从画布边缘泛洪，只删除与边缘连通的低饱和近白色像素。
   - 从保守阈值开始，例如每个 RGB 通道 `>=245` 且通道差值较小；不要用单纯亮度阈值删除浅灰结构。
   - 优先复用脚本：
     `go run .codex/skills/jutland-clean-ship-image/scripts/clean_ship_white_background.go -input <image.png> -output <stem>.cleaned.png`
   - 脚本默认只做边缘连通背景清理，并输出残留近白连通组件统计；默认不生成 checker/dark 预览。

3. 处理封闭白底。
   - 统计残留近白色连通组件的像素数和包围盒，再决定处理范围。
   - 可先只分析不写文件：
     `go run .codex/skills/jutland-clean-ship-image/scripts/clean_ship_white_background.go -input <image.png> -analyze-only`
   - 只删除能明确判断为背景的组件。栅栏、索具、吊臂等细线包围区域可在原始分辨率下使用约 `225~235` 的低饱和亮色阈值，但必须保留深灰、黑色和抗锯齿线条。
   - 每次放宽阈值或扩大区域前，如需人工对比，可把候选文件临时复制到 `/tmp` 或显式使用 `-backup-before-aggressive`；完成前必须删除中间版本，仓库中只保留最终 `<stem>.cleaned.png`。
   - 如需删除明确的大块封闭背景，使用脚本的显式面积阈值，例如：
     `go run .codex/skills/jutland-clean-ship-image/scripts/clean_ship_white_background.go -input <image.png> -output <stem>.cleaned.png -remove-enclosed-min-area <pixels>`
     脚本默认直接覆盖输出文件；只有显式传入 `-backup-before-aggressive` 时才会生成 `<stem>.before-aggressive-cleanup.png` 备份。
   - 如需清理索具、吊臂等局部围住的小块背景，使用一个或多个 `-remove-box minX,minY,maxX,maxY` 限定人工确认区域，避免误删甲板标线、文字和飞机细节；脚本仍按连通组件删除，不按矩形逐像素清空。
   - 如果栅栏/索具内仍有纯白斑点，可更激进地只清纯白小组件：提高 `-component-min` 到 `250` 左右，降低 `-remove-enclosed-min-area` 到 `1`，并必须配合精确 `-remove-box`。示例：
     `go run .codex/skills/jutland-clean-ship-image/scripts/clean_ship_white_background.go -input <image.png> -output <stem>.cleaned.png -edge-min 235 -component-min 250 -remove-enclosed-min-area 1 -remove-box <minX,minY,maxX,maxY>`
   - 使用纯白小组件清理时，框选区域要避开旗帜、比例尺、文字、飞机、高光和甲板白色标线；若这些内容位于同一区域，先缩小或拆分 `-remove-box`，不要全图执行。
   - 用户指出局部残留时，重新分析该区域的连通组件，不要逐像素猜测；发现误删则从最近备份恢复。

4. 可选：统一迷彩颜色。
   - 仅在用户明确要求时执行。不要全图换色，必须用一个或多个 `-recolor-box` 限定舰体或上层建筑区域。
   - 默认海军蓝为 Princeton 任务中使用的取样色 `56,68,93`；如用户给定新色样，可先取样再传入 `-target-color R,G,B`。
   - 使用脚本：
     `go run .codex/skills/jutland-clean-ship-image/scripts/uniform_ship_color.go -input <image.png> -output <stem>.cleaned.png -recolor-box <minX,minY,maxX,maxY>`
   - 用 `-skip-box` 避开飞机、旗帜、文字、比例尺、作者署名和其他不应调色的局部。默认保留近黑线稿和饱和红色区域，避免破坏船底红色、防空炮线稿和旗帜。
   - Princeton 类似合图的侧视图示例：
     `go run .codex/skills/jutland-clean-ship-image/scripts/uniform_ship_color.go -input princeton.png -output princeton.png -recolor-box 15,168,1280,455 -skip-box 745,185,770,205 -skip-box 925,278,1165,318`
   - 如果仍有很浅的灰色迷彩图块未统一，可适当提高 `-max-luma` 或增大/细分 `-recolor-box`；如果误伤文字、飞机或标线，优先增加 `-skip-box`，不要放宽到全图处理。

5. 输出确认材料。
   - 保存未缩放的透明 PNG 候选文件，不覆盖输入或 `resources/images/ships/...` 正式资源；仓库中只保留 `<stem>.cleaned.png`。
   - 用 `view_image` 检查透明 PNG；如需高对比背景预览，输出到 `/tmp`，完成前删除，不保留在仓库。
   - 报告原始/输出尺寸、删除的像素或组件数量、调色像素数量，以及仍需人工判断的区域。
   - 明确请求用户确认，然后停止。

## 验证

- 输入文件未改变，输出为 RGBA PNG。
- 输出尺寸、方向和比例与输入完全一致。
- 外部白底已透明，细线和非背景白色内容仍保留。
- 高对比背景下没有明显白块；无法可靠区分的区域已留给用户确认。
- 如执行调色，调色只发生在用户指定区域内，飞机、旗帜、文字、比例尺、红色水下船体和黑色线稿未被误改。
