---
name: jutland-design-map-mission
description: Design or revise Jutland missions on an existing map, including historical order of battle, ship placement and heading, opening fleet separation, player-centered initial camera placement, map-to-ship scale audits, same-class or similar-tonnage substitutions, JSON5 mission configuration, and navigability validation. Use when adding a scenario to configs/missions.json5, reconstructing a historical harbor or fleet layout, placing fleets on an existing .map/PNG pair, diagnosing terrain-versus-ship scale, or validating mission ship references and coordinates.
---

# Jutland 设计地图关卡

## 核心规则

- 在仓库根目录工作。先阅读 `AGENTS.md` 并执行 `git status --short`，保留无关用户改动。
- 把“历史考据”“地图尺度”“游戏部署”分成三个可独立验证的层次；不要用布阵微调掩盖地图比例错误。
- 在写 `configs/missions.json5` 前完成比例审计。若地图地标与舰船的相对尺度明显错误，先报告并确认修图或游戏化近似方案。
- 明确关卡快照时间，例如“1941-12-07 07:55”；不要混用战斗开始前、受击后或已起航舰船的位置。
- 历史舰不存在时，按“同名 -> 同级 -> 同类别且吨位/尺寸接近”替代。逐舰在 JSON5 注释中写明历史舰名、替代资源和理由。
- 舰首朝向使用游戏约定：`0°=北`、`90°=东`、`180°=南`、`270°=西`。区分舰首方向、泊位轴线和图片箭头方向。
- 港池、码头和干船坞在粗网格中可能变成岸格。允许移动到最近可航行格，但必须保持纵向顺序、内外舷关系和历史朝向，并在配置中说明。
- 只部署用户要求的舰种。辅助舰、潜艇、岸防设施和增援点不得因史料中出现而自动加入。
- 敌我舰队不得无意中过近。布阵完成后同时计算“双方编队质心距离”和“最近敌我舰距离”；无特殊设定时，以最近敌我警戒舰至少相隔 50 格（约 6.4km）为默认下限。航母战若地图允许，双方航母或主力编队宜相隔 80–100 格（约 10.2–12.8km），同时确认舰载机仍可抵达目标。史实近战、伏击或用户明确要求开场进入射程时可以例外，但必须在配置注释和交付说明中写明原因。
- `initCameraPos` 默认对准玩家阵营初始舰位的算术质心，而不是地图中心、双方战场中点或敌军方向。舰位全部确定后再计算 `round(sum(x)/n), round(sum(y)/n)`；玩家有多个远离的分舰队时，改用开场最需要操作的主力编队质心，并在配置注释中说明例外。

## 工作流程

1. **确认范围。**
   - 定义地图、年代、阵营、玩家方、敌方、部署舰种、是否包含增援和胜负玩法。
   - 检查 `configs/maps.json5`、`resources/maps/<map>.map`、`resources/images/map/abbrs/<source>.png` 是否完整一致。

2. **审计地图尺度。**
   - 读取网格宽高、PNG 尺寸和 `constants.MapBlockSize`。
   - 计算游戏逻辑范围：`网格格数 × MapBlockSize`。
   - 选择至少一个已知真实尺寸的地标和一型已知舰长，比较“地标长度/舰长”的历史比值与游戏比值。
   - 同时检查长轴和短轴；两者误差接近时通常是整体缩放问题，只有一轴错误时通常是非等比拉伸。
   - 查看 `pkg/resources/images/mapblock/block.go` 的地图块缩放，不把源 PNG 像素数直接当成游戏米制。
   - 详细方法和珍珠港案例见 [references/design-rules.md](references/design-rules.md)。

3. **建立历史锚点。**
   - 优先使用官方战斗报告、海军舰位图、同时期垂直航拍和港务泊位图；二手资料只用于补缺。
   - 先标地标、泊位和编队中心，再放单舰。记录证据、时间点、估算误差和朝向来源。
   - 将参考图坐标映射到游戏网格时，至少使用两个方向上的控制点；地图有旋转或形变时使用仿射变换，不只按截图宽度等比换算。

4. **盘点舰船资源并确定替代。**
   - 从 `configs/ships.json5` 获取名称、国家、类型、级别线索、长度和吨位。
   - 保持舰种不变；不要以战列巡洋舰替代轻巡、以大型驱逐舰替代巡洋舰。
   - 同级替代优先于吨位近似。吨位接近时再比较长度、年代、航速和武器角色。
   - 为每艘历史舰维护一条显式映射，不依赖相同资源名的重复次数表达身份。

5. **写入任务。**
   - 沿用相邻关卡字段和 JSON5 风格，最小修改 `configs/missions.json5`。
   - 为新任务补齐四种显示名和描述：`displayName`、`displayNameEn`、`displayNameRu`、`displayNameJa` 及对应描述。
   - 先写防守方历史泊位，再写进攻方编队；按舰种和泊位组分段。
   - 注释至少包含历史舰名/编号、实际资源名、替代原因、泊位或编队位置、舰首方向。
   - 舰位完成后计算双方质心距离、最近敌我舰距离，以及航母战的最近敌我航母距离；距离不足时优先整体平移编队，保持内部队形不变，并重新检查边界、岸格和舰载机航程。
   - 用最终舰位计算玩家阵营质心并写入 `initCameraPos`。单一编队必须四舍五入到最近网格；多个远离的玩家编队则聚焦开场主力编队，不得默认使用地图中心或双方中点。
   - 无增援或油井时使用空数组。

6. **验证并交付。**
   - 运行：
     ```bash
     go run .codex/skills/jutland-design-map-mission/scripts/validate_map_mission.go \
       -mission <mission-name>
     ```
   - 对独立岛屿做比例检查时附加：
     ```bash
     go run .codex/skills/jutland-design-map-mission/scripts/validate_map_mission.go \
       -mission <mission-name> \
       -land-component <x,y> \
       -real-length <meters> \
       -real-width <meters>
     ```
   - 修复全部错误；逐项判断舰体边缘压岸警告是否属于粗网格泊位近似。
   - 核对 `initCameraPos` 与玩家舰队质心的偏差：单一编队应不超过 1 格；存在分舰队例外时，确认镜头落在指定主力编队内部。
   - 在配置注释或交付说明中给出双方质心距离、最近敌我舰距离、航母距离（如适用）和采用的最小间距例外。
   - 运行 `git diff --check -- configs/missions.json5`、相关包测试、`go test ./...` 和 `go build ./...`。
   - 最终说明双方舰数、替代策略、尺度结论、开场舰距、初始镜头坐标、粗网格位移、实际验证和未做的视觉运行检查。

## 禁止事项

- 不在未经审计的地图上直接照图撒点。
- 不声称“历史精确”而不给出快照时间、证据和网格近似说明。
- 不只放大舰船贴图来修复尺度；这会让碰撞、射程、航速和视觉尺寸不一致。
- 不因历史舰缺失而静默替换或改变舰种。
- 不把舰体中心位于水面当作完整通过；还要检查舰首、舰尾和并靠间距。
- 不让两军因照搬参考图、交换位置或整体平移后意外贴近；每次舰位调整后都重新计算敌我最短距离。
- 不在舰位变更后沿用旧的 `initCameraPos`，也不把地图中心或双方中点当作玩家镜头的默认值。
