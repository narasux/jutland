# manager

`manager` package 是任务关卡运行时的帧更新中枢。它把任务状态、绘制器、侧边栏、终端、玩家输入、电脑输入、指令集、音效和战斗结算串在一起，负责在每一帧推进 `MissionState`。

核心类型是 `MissionManager`。它本身不定义舰船、飞机、弹药或建筑的底层行为，而是按固定顺序调用这些对象和 `instruction` 的逻辑。

## 初始化

`New(mission, ui)` 创建一个完整的任务管理器：

- `state.NewMissionState` 初始化任务状态。
- `drawer.NewDrawer` 初始化任务绘制器。
- `sidebar.New` 初始化侧边栏 UI。
- `hacker.NewTerminal` 初始化调试终端。
- `NewInstructionSet` 创建运行中的指令集。
- 玩家一使用 `human.NewHandler(faction.HumanAlpha)`。
- 玩家二使用 `computer.NewHandler(faction.ComputerAlpha)`。
- `audioPlayer.NewWeaponFire` 初始化武器开火音效播放器。

目前玩家阵营是固定的：用户为 `HumanAlpha`，电脑为 `ComputerAlpha`。

## 每帧主流程

`Update()` 是任务运行的主入口。它先根据当前 `MissionStatus` 处理 UI 和模式输入，再在可模拟状态下推进游戏模拟，最后统一结算任务状态。

整体顺序如下：

1. 读取当前任务状态 `status := m.state.Core.MissionStatus`。
2. 如果是 `MissionRunning`，更新侧边栏；否则清除 `SidebarConsumesCursor`。
3. 按状态处理即时输入：
   - `MissionRunning`：处理集结线点击、集结点右键设置、滚轮缩放。
   - `MissionInTerminal`：更新终端。
   - `MissionPaused`：处理暂停菜单输入，并允许移动相机。
4. 如果当前状态会推进模拟，依次执行：
   - `updateCommandPhase`
   - 按状态更新相机或增援点选择
   - `updateSupportPhase`
   - `updateMapBlockPrewarm`
   - 运行态下更新选择和编组
   - `updateCombatPhase`
5. 调用 `updateMissionStatus` 处理胜负、暂停、地图、建筑和终端模式切换。
6. 返回更新后的 `MissionStatus`。

会推进模拟的状态由 `missionStatusRunsSimulation` 判断：

- `MissionRunning`
- `MissionInMap`
- `MissionInBuilding`

暂停和终端模式不会推进战斗模拟。终端模式只更新终端输入；暂停模式只处理暂停菜单和相机。

## 绘制入口

`Draw(screen)` 负责一帧绘制：

- 先调用 `drawer.Draw(screen, m.state, m.terminal)` 绘制主战场和终端相关内容。
- 再调用 `sidebar.Draw(screen, m.state)` 绘制侧边栏。

绘制不推进模拟状态。

## 命令阶段

`updateCommandPhase()` 包含两步：

1. `updateInstructions()`
2. `executeInstructions()`

`updateInstructions()` 的执行顺序是：

- `RemoveExecuted` 删除已经执行完成的指令。
- 调用人类玩家输入处理器，把返回的指令合并到指令集。
- 调用电脑玩家输入处理器，把返回的指令合并到指令集。

指令合并是覆盖合并：相同指令 UID 的新指令会覆盖旧指令。指令 UID 通常由对象 UID 和指令名组成，因此同一对象通常只能有一个同名指令，例如一艘舰不会同时保留两个普通移动目标。

`executeInstructions()` 调用 `InstructionSet.ExecAll(m.state)`，逐条执行当前指令。指令执行失败只记录日志，不中断本帧更新。

## InstructionSet

`InstructionSet` 是带锁的指令容器，内部字段为：

- `instructions map[string]instr.Instruction`

主要函数：

- `NewInstructionSet`：创建空指令集。
- `Add`：按 `Instruction.Uid()` 添加或覆盖单条指令。
- `Assign`：批量覆盖合并指令。
- `Remove`：按 UID 删除指令。
- `RemoveExecuted`：清理 `Executed() == true` 的指令。
- `Items`：返回当前指令 map。
- `Exists`：检查某个 UID 是否存在。
- `ExecAll`：遍历并执行所有指令。

注意：`Items()` 返回的是内部 map 本身，不是拷贝；调用者需要避免在无锁上下文中长期持有并修改它。

## 相机和缩放

`updateGameOptions(skipCursorInput)` 处理滚轮缩放：

- 如果 `skipCursorInput == true`，跳过鼠标滚轮。
- 鼠标滚轮向上调用 `StepZoomAtScreenPoint(1, sx, sy)`。
- 鼠标滚轮向下调用 `StepZoomAtScreenPoint(-1, sx, sy)`。
- 缩放以当前鼠标屏幕坐标为锚点。

