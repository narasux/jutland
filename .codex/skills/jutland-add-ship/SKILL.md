---
name: jutland-add-ship
description: 根据用户提供的战舰蓝图向 Jutland Go/Ebiten 游戏添加新战舰。用于需要在不重新生成美术的前提下拆分战舰蓝图为俯视图和侧视图 PNG、透明化背景、根据资料链接分析舰船尺寸与武装、更新 configs/ships.json5 和 configs/references.json5，并把新战舰加入 TestAll 任务和对应增援列表的场景。
---

# Jutland 添加战舰

## 核心规则

- 在 Jutland 仓库根目录工作。先阅读 `AGENTS.md`，执行 `git status --short`，保留所有无关的用户改动。
- 不重新生成、不重绘、不风格化、不替换用户提供的蓝图素材。只允许裁剪、拆分、清理透明背景和缩放。
- 如果视图边界、舰体轮廓、栏杆空隙、桅杆/索具、迷彩边缘、背景与舰体分离关系不确定，先停下来询问用户。
- 每次尝试更激进的透明化或边缘清理前，先在同目录备份当前 PNG，例如 `<name>.before-rail-cleanup.png` 或 `<name>.before-aggressive-cleanup.png`。
- 保持配置和资源引用链一致：同一个战舰 `name` 必须同时匹配 PNG 文件名和所有配置引用。
- 优先沿用现有战舰、武器、飞机、发射器配置模式。只有现有配置无法合理表达新战舰时，才新增武器族配置。
- 本仓库默认只做编译/构建验证；除非用户明确要求，不主动跑测试。

## 仓库目标文件

- 战舰配置：`configs/ships.json5`
- 图鉴引用：`configs/references.json5`
- 测试任务：`configs/missions.json5` 中 `name: "TestAll"` 的任务
- 战舰图片：
  - `resources/images/ships/top/<type>/<name>.png`
  - `resources/images/ships/side/<type>/<name>.png`
- 现有战舰图片加载器在 `pkg/resources/images/ship/ship.go` 中扫描固定舰种目录。战舰 `type` 必须匹配已存在的资源目录，例如 `aircraft_carrier`、`battleship`、`cruiser`、`destroyer`、`frigate`、`torpedo_boat`、`cargo`、`hospital`。

## 工作流程

1. 检查现有模式。
   - 阅读 `configs/ships.json5`、`configs/references.json5` 和 `configs/missions.json5` 的 `TestAll` 区段。
   - 按 `type`、海军阵营、时代、吨位和定位选择 1-3 艘相似舰作为平衡基准，用于推导费用、减伤、加速度和转向速度。

2. 确认用户输入。
   - 询问或推断内部名 `name`、展示名 `displayName`、战舰 `type`、`typeAbbr`、舰船资料链接、蓝图/美术来源链接和作者。
   - 如果用户只提供常用舰名，先提出 snake_case 格式的 `name` 建议，并在写文件前让用户确认。

3. 拆分并清理图片。
   - 从用户提供的蓝图中提取侧视图和俯视图。除非用户明确要求保留正视图，否则不处理正视图。
   - 先裁剪，再透明化，再缩放；缩放后还要再次检查并清理边缘白点，因为白底图缩放会产生新的近白色抗锯齿残留。
   - 将非战舰背景透明化。明显的栏杆、桅杆、吊臂、上层建筑间隙需要处理；边界不确定时交给用户确认或手工处理。
   - 将透明 PNG 保存到对应 `top` 和 `side` 舰种目录。

