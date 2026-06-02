# mission/state Agent 指南

本目录定义任务运行时的共享状态。`MissionState` 是任务管理器、控制器、绘制层、指令系统和 Hacker 终端之间的统一状态入口；修改这里会影响大量调用点，必须保持职责边界清晰。

## 分层设计

`MissionState` 只保留一层领域分组，不再直接堆放所有字段：

- `Core`：任务身份、任务状态、退出确认、任务元数据。
- `View`：屏幕布局和相机视野。
- `Player`：当前玩家、敌方玩家、资金等玩家侧运行状态。
- `Interaction`：选择、编组、当前增援点、当前召唤舰船等输入交互状态。
- `Arena`：任务战场内的运行对象集合，包括舰船、飞机、增援点、油井、弹药、尾流、摧毁对象和 UID 生成器。
- `UI`：界面展示相关状态，包括游戏标识、集结线提示、游戏显示选项和调试开关。

这个分层是为了让字段归属一眼可见，而不是为了隐藏状态或强接口化。后续使用时继续通过 `*MissionState` 传递状态，但字段访问应走对应分组，例如 `ms.Arena.Ships`、`ms.View.Camera`、`ms.UI.GameOpts`。

## 字段归属规则

- 关卡流程、暂停、胜败、模式切换字段放入 `Core`。
- 和屏幕尺寸、相机坐标、视野范围有关的字段放入 `View`。
- 和玩家身份、资金、阵营关系有关的字段放入 `Player`。
- 和鼠标键盘操作、选择状态、编组状态、当前 UI 选择项有关的字段放入 `Interaction`。
- 真正存在于任务战场中、会被游戏循环更新或绘制的对象集合放入 `Arena`。
- 只影响界面提示、调试显示、临时闪烁、标记绘制、显示选项的字段放入 `UI`。

如果一个字段看起来能放进多个分组，优先按“谁拥有这个状态”判断，而不是按“谁读取它”判断。例如集结点本身属于 `Arena.ReinforcePoints`，但当前展示哪条集结线属于 `UI.ShowRallyLinePointUid`。

## 修改原则

- 不要把新字段直接加回 `MissionState` 顶层；必须先选择一个子状态。
- 不要为了一次性字段新增 getter/setter；当前模式是清晰分组 + 直接访问。
- 不要把配置元数据拆离 `Core.MissionMD`，配置读取和初始化仍由 `NewMissionState` 负责。
- 不要在本包里引入绘制、输入检测、音频播放或 AI 决策逻辑；这里只保存状态和非常轻量的状态查询方法。
- 新增状态方法时，优先保持行为纯粹，例如 `Fleet()`、`CameraPosBorder()` 这种只基于状态计算结果的方法。
- 修改 `NewMissionState()` 时同时检查初始化对象、默认 UI/Debug/GameOptions 值，以及选中增援点等默认交互状态。

## 调用侧使用建议

- `manager` 可以读写全部分组，但应保持每个 update 函数只碰自己负责的状态。
- `drawer` 应以读取状态为主，避免在绘制函数中修改 `Arena` 或 `Player`。
- `controller` 和 `instruction` 可以生成或执行对 `Arena`、`Interaction`、`UI.GameMarks` 的改变，但不要绕过已有对象方法。
- `hacker` cheat 可以直接修改状态，但必须保持字段路径符合分组语义。
- 如果需要跨分组计算，优先放在使用方；只有多个包都需要且逻辑稳定时，再考虑加到 `state` 包。

## 验证

改动本包或字段路径后，至少运行：

```bash
GOCACHE=/tmp/go-build go build -o /dev/null ./pkg/...
```

完成前用 `rg` 检查是否误把旧字段路径重新引入，例如 `ms.Ships`、`m.state.Camera`、`misState.GameOpts`。旧顶层路径不应重新出现。