`updateCameraPosition()` 根据当前模式选择相机移动策略：

- `MissionInMap` 使用 `getNextCameraPosInFullMapMode`。
- 其他模式使用 `getNextCameraPosInGameMode`。

得到下一帧位置后，会按 `Camera.BaseMoveSpeed` 对坐标取整裁剪，避免移动后出现黑边。

`getNextCameraPosInFullMapMode()` 用于全屏地图：

- 只有鼠标左键刚按下时才计算。
- 把点击的缩略图坐标换算成地图坐标。
- 将相机移动到点击位置居中。
- 调用 `EnsureBorder` 限制在地图边界内。
- 如果点击后相机位置没有变化，则退出全屏地图，模拟双击关闭效果。

`getNextCameraPosInGameMode()` 用于正常游戏画面：

- 根据鼠标是否悬停在屏幕边缘或角落决定移动方向。
- 每帧移动 `Camera.BaseMoveSpeed`。
- 移动后同样限制在地图边界内。

## 地图块缓存预热

`WarmupMapBlocks()` 和运行时的 `updateMapBlockPrewarm()` 使用同一套逻辑，用于分帧预热相机附近地图块的缩放缓存，减少首次缩放卡顿。

关键参数：

- `mapBlockPrewarmMargin = 2`：预热视野外额外边距。
- `mapBlockPrewarmIdleBudget = 4`：普通帧预热预算。
- `mapBlockPrewarmZoomBudget = 24`：缩放或视野变化后的突发预算。
- `mapBlockPrewarmZoomTicks = 45`：突发预算持续帧数。

触发重置的条件：

- 当前 zoom 变化。
- 相机位置变化。
- 相机宽高变化。

预热顺序：

1. 先预热当前 zoom 下相机附近地图块。
2. 如果当前视野附近仍有缺失块，本帧结束。
3. 如果还有剩余预算，再通过 `getAdjacentZooms` 预热相邻 zoom 档位。

`getAdjacentZooms(zoom)` 会在 `state.AvailableZooms` 中找到当前 zoom，并返回前后相邻档位。

## 交互选择和编组

`updateSelectedShips()` 只在 `MissionRunning` 下有效：

- 鼠标框选区域内的己方舰船会成为当前选择。
- 非编组模式下，按数字键会选中对应编组的己方舰船。
- 如果再次按下当前已选编组，会把相机移动到该编组第一艘舰船附近。
- 已被摧毁或不存在的舰船会从选择列表中移除。
- 如果选择列表为空，会重置 `SelectedGroupID`。

`updateShipGroups()` 只在 `MissionRunning` 下有效：

- 左 Ctrl 刚按下时切换编组模式。
- 左 Ctrl 松开时退出编组模式。
- 编组模式下按数字键，会先清除该数字对应的旧编组，再把当前选中舰船设置为该编组。

进入终端的快捷键包含 Ctrl，因此 `updateMissionStatus` 在进入终端时会强制关闭 `IsGrouping`，避免误留在编组模式。

## 增援点交互

`updateReinforcePoints()` 只在 `MissionInBuilding` 下有效，处理查看和配置己方增援点：

- 收集当前玩家所属的增援点 UID，并排序。
- 上/下方向键切换增援点。
- 左/右方向键切换当前增援点可提供的舰船。
- Enter 调用 `rp.Summon` 追加待增援舰船。
- Backspace 取消最后一个待增援舰船。
- 在界面中的缩略地图区域左键点击，可以设置增援点集结位置。

`setReinforcePointRallyPos(rp, pos)` 负责实际设置集结点：

- 坐标先裁剪到地图范围内。
- 如果目标格是陆地，设置 `RallySetFailedTick = 60` 并拒绝设置。
- 如果不是陆地，再裁剪实际坐标并调用 `rp.SetRallyPos(pos)`。

`updateRallyLineClick()` 用于正常游戏画面：

- 左键点击主地图。
- 如果点击位置距离己方增援点小于 3，则显示该增援点的集结线。
- 如果没有点中己方增援点，则清空集结线显示。

`updateRallyPointRightClick()` 用于正常游戏画面：

- 只有当前已经显示某个集结线时才生效。
- 右键点击主地图后，将当前显示的增援点集结点设置为点击位置。
- 如果增援点不存在或不属于当前玩家，会清空显示状态。

## 支援阶段

`updateSupportPhase()` 依次调用：

1. `updateGameMarks`
2. `updateBuildings`
3. `updateHospitalShipHealing`

`updateGameMarks()` 更新浮动文字等局内标识：

- 每帧递减 `mark.Life`。
- 生命周期归零后删除。
- `RallySetFailedTick` 大于 0 时每帧递减。

`updateBuildings()` 更新增援点和油井：

