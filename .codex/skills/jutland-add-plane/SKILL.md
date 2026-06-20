---
name: jutland-add-plane
description: Add aircraft to the Jutland game from supplied drawings, reference images, or historical specifications. Use when creating or updating entries in configs/planes.json5, producing transparent top-view plane PNGs, completing gun/bomb/torpedo/rocket references, assigning aircraft to carriers, or validating the resulting resource chain.
---

# Jutland 添加飞机

## 核心规则

- 在仓库根目录工作。先阅读 `AGENTS.md` 并执行 `git status --short`，保留所有无关用户改动。
- 先检查同阵营、同任务类型的现有飞机，再按现有命名、数值和素材风格做最小修改。
- 保持配置引用链完整：弹药 -> 机炮/释放器/火箭发射器 -> 飞机 -> 舰船编组。
- 保持飞机 `name`、PNG 文件名和舰船编组引用完全一致。
- 不替换、压缩或清理用户未要求处理的现有素材。
- 历史资料存在冲突时，说明采用的型号、数据口径和游戏近似；不要伪造精确值。

## 目标文件

- 飞机配置：`configs/planes.json5`
- 机炮：`configs/guns.json5`，弹药：`configs/bullets.json5`
- 炸弹/鱼雷释放器：`configs/releasers.json5`
- 航空火箭：`configs/plane_rocket_launchers.json5`
- 舰载机编组：`configs/ships.json5`
- 舰船图鉴：`configs/references.json5`
- 顶视图素材：`resources/images/planes/<type>/<name>.png`
- 图片加载器：`pkg/resources/images/plane/plane.go`
- 飞机类型和行为：`pkg/mission/object/unit/plane.go`
- 字段说明：`configs/Readme.md` 的飞机配置章节

## 工作流程

1. 确认输入与型号。
   - 从用户资料中确定内部名、展示名、国家、具体改型、任务类型和武器挂载。
   - 用户只给常用名时，结合仓库命名习惯选择简洁型号名，例如 `A7M2`；高风险歧义先询问。
   - 区分原型机、计划编组和实战型号，不把设计图展示数量默认当成完整载机量，除非用户要求按图配置。

2. 选择游戏类型。
   - 当前仅支持 `fighter`、`dive_bomber`、`torpedo_bomber`。
   - `fighter` 只选择飞机目标；其余两类只选择舰船目标，并使用对应移动/返航逻辑。
   - 侦察机、攻击机等缺少专用行为时，选择最接近的现有类型，在 `description` 和最终说明中明确近似。
   - 只有用户明确要求独立玩法，且现有类型无法表达时，才扩展 Go 枚举、目标选择、移动策略、资源目录和相关判断。

3. 生成顶视图素材。
   - 先检查现有同国飞机 PNG：透明 RGBA、机头朝上，通常约 `10 px/米`。
   - 用户提供组合图时，优先裁取无遮挡、结构完整的同型机实例；若飞机互相遮挡，换用另一实例，不猜测被遮挡轮廓。
   - 仅做裁剪、旋转、透明化、边缘清理和按真实尺寸缩放，不重绘用户素材，除非用户明确要求生成新美术。
   - 颜色掩膜不能只保留高饱和机身。逐项检查黑色发动机整流罩、螺旋桨、座舱、轮廓线、尾翼和武器等低饱和部件。
   - 去除甲板线、网格、升降机边框和相邻飞机残片。预览透明 PNG，并在黑底或棋盘格上检查白边、断裂和漏选。
   - 按类型保存：`fighter`、`dive_bomber` 或 `torpedo_bomber`；文件名必须是 `<name>.png`。

4. 添加飞机配置。
   - 从 1-3 架同阵营、同年代、同任务飞机推导 HP、减伤、加速度、转向、费用和建造时间。
   - `maxSpeed` 使用 km/h，`range` 使用 km，`length`/`width` 使用米，`tonnage` 使用吨；初始化阶段会转换速度和航程。
   - 描述列出角色、国家和游戏中实际配置的武器，不写入未配置挂载。
   - 战斗机配置前向机炮；俯冲轰炸机配置炸弹；鱼雷轰炸机配置鱼雷。多用途机若同时配置炸弹和鱼雷，会在当前开火循环中都尝试释放，必须先确认这种行为符合需求。
   - 新武器只能引用已存在名称。缺少释放器但已有弹药时，补充最小释放器条目；缺少弹药或机炮时继续补齐整条引用链。

5. 接入舰船和图鉴。
   - 用户要求可实际出击或任务上下文指向某艘航母时，更新该舰 `aircraft.groups`。
   - 同目标类型的多个编组按数组顺序消耗；据此安排战斗机、侦察机和不同攻击机的顺序。
   - 同步更新对应 `references.json5` 舰载机名称与数量。不要改动无关舰船或任务。

6. 验证。
   - 用 `git diff --check` 检查文本问题，用 `file` 或 `sips` 确认 PNG 为 RGBA 且尺寸合理。
   - 搜索每个新飞机名，确认配置、资源和所有舰船引用一致且没有重复定义。
   - 搜索每个机炮、炸弹、鱼雷和火箭名称，确认上游配置存在。
   - 逐张使用 `view_image` 检查完整轮廓、朝向、透明边缘及发动机/螺旋桨等深色部件。
   - 运行 `GOCACHE=/tmp/go-build go build -o /dev/null ./pkg/...`。修改 Go 行为时再运行相关单元测试和 `gofmt`。
   - 最终说明新增型号、类型近似、舰船编组变化、素材来源处理和实际执行的验证。
