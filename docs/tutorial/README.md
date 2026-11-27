# 教学关卡设计草案

> 状态：第一版已实现（`wip-tutorial` 分支），待实机测试。最后更新：2026-11-26。
> **一句话**：教学 = 真实任务关卡 + 可配置步骤控制器 + 键位/鼠标提示层；复用现有地图、舰船、相机和 HUD，不做一套新场景。

## 1. 推荐方案

- 新增 `MissionCategoryTutorial = "tutorial"`，任务选择页加「教学」页签。
- **一章一个教学 mission**，脚本放 `configs/tutorials.json5`，场景仍写在 `missions.json5`。
- 新增 `pkg/mission/tutorial` 步骤控制器，只在教学任务创建；普通关卡零开销。
- 新增 `drawer/tutorial.go` 画底部步骤卡、键帽、鼠标徽标；用 `vector` + `text` 手绘。
- 完成判定优先读状态结果（选中了谁、开没开地图、潜没潜），不只看按了哪个键。
- 教学任务绕过常规胜负和战报；完成/退出回教学列表。
- 第一版不做存档、不做顺序解锁、不做全屏键盘。

## 2. 为什么不能只加一个分类

1. **分类是二选一模型**：`Tab`、两个按钮、`missionSelectUI`、`drawMissionSelect` 和测试都写死 classic/test。
2. **教学关会自动胜利**：`calcNextStatusByShips` 在没有敌方舰船时返回 `MissionSuccess`；只有玩家舰队的教学关第一拍就通关。必须显式绕过。
3. **输入没有统一入口**：移动/侦察在 human controller，编组/相机在 manager，暂停/地图/终端在 status，侧栏和单位面板又各自处理鼠标。不要在每个输入点插 `if`。
4. **缺少键帽、进度和教学文案设施**：现有 i18n 要求五套目录同步，UI 目录还有“禁止硬编码中文”测试。

## 3. 配置与数据

### 3.1 教学任务

```json5
{
  name: "TutorialBasic",
  category: "tutorial",
  tutorialScript: "basic",       // v1 建议必填
  displayName: "基础操作",
  // displayNameEn / Ru / Ja / ZhHant 按现有任务字段补齐
  initCameraPos: [[14, 30], [14, 30]], // P2 行留同值；教学固定 P1
  mapName: "darwin",
  initFunds: 0,
  initReinforcePoints: [],
  initOilPlatforms: [],
  initAirfields: [],
  initShips: [
    { name: "johnston",  pos: [20, 36], rotation: 90, belongPlayer: "P1" },
    { name: "johnston",  pos: [24, 40], rotation: 90, belongPlayer: "P1" },
    { name: "baltimore", pos: [30, 37], rotation: 90, belongPlayer: "P1" }
  ]
}
```

示例坐标仅示意，落地时按水域/舰体重新校验；仍走 `validateMissionShipPlacement`。第一版不放 P2，AI 不行动，代价是必须绕过自动胜利。

### 3.2 教学脚本

```json5
[{
  id: "basic",
  mission: "TutorialBasic",
  title: "基础操作", titleEn: "Basic Controls",
  titleRu: "Основы управления", titleJa: "基本操作", titleZhHant: "基礎操作",
  steps: [
    {
      id: "select-fleet",
      mode: "guided",
      title: "先选中舰队", titleEn: "Select the fleet",
      // 其余语言字段同上
      body: "按住鼠标左键拖出一个框，框住海上的舰船。",
      bodyEn: "Hold the left mouse button and drag a box around the ships.",
      // 其余语言字段同上
      hint: { actions: ["select-area"] },
      setup: ["clear-selection"],
      completion: { kind: "selectedShips", min: 1 }
    },
    {
      id: "open-map",
      mode: "guided",
      title: "看一眼全图", titleEn: "Check the full map",
      body: "按 M 打开全图，再按 M 或 Esc 返回。",
      bodyEn: "Press M for the full map; press M or Esc to return.",
      hint: { actions: ["map"] },
      completion: { kind: "mapOpened" }
    }
  ]
}]
```