- 每个增援点调用 `rp.Update`，如果返回新舰船，就加入 `Arena.Ships`。
- 当前玩家召唤舰船会扣除 `CurFunds`。
- 电脑玩家资金暂时固定传入 `50000`，不受真实资金限制。
- 新生成舰船会收到一个前往集结点附近的 `ShipMove` 指令，随机散开范围是 `[-3, 3]`。
- 油井会检测附近己方货轮；货轮在油井半径内时加入装载列表，不在时移除。
- 装载计时完成后增加玩家资金，并生成金色收益文字。

`updateHospitalShipHealing()` 更新医疗船治疗：

- 只处理存活的医疗船。
- 治疗间隔使用真实时间 `time.Now().UnixMilli()`，固定为 5000ms，不受游戏速度倍率影响。
- 目标必须同阵营、存活、未满血、位于 `HospitalShipEffectRange` 内。
- 治疗量为 `ship.Length * ship.Width / 6`，不会超过目标最大 HP。
- 治疗时生成绿色浮动文字。
- 一艘医疗船完成一轮扫描后更新 `LastHealAt`。

## 战斗阶段

`updateCombatPhase()` 的执行顺序固定为：

1. `weaponFirePlayer.Update`
2. `updateShipWeaponFire`
3. `updatePlaneAttackOrReturn`
4. `updatePlaneWeaponFire`
5. `updateObjectTrails`
6. `updateShotBullets`
7. `updateExplosions`
8. `updateMissionShips`
9. `updateMissionPlanes`

这个顺序很重要：本帧先产生新弹药和飞机指令，再更新尾流、推进弹药、结算命中，最后把 HP 归零的单位移入消亡队列。

### 舰船开火

`updateShipWeaponFire()` 遍历所有舰船：

- 如果舰船已有 `AttackTarget`，且目标敌舰在 `MaxToShipRange` 内，则只把该目标加入候选。
- 如果没有指定目标，则收集射程内敌机和敌舰。
- 敌机使用 `MaxToPlaneRange` 判断。
- 敌舰使用 `MaxToShipRange` 判断。
- 候选不为空时随机选择一个目标并调用 `ship.Fire(enemy)`。
- 生成的弹药加入 `Arena.ForwardingBullets`。
- 只有开火舰船在当前相机内时，才统计音效。

音效统计逻辑：

- 鱼雷只记录是否有发射。
- 火箭只记录是否有发射。
- 普通炮弹记录本帧最大口径。
- 最后统一调用 `PlayShipFire`。

### 飞机出动、攻击和返航

`updatePlaneAttackOrReturn()` 分两段处理。

第一段遍历携带飞机的舰船：

- 没有飞机能力的舰船跳过。
- 如果舰船有 `AttackTarget`，直接作为候选目标。
- 否则从所有敌机和敌舰中随机选目标。
- 调用 `ship.Aircraft.TakeOff(ship, enemy.ObjType())` 起飞合适飞机。
- 起飞成功后加入 `Arena.Planes`。
- 立即添加 `PlaneAttack` 指令。

第二段遍历已经在场的飞机：

- 如果 `plane.MustReturn()`，添加 `PlaneReturn` 指令。
- 如果已有 `PlaneAttack` 指令，跳过。
- 否则根据飞机攻击对象类型选择敌机或敌舰作为新目标。
- 有目标则添加新的 `PlaneAttack` 指令。
- 没有目标则添加 `PlaneReturn` 指令。

当前飞机目标选择不检查作战半径，代码中已有 TODO。

### 飞机开火

`updatePlaneWeaponFire()` 遍历在场飞机：

- 空战飞机收集 `MaxToPlaneRange` 内敌机。
- 对舰飞机收集 `MaxToShipRange` 内敌舰。
- 候选不为空时随机选择目标并调用 `plane.Fire(enemy)`。
- 生成弹药加入 `Arena.ForwardingBullets`。
- 只有飞机在相机内时，才统计炸弹、火箭、鱼雷音效。
- 每类飞机音效本帧只需记录一次，最后调用 `PlayPlaneFire`。

### 弹药推进和命中

`updateShotBullets()` 先让所有 `ForwardingBullets` 调用 `Forward()`，再逐颗判断是否继续飞行或结算伤害。

命中计算分几类：

- 对舰直射：检查弹药上一帧到当前帧的线段是否与旋转舰体矩形相交。
- 对舰曲射：只有到达目标点附近时，检查目标点是否落在旋转舰体矩形内。
- 对空射击：按线段与旋转飞机矩形相交计算。
- 鱼雷如果碰到陆地，命中类型设为 `Land` 并停止。
- 生命周期归零但未命中时，命中类型设为 `Water`。

友军伤害受 `GameOpts.FriendlyFire` 控制。关闭友军伤害时，同阵营目标不会受伤；射手自己也不会被自己的弹药命中。