4. 根据资料调整图片尺寸。
   - 优先读取用户提供的资料链接。如果无法联网或页面不可访问，向用户索要长度、宽度/型宽、排水量、速度和武装信息。
   - 真实尺寸四舍五入取整后写入配置：`length` 和 `width`。
   - `configs/ships.json5` 的 `width` 使用资料中的实际舰体宽度、型宽或船体宽度，不要用飞行甲板外廓宽度替代。百科中的“宽度/型宽”通常不是航母飞行甲板宽度。
   - 俯视图纵向长度约为 `length*4`。普通舰船可以让图片宽度接近 `width*4`；航空母舰要优先保留蓝图俯视图的原始视觉比例，因为飞行甲板、舷侧平台和炮座会明显外伸，不能用舰体/型宽强行压缩图片画布。
   - 侧视图宽度约为 `length*4`。高度保持原侧视图素材比例，除非资料明确需要修正。
   - 定稿前用 `sips -g pixelWidth -g pixelHeight` 对比现有同类图片尺寸。

## 图片透明化和边缘清理

- 优先使用“从画布边缘泛洪”的方式删除背景：只让透明像素和近白色背景像素从图片外部向内连通扩散，再把这些连通像素设为透明。这样不会误删飞行甲板中间的白色跑道线、识别线和标识。
- 背景阈值从保守开始，例如 `r/g/b >= 245`；缩放后出现白边或白点时，再放宽到 `>= 210` 左右，并要求 RGB 差值较小（低饱和近白色），避免删除浅灰舰体。
- 对栏杆、索具、吊臂围起来的封闭白底，外部泛洪不会处理。需要单独做连通组件清理：
  - 先统计近白色组件的位置、大小和包围盒，确认残留集中在哪些区域。
  - 俯视图优先只处理左右舷外缘带，保留中央飞行甲板线。
  - 侧视图优先处理舰体上方的索具、栏杆和外轮廓区域，避免清理红色船底、船体高光和内部浅灰结构。
  - 从小组件、外缘组件开始；如果仍不干净，再备份后扩大区域或放宽阈值。
- 每次处理后用 `view_image` 预览，并记录清理掉的组件数或像素数。若发现误删结构，恢复最近的备份后收紧区域或阈值。
- 如果用户指出局部区域仍有白点，优先对该视图重新分析近白色连通组件，不要手工逐像素猜。

5. 更新 `configs/ships.json5`。
   - 将新舰放到同舰种/同阵营的相邻分组附近。
   - `totalHP` 优先使用满载排水量；资料只有标准排水量时，使用最可靠的吨位并在回复中说明。
   - `maxSpeed` 使用资料中的最高航速，单位为节。
   - `fundsCost`、`timeCost`、`horizontalDamageReduction`、`verticalDamageReduction`、`acceleration`、`rotateSpeed` 从相似现有战舰推导。
   - 根据资料和图片可见武装配置主炮、副炮、防空炮、鱼雷、火箭和舰载机。尽量复用已有 `guns.json5`、`torpedo_launchers.json5`、`rocket_launchers.json5`、`planes.json5` 中的名称。每个武器挂点根据图片和同类配置设置 `posPercent` 与射界。
   - 航空母舰只在项目中已有对应飞机配置时添加 `aircraft` 编组。

6. 更新 `configs/references.json5`。
   - 添加同名 `name` 条目。
   - `specs` 包含类型、吨位、速度、费用和减伤。
   - `armaments` 简洁概括已配置的武器和舰载机。
   - `description` 根据资料写一段简短历史介绍。
   - `author` 使用蓝图/美术作者；未知则写 `未知`。
   - `links` 同时包含蓝图/美术来源和舰船资料来源。

7. 更新 `configs/missions.json5` 的 `TestAll`。
   - 按战舰 `type` 将新舰加入对应 `providedShipNames` 列表。
   - 在 `initShips` 中新增一艘己方 `HA` 初始舰，延续现有同舰种网格布局，避免与附近舰船重叠。
   - 除非用户明确要求，不修改其他任务。

8. 完成前验证。
   - 确认俯视图和侧视图 PNG 都存在，尺寸符合配置比例。
   - 确认 `ships.json5`、`references.json5`、图片文件名和 `TestAll` 都使用同一个 `name`。
   - 确认引用到的武器、发射器、飞机和弹药配置都存在。
   - 运行 `GOCACHE=/tmp/go-build go build -o /dev/null ./pkg/...`。
   - 最终回复中明确说明未验证的图片清理区域或资料假设。