- `mode`：`info`（信息卡，冻结）、`guided`（实时，等动作）、`scripted`（自动演示，后置）、`free`（练习，按钮结束）。
- `actions` 是语义 ID（`select-area`、`move`、`group`、`dive`、`map`…），由代码映射成键帽，不把 `ebiten.Key` 写进配置。
- `setup` 只放安全动作：`clear-selection`、`select-player-fleet`、`focus-airfield`、`open-sidebar`、`close-sidebar`。`clear-selection` 要同时清 `SelectedShips`、`FocusedShipUid`、`SelectedGroupID`。
- 步骤文案放 `tutorials.json5`，每种语言一个字段；UI 框架文案（页签、下一步、跳过、完成）放 i18n，五套 TOML 一次补齐。

启动期 `log.Fatal` 校验：脚本/任务一一对应、mission 必须是 tutorial、step id 唯一、mode/completion/action/setup 已知、所需舰种/机场/增援点存在。

### 3.3 类型草案

```go
type Step struct {
    ID         string
    Mode       StepMode
    Title      LocalizedText
    Body       LocalizedText
    Hint       Hint         // Actions []ActionID, Hold bool
    Setup      []SetupAction
    Completion Completion   // Kind, Min, Distance, Group
}
```

## 4. 运行时

### 4.1 每帧流程

```text
MissionManager.Update
  1. 教学信息卡/完成卡：教学层独占输入，冻结模拟，跳过普通更新
  2. 普通任务更新（侧栏、相机、输入、模拟）
  3. 教学层 Observe(状态快照) -> 完成则下一步 / 整章完成
```

信息卡冻结时**不要**设 `MissionStatus = MissionPaused`，否则会画暂停遮罩、Esc 语义冲突；用教学层自己的 `hold` 标志。冻结时清掉 `UI.ShowSonarRanges`、`UI.UIConsumesCursor` 等残留。

### 4.2 完成判定

每步进入时记基线，之后比较状态变化：

| 动作 | 判定 |
|---|---|
| 框选舰队 | `SelectedShips` 数量 ≥ min |
| 右键/方向键移动 | 选中舰位移或速度变化 |
| Ctrl+数字编队 | `BattleShip.GroupID` 被赋值 |
| 数字键选队 | `SelectedGroupID` + 选中集 |
| X 散开 | 静止舰位置变化 |
| D 潜航 | `Submarine.DiveState` 变化 |
| V 声呐 | `UI.ShowSonarRanges` 为 true（按住期间） |
| B/M/Esc | `MissionInBuilding` / `MissionInMap` / `MissionPaused` |
| Shift+右键侦察 | 出现 `ScoutManual` 飞机或侦察标记 |
| 滚轮/边缘滚屏 | `GameOpts.Zoom` / `Camera.Pos` 变化 |
| 单位面板/机场 | 对应 `Disabled` / `CurProducing` 变化 |
| 增援召唤 | `ReinforcePoint.OncomingShips` 增加 |

原则：一步一个动作；做错不惩罚；必须建立基线，避免上一步残留状态直接完成；没有状态可读的动作（如纯按键提示）才看输入。

### 4.3 输入与退出

- **信息卡**：只响应 `Enter`/`Space`/下一步、`Esc` 退出、跳过；不响应游戏输入。
- **guided**：不拦截正常输入，玩家可暂停/开地图/点面板；教学层只观察。暂停/地图/终端期间教学暂停推进，返回后继续。
- **完成卡**：只保留“返回教学列表 / 重玩本节”。

教学任务必须显式绕过 `calcNextStatusByShips`，但**不要整个跳过 `updateMissionStatus`**：暂停、地图、增援、终端的输入仍要处理，只替换 running 分支的存亡判定。第一版不新增 `MissionStatus`，用 `MissionManager.TutorialExitRequested()` + 控制器 `ExitKind` 表示完成/退出，避免踩 `updateMissionStatus` 的 `default` 重置分支。

暂停页的 `Q` 在教学中映射为“放弃教学”，不要走 `MissionFailed`；教学完成/退出不进 `MissionReport`，不播失败音效，回列表时清掉 manager/音频/加载标志。

## 5. UI

### 5.1 任务选择页

- 第三个页签「教学」，`Tab` 循环、跳过空分类；空分类建议隐藏。
- 教学页签隐藏 P1/P2 选择；统计行可改成「步骤 N / 本章要点」。
- 开始按钮可用「开始教学」；描述用任务 `description`，长文用现有 `wrapText` 换行。

### 5.2 任务内步骤卡

底部居中，避开右上侧栏和左上 FPS：