舰对空有额外命中概率限制：

- 对俯冲轰炸机，只有 `rand.Intn(12) == 0` 才继续判定命中。
- 对鱼雷机或战斗机，只有 `rand.Intn(8) == 0` 才继续判定命中。

对空火箭有近炸逻辑：

- 火箭生命结束、接近目标点，或进入敌机近炸半径时爆炸。
- 爆炸后对 `BlastRadius` 内合法飞机造成伤害。
- 生成局部火箭爆炸效果。
- 如果爆炸位置在相机内，播放火箭爆炸音效。

命中舰船或飞机后，如果开启 `DisplayDamageNumber`，会生成伤害数字：

- 普通命中：白色，字号 16。
- 三倍暴击：黄色，字号 20。
- 十倍暴击：红色，字号 24。
- Debug 的 `DamageColorByTeam` 开启时，会用青色/暗红区分敌我伤害。

## 尾流、爆炸和单位消亡

`updateObjectTrails()` 更新并生成尾流：

- 先更新已有尾流并删除生命周期结束的尾流。
- 战舰通过 `ship.GenTrails()` 生成尾流。
- 非炸弹弹药通过 `bt.GenTrails()` 生成尾流。
- 消亡中的飞机如果仍有速度，会在尾部生成火焰和黑烟尾流。

`updateExplosions()` 更新局部爆炸效果，并清理已经结束的爆炸。

`updateMissionShips()` 处理战舰消亡：

- 当战舰 `CurHP <= 0` 时，从 `Arena.Ships` 移入 `Arena.DestroyedShips`。
- 进入消亡队列时，把 `CurHP` 设置为 `textureImg.MaxShipExplodeState`，复用为爆炸动画状态。
- 相机内最多播放 2 次战舰爆炸音效。
- 消亡中的战舰每帧 `CurHP -= 0.5`。
- 速度每帧减少 `MaxSpeed / 30`，直到 0。
- `CurHP <= 0` 后从消亡队列移除。

`updateMissionPlanes()` 处理飞机消亡：

- 当飞机 `CurHP <= 0` 时，从 `Arena.Planes` 移入 `Arena.DestroyedPlanes`。
- 进入消亡队列时，把 `CurHP` 设置为 `textureImg.MaxPlaneExplodeState`。
- 使用 `RemainRange = rand.Float64() - 0.5` 临时保存坠落旋转偏转。
- 消亡中的飞机每帧 `CurHP -= 1`。
- 速度每帧减少 `MaxSpeed / 60`，直到 0。
- 如果仍有速度，会按当前旋转保持惯性前进，并限制在地图边界内。
- `CurHP <= 0` 后从消亡队列移除。

## 任务状态切换

`updateMissionStatus()` 在每帧末尾执行，用于处理胜负和模式切换。

胜负判断由内部函数 `calcNextStatusByShips` 完成：

- 只要还有 `DestroyedShips`，任务继续，避免沉没动画未结束就立刻胜负结算。
- 当前玩家没有任何存活舰船时，任务失败。
- 敌方没有任何存活舰船时，任务成功。
- 其他情况保持当前状态。

各状态下的输入：

- `MissionRunning`
  - Esc：进入暂停。
  - 同时做胜负判断。
- `MissionPaused`
  - Debug 模式下：Q 直接失败退出，Esc 直接恢复。
  - 普通模式下：Q、Esc、鼠标点击暂停面板按钮会交给 `state.ApplyPauseInput` 处理确认流程。
- `MissionInMap`
  - Esc：回到运行态。
  - 同时做胜负判断。
- `MissionInTerminal`
  - Esc：回到运行态。
  - 不继续处理后续快捷键。
- `MissionInBuilding`
  - Esc：回到运行态。
- 其他未知状态
  - 重置为 `MissionRunning`。

通用快捷键在上述状态处理后执行：

- M：在全屏地图和运行态之间切换。
- B：在建筑/增援点查看模式和运行态之间切换。
- LeftCtrl + LeftShift + `：进入终端，播放作弊音效，并退出编组模式。

## 当前实现特征

- `Update` 是严格的阶段式流程，阶段顺序会影响同一帧的结果。
- 玩家和电脑输入都先转成 `instruction`，再统一执行。
- 多数目标选择使用随机候选，而不是威胁评估或最优目标选择。
- 命中和消亡动画都在 manager 中集中结算，底层对象主要提供移动、开火、受伤、尾流等局部行为。
- 部分字段被复用于动画状态，例如消亡单位的 `CurHP` 和坠落飞机的 `RemainRange`。
- 医疗船使用真实时间间隔，和基于帧推进的战斗、尾流、爆炸不同。
- 电脑玩家资金暂不严格受经济系统限制。
