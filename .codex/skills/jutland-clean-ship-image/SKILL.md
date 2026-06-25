---
name: jutland-clean-ship-image
description: Remove white or near-white backgrounds from ship drawings and PNG assets for Jutland while preserving image dimensions, orientation, scale, and artwork. Use before jutland-add-ship when source images still have a white background, or when a user asks only for transparent-background cleanup and a reviewable preview.
---

# Jutland 清理战舰图片白底

## 范围

- 只把白色或近白色背景变为透明；保留原始画布尺寸、方向、比例和图中内容。
- 不裁剪、不拆分视图、不判断俯视/侧视、不旋转、不缩放、不更新配置，也不覆盖正式游戏资源。
- 不重绘、补画、风格化或删除飞机、标注等非背景内容。边界不确定时停止并请用户判断。
- 始终保留输入文件，输出独立的 `<stem>.cleaned.png` 候选文件。
- 完成预览后停止，等待用户人工确认；不要在同一轮继续执行 `jutland-add-ship`。

## 工作流程

1. 检查输入。
   - 在仓库根目录阅读 `AGENTS.md`，执行 `git status --short`，保留无关用户改动。
   - 用 `sips` 或等效工具记录尺寸和色彩模式，并用 `view_image` 检查白底、浅灰结构、白色标线和细线区域。

2. 清理外部连通背景。
   - 转为 RGBA，但不得改变像素尺寸。
   - 优先从画布边缘泛洪，只删除与边缘连通的低饱和近白色像素。
   - 从保守阈值开始，例如每个 RGB 通道 `>=245` 且通道差值较小；不要用单纯亮度阈值删除浅灰结构。

3. 处理封闭白底。
   - 统计残留近白色连通组件的像素数和包围盒，再决定处理范围。
   - 只删除能明确判断为背景的组件。栅栏、索具、吊臂等细线包围区域可在原始分辨率下使用约 `225~235` 的低饱和亮色阈值，但必须保留深灰、黑色和抗锯齿线条。
   - 每次放宽阈值或扩大区域前，备份当前候选文件为 `<stem>.before-aggressive-cleanup.png`。
   - 用户指出局部残留时，重新分析该区域的连通组件，不要逐像素猜测；发现误删则从最近备份恢复。

4. 输出确认材料。
   - 保存未缩放的透明 PNG 候选文件，不覆盖输入或 `resources/images/ships/...` 正式资源。
   - 生成透明底预览以及黑底或洋红底预览；用 `view_image` 检查白点、白边、断线和误删。
   - 报告原始/输出尺寸、删除的像素或组件数量，以及仍需人工判断的区域。
   - 明确请求用户确认，然后停止。

## 验证

- 输入文件未改变，输出为 RGBA PNG。
- 输出尺寸、方向和比例与输入完全一致。
- 外部白底已透明，细线和非背景白色内容仍保留。
- 高对比背景下没有明显白块；无法可靠区分的区域已留给用户确认。