```text
┌─ 教学 2 / 8 ─────────────────────────────┐
│ 让舰队移动                                │
│ 在海上右键点击一个位置，选中的舰会驶过去。 │
│ [鼠标右键] 点击海面              [跳过]   │
└──────────────────────────────────────────┘
```

- 标题一行；正文 2~3 行；英文/俄文按词换行，日文按字符换行。
- 键帽条推荐：`[Ctrl] + [1]`、`[V] 按住`、`[鼠标右键]`；鼠标动作画左/右键徽标。
- 迷你键盘图可选，第一版不做全屏键盘图。
- `MissionInMap` / `MissionInBuilding` / 暂停 / 战报 / 终端下按模式缩小或隐藏卡片，避免遮挡。
- 侧栏/单位面板布局目前私有，第一版不做指向具体按钮的聚光；只做地图区域和键帽。

## 6. 第一版内容

| 章节 | 场景 | 教什么 |
|---|---|---|
| 1. 基础操作 | `darwin` 开阔水域，3 艘 P1 舰 | 框选、右键移动、滚轮缩放、M 全图、Esc 暂停/继续 |
| 2. 编队与机动 | `darwin` 开阔水域，5 艘 P1 舰 | Ctrl+数字编队、数字键选队、X 散开、方向键微调 |
| 3. 潜艇操作 | `darwin` 开阔水域，1 艘潜艇 | D 上浮/下潜、V 声呐范围、水下限制 |
| 4. 航空侦察 | `darwin` 近岸水域，1 艘航母 + 1 艘战列舰 + 北岸机场 | 选中航母 / 战列舰、Shift+右键派侦察机（含战舰水上侦察机）、机场面板切换生产机型 |
| 5. 增援与经济 | `darwin` 开阔水域，增援点 + 资金 | B 增援页、左右选舰、Enter 召唤、Backspace 取消、Esc 返回 |
| 6. 火力打击 | `darwin` 近岸海域，巡洋舰 + 驱逐舰 + 航母，不还击的运输舰靶船 + 敌岸基机场 | 对舰鱼雷、对舰火炮、对空拦截、舰载机空袭；右键指定目标、单位面板开关武器 |

后续可做：陆地机场生产、撞击/碰撞/迷雾规则信息卡、Hacker 彩蛋章节。

README 已同步修正 X 散开重复、暂停 Q 的语义和全屏地图退出方式。撞击伤害默认值仍以 `game_settings.json5` 为准，教学文案不写死。

## 7. 实现与验证

已改动：

- `pkg/mission/metadata`：分类常量、`normalizeMissionCategory`、`tutorialScript`。
- `pkg/game`：三分类切换/绘制、教学开始/退出、`missionSelectUI`。
- `pkg/mission/manager`：创建/更新/绘制教学控制器；绕过胜负；暂停退出特判。
- 新增 `pkg/mission/tutorial`、`drawer/tutorial.go`、`configs/tutorials.json5`。
- `i18n` 五套文案；`configs/Readme.md`、`README.md`/`README_EN.md` 同步。

验证：

- 单测：分类循环、脚本校验、完成判定、键帽布局、文案回退。
- 集成：教学关不自动胜利、信息卡不推进 `SimTick`、完成/暂停/退出、普通关卡回归。
- 手动：1280×720 / 1920×1080、五语言、鼠标动作、暂停/地图/增援/终端遮挡。

## 8. 第一版已采用的值

- 分类：`tutorial`，页签顺序 `经典 → 教学 → 测试`。
- 结构：一章一个 mission；当前五章（基础、编队、潜艇、航空、增援）。
- 场景：`darwin` 开阔水域，仅 P1，无敌人，不新建地图。
- 节奏：信息卡冻结，操作步骤实时；卡片本身消费鼠标点击。
- 提示：卡片内键帽条 + 鼠标徽标；可选 `target` 目标点画脉冲圈；不做全屏键盘图。
- 完成：状态快照判定；正常完成写入 `gameSettings.tutorialCompleted`，任务选择页显示「已完成」；首次启动优先进入教学列表。
- 反馈：操作步骤超过 12 秒显示额外提示；完成卡支持「重玩本章」。
- 关卡配置：`configs/tutorials.json5`；任务通过 `category: "tutorial"` + `tutorialScript` 关联。

后续可选：陆地机场生产、规则信息卡、Hacker 彩蛋章节、迷你键盘图、幽灵鼠标演示。
