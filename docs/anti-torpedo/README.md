# 机炮反鱼雷设计（Autocannon Anti-Torpedo Defense）

> 状态：已实现（2026-11-24）。本文记录首版规则、实现链路与取舍，供后续调平衡和扩展。
> 代码位置以文件 / 函数名为准，行号会漂移。

**一句话**：舰上的小口径自动炮在来袭鱼雷接近时组织一次短促拦截，属于最后一道自卫手段，不是可靠的鱼雷盾。

## 目录

- [1. 目标与非目标](#1-目标与非目标)
- [2. 玩法规则](#2-玩法规则)
- [3. 实现链路](#3-实现链路)
- [4. 关键数值](#4-关键数值)
- [5. 设计取舍](#5-设计取舍)
- [6. 性能](#6-性能)
- [7. 平衡观测](#7-平衡观测)
- [8. 调参与后续优化](#8-调参与后续优化)
- [9. 已知限制](#9-已知限制)
- [10. 测试与验收](#10-测试与验收)
- [11. 代码索引](#11-代码索引)

---

## 1. 目标与非目标

### 目标

- 给舰船一个**近程、低概率、不依赖新配置字段**的鱼雷自卫手段。
- 复用现有防空炮的射界、射程、装填、散布与提前量计算，不另写弹道。
- 让“机炮反鱼雷”有可见表现（小水花），但不改变鱼雷命中舰船时的伤害结算。
- 不显著增加每拍开销，不在热路径引入堆分配。

### 非目标

- 不做史实级拦截概率；二战舰炮反鱼雷本身命中率很低。
- 不做舰队协同掩护：本舰只打朝自己来的鱼雷，不替旁边友舰拦截。
- 不做鱼雷耐久 / 多段命中 / 引信损伤模型；首版是“穿过命中框后单发概率引爆”。
- 不新增图鉴战力项、战报条目、拦截统计或专门 UI。
- 不为每座炮单独配置反鱼雷开关；首版按口径自动推导。

## 2. 玩法规则

### 2.1 哪些炮能反鱼雷

初始化阶段按“**对空 + 弹径 ≤ 40mm**”自动标记：

```text
AntiTorpedo = AntiAircraft && bullet.Diameter <= 40
```

- 覆盖 12.7 / 13 / 20 / 23 / 25 / 28 / 30 / 37 / 40mm 等小口径炮。
- 57mm 六磅炮及更大口径不参与；127mm 等高平两用炮不参与。
- `guns.json5` 不新增字段，`critical` 的推导逻辑在 `initGunMap`。
- 舰船初始化时把所有可反鱼雷炮的最大射程汇总到 `ShipWeapon.MaxToTorpedoRange`。

这是首版最大的隐式规则：**改口径阈值会直接改变所有舰船的反鱼雷能力**。

### 2.2 目标选择

每拍由 `MissionManager` 收集一次仍在航行的鱼雷（`TypeTorpedo`、未命中、`Life > 0`），
舰船只在火控就绪时查找目标：

- 敌我过滤：只打敌方鱼雷；友军鱼雷不拦。
- 射程：鱼雷到本舰中心距离 ≤ `MaxToTorpedoRange`。
- 航迹：鱼雷仍在本舰前方（沿鱼雷航向投影 > 0），且航迹到本舰中心的横向距离 ≤ `0.35` 格。
- 多目标：取距离最近的威胁鱼雷。

航迹判定用鱼雷当前航向的直线外推。制导鱼雷在转向时会有少量误差，但首版不为此增加采样。

### 2.3 开火

反鱼雷优先于对空：

1. 若本舰火控就绪且有来袭鱼雷，先尝试反鱼雷。
2. 一次只动用**一座炮**，按防空炮 → 副炮 → 主炮的顺序找第一座能开火的炮。
3. 该炮走正常的 `Gun.Fire`：提前量、射程、射界、散布、装填全部沿用。
4. 成功开火后，整舰进入 **1 秒反鱼雷火控冷却**。
5. 若所有炮都因射界 / 射程 / 装填无法开火，则不消耗冷却，下一拍继续尝试。

关键点：**不是每座炮各自有 1 秒冷却，而是整舰每 1 秒只组织一次拦截**。否则战列舰十几座防空炮会轮流开火，等效射速随炮座数量线性增长，重新变成“大舰鱼雷免疫”。

### 2.4 命中与表现

反鱼雷炮弹在发射时记录目标鱼雷指针（`Bullet.TargetTorpedo`），命中结算时只检查这一条鱼雷，不做“炮弹 × 全场鱼雷”的二次遍历。

每拍结算：

- 用本拍炮弹扫过的线段与鱼雷小矩形（`0.08 × 0.03` 地图格，约 10m × 4m）做相交。
- 相交后按 `antiTorpedoHitChance = 0.05` 掷一次概率。
- 成功：目标鱼雷和炮弹都标记为 `TypeTorpedo` 命中；鱼雷在**本拍内**移出战场，炮弹进入到达列表。
- 表现：鱼雷位置生成一个小水花（`KindSplash`，`0.5` 格，`30` tick），不生成伤害数字，不结算舰船伤害。
- 失败：炮弹继续飞，鱼雷继续前进，下一拍仍可能被同一发炮弹或其他炮弹判定。

已命中的鱼雷会在 `updateShotBullets` 末尾的统一清理段中移入到达列表，避免“炮弹在鱼雷已经推进过本拍之后才命中，鱼雷还多活一帧”。

## 3. 实现链路

### 3.1 目标类型

鱼雷原本是 `*objBullet.Bullet`，不是 `Hurtable` 战斗单位。为了复用 `Gun.Fire`：

- 在 `object.Type` 末尾追加 `TypeTorpedo`，不插在 `TypeWater` / `TypeLand` 前面，避免已有枚举值整体位移。
- 新增 `unit.TorpedoTarget` 适配器，把一条鱼雷包装成 `Hurtable`：
  - `ObjType()` 返回 `TypeTorpedo`；
  - `MovementState()` 返回鱼雷当前位置、航向、速度；
  - `Player()` 返回鱼雷所属阵营；
  - `ID()` 只返回固定字符串，鱼雷没有独立 UID。
- `Gun.Fire` 在命中类型为 `TorpedoTarget` 时，把真实鱼雷指针写进炮弹的 `TargetTorpedo`。

这样反鱼雷弹仍然走现有的 `Fire`、`InShotRange`、`CalcWeaponFireAngle`、装填和射界逻辑，不需要复制一套弹道代码。

### 3.2 管理端

`MissionManager` 新增：

- `antiTorpedoCandidates []antiTorpedoCandidate`：每拍收集的鱼雷候选，预存航向三角函数。
- `antiTorpedoTarget unit.TorpedoTarget`：逐舰复用的目标包装，避免每舰每帧堆分配。
- `rebuildAntiTorpedoCandidates()`：每拍遍历一次 `ForwardingBullets`，只挑航行中的鱼雷。
- `nearestThreatTorpedo(ship)`：在候选列表里做平方距离与航迹判定，取最近威胁。
- `shipFireTargets`：仅当 `ship.AntiTorpedoReady()` 时才查找鱼雷目标，冷却中直接跳过。
- `updateShotBullets`：新增 `TargetObjType == TypeTorpedo` 的命中分支与统一清理段。
- `hitEffect`：`HitObjType == TypeTorpedo` 返回小水花。

### 3.3 舰船端

- `BattleShip.antiTorpedoReadyTick`：整舰反鱼雷火控下一次可组织拦截的拍。
- `AntiTorpedoFireInterval = 1.0` 秒。
- `AntiTorpedoReady()`：`SimTick() >= antiTorpedoReadyTick`。
- `BattleShip.FireAt`：反鱼雷优先，`fireOneMount` 只允许一座炮开火，成功后写冷却。
- `ShipWeapon.MaxToTorpedoRange`：初始化时汇总所有可反鱼雷炮的最大射程。

## 4. 关键数值

| 常量 / 字段 | 值 | 位置 | 含义 |
|---|---:|---|---|
| `antiTorpedoMaxCaliber` | 40mm | `pkg/mission/object/initialize` | 可反鱼雷的最大弹径 |
| `AntiTorpedoFireInterval` | 1.0s | `pkg/mission/object/unit/ship.go` | 整舰火控组织间隔 |
| `antiTorpedoHitChance` | 0.05 | `pkg/mission/manager/combat.go` | 单发炮弹穿过命中框后的引爆概率 |
| `antiTorpedoThreatCorridor` | 0.35 格 | `pkg/mission/manager/combat.go` | 航迹到舰心的最大横向偏移 |
| `torpedoHitLengthCells` | 0.08 格 | `pkg/mission/manager/combat.go` | 鱼雷命中框长度（约 10m） |
| `torpedoHitWidthCells` | 0.03 格 | `pkg/mission/manager/combat.go` | 鱼雷命中框宽度（约 4m） |
| `interceptSplashCells` | 0.5 格 | `pkg/mission/manager/combat.go` | 拦截水花尺寸 |
| `interceptSplashLife` | 30 tick | `pkg/mission/manager/combat.go` | 拦截水花寿命 |

## 5. 设计取舍

### 5.1 为什么按口径自动推导，而不是加配置字段

优点：零配置迁移，287 条炮配置不用动，新增舰船自动获得一致行为。

代价：规则是隐式的，舰船无法单独关闭反鱼雷；12.7mm 机枪也会被算进来（虽然射程很短）；以后想做“只有某些炮座能反鱼雷”必须再加字段。

首版选择隐式推导，是因为这个功能还处于观察期，先避免配置面扩散。

### 5.2 为什么“整舰 1 秒一次 + 一次一座炮”

这是本功能最重要的平衡护栏。

第一版实现是“所有防空炮同时打同一条鱼雷”。实测战列舰单条鱼雷拦截率极高，因为炮座数量直接乘进射速。后来改成“每拍一座炮”，但战列舰仍有十几座炮可以轮流开火，等效射速依旧随炮座数增长。

最终改成整舰火控冷却：

- 每艘舰每 1 秒最多组织一次拦截；
- 一次只允许一座炮开火；
- 炮座数量只影响“哪座炮能打、射界覆盖好不好”，不再线性放大射速。

代价：驱逐舰和战列舰的反鱼雷上限接近，单舰炮座多不再有明显优势；这是有意为之，避免大舰免疫。

### 5.3 为什么单发只有 5%

反鱼雷应该像“最后一道近防”，不是可靠护盾。

5% 的含义是：炮弹确实扫过鱼雷命中框，也不保证引爆；鱼雷是海面小目标，弹丸可能擦过、引信可能不触发。这个值同时承担：

- 控制单条鱼雷的突防概率；
- 控制多雷齐射时被“顺带扫掉”的数量；
- 给后续平衡留一个单点旋钮。

### 5.4 为什么只做本舰自卫

- 实现简单：不需要处理“帮友舰拦截时鱼雷归属谁、战报记给谁、友军鱼雷误伤”等问题。
- 性能可控：每舰只查自己的威胁，不需要按舰队做共享目标分配。
- 玩法上保留“护航舰替主力舰挡雷”的设计空间，但首版不做。

代价：战列舰被多艘驱逐舰从不同方向夹击时，附近友舰不会帮忙，玩家需要自己编队。

### 5.5 为什么用炮弹指针锁定鱼雷

反鱼雷炮弹发射时记录 `TargetTorpedo *Bullet`，命中结算只检查这一条鱼雷：

- 避免每发反鱼雷弹每拍遍历全场鱼雷，复杂度从 O(炮弹 × 鱼雷) 降到 O(炮弹)。
- 鱼雷被其他炮弹提前击毁时，通过 `HitObjType != TypeNone` 直接跳过。
- 鱼雷在本拍已经推进过之后才被命中时，由统一清理段移出战场，不会多活一帧。

代价：炮弹持有鱼雷指针，依赖 `HitObjType` / `Life` 做失效判断；这是当前单线程模拟下的安全做法。

### 5.6 为什么反鱼雷优先于防空

来袭鱼雷的命中是“几秒内发生”，飞机投弹通常还有一段飞行 / 俯冲时间。首版让反鱼雷先消耗本拍开火机会，避免舰船在鱼雷已经贴近时还在追飞机。

代价：鱼雷和空袭同时到来时，防空火力会短暂转向海面，飞机压力会变大。如果后续觉得这个取舍太强，可以改成：

- 只有鱼雷进入更近的距离（如 1.5 格）才优先；
- 或按鱼雷预计命中时间与飞机威胁评分排序。

### 5.7 为什么不做鱼雷耐久 / 多段命中

首版用“低概率单发引爆”表达“鱼雷很脆弱，但机炮很难打中”。如果改成“鱼雷有 3 点耐久，每发命中扣 1 点”，需要新增鱼雷状态、命中反馈和平衡工作量，而当前玩法收益不明显。

未来若要做，建议与“反鱼雷命中率”二选一，不要同时叠加两层概率。

## 6. 性能

### 6.1 测量方法

临时基准，Apple M1 Pro / Go 1.26.2，合成 60 / 200 艘舰、20 / 100 / 400 条鱼雷，并混入 2000 发普通炮弹：

- 每轮执行 `rebuildAntiTorpedoCandidates()`；
- 再对每艘舰执行一次 `nearestThreatTorpedo()`；
- 统计 `ns/op` 与分配。

这是“每舰每拍都找雷”的**上界**。当前版本舰船在火控冷却中会跳过逐舰找雷，实际开销更低。

### 6.2 上界数据

| 场景 | 每拍开销 | 占 16.7ms 帧 |
|---|---:|---:|
| 60 舰 × 20 鱼雷 | 6.3 µs | 0.04% |
| 60 舰 × 100 鱼雷 | 24.6 µs | 0.15% |
| 60 舰 × 400 鱼雷 | 95 µs | 0.57% |
| 200 舰 × 100 鱼雷 | 77 µs | 0.46% |
| 200 舰 × 400 鱼雷 | 306 µs | 1.8% |

- 0 次堆分配。
- 200 舰 × 400 鱼雷已远超当前最大 65 舰的关卡规模。
- 当前版本把逐舰找雷频率降到“最多每舰每秒一次”，同时反鱼雷炮弹数量也因整舰冷却下降，实际成本低于上表。
- 余下的每拍固定成本是遍历一次 `ForwardingBullets` 收集鱼雷；`updateShotBullets` 本身已经遍历两遍弹药，这一遍同量级。

### 6.3 什么时候需要再优化

如果以后出现“数百条鱼雷同时在场”的极端场景，可以把 `antiTorpedoCandidates` 改成按 4~8 格分桶的鱼雷空间索引，逐舰只查邻域。当前规模不需要，避免为低频场景增加维护面。

## 7. 平衡观测

用真实舰船配置、真实鱼雷速度（`50/600 ≈ 0.083 格/拍`）、固定随机种子，跑 200 次单条鱼雷从 8 格外直冲本舰：

| 舰船 | 单条鱼雷被拦截率 |
|---|---:|
| Yamato | 4.0% |
| Bismarck | 4.5% |
| Iowa | 6.5% |
| Porter | 8.5% |

改“整舰冷却”之前，同一测试口径下单雷拦截率在 75% 以上（还是“每拍单炮 + 5%”的保守版本；初版全炮 10% 更高）。当前版本基本回到“最后一道近防”的定位。

需要注意：**战列舰对鱼雷的抗性不只来自反鱼雷**。以 Yamato 为例：

- 72,500 HP，水平减伤 0.55；
- 4,800 伤害的鱼雷实际约 `4800 × (1 - 0.55) ≈ 2160`；
- 需要约 33 发鱼雷才击沉；
- 8 发驱逐舰齐射约 17,000 伤害，占 24% 左右。

如果目标是“鱼雷对战列舰更有威胁”，应该单独调鱼雷对装甲的结算，而不是继续加强或削弱反鱼雷。候选方案：

1. 鱼雷命中改用垂直减伤或 `min(水平减伤, 垂直减伤)`；
2. 给鱼雷加“对主力舰伤害系数”；
3. 降低战列舰的水平减伤。

这三项都会影响所有鱼雷武器，属于独立平衡改动，不应和反鱼雷功能捆在一起。

## 8. 调参与后续优化

### 8.1 首版调参顺序

建议按以下顺序，每次只动一个：

1. `antiTorpedoHitChance`（反鱼雷整体强度最直接）；
2. `AntiTorpedoFireInterval`（整舰拦截频率）；
3. `antiTorpedoMaxCaliber`（哪些炮参与）；
4. `antiTorpedoThreatCorridor` / 鱼雷命中框（目标判定宽严）。

不要同时改“概率 + 冷却 + 口径”，否则很难从实战表现反推是哪一项造成的差异。

### 8.2 后续可做的扩展

- **每炮配置开关**：在 `Gun` 增加 `antiTorpedo` 配置字段，替代口径推导；可表达“只有某型 40mm 炮座能反鱼雷”。
- **舰种差异**：给驱逐舰 / 护卫舰更短的火控冷却或更高概率，让护航舰比战列舰更适合反鱼雷；当前整舰冷却刻意抹平了炮座数量差异。
- **鱼雷耐久**：一条鱼雷需要多次命中才被摧毁，与低概率模型二选一。
- **舰队掩护**：允许舰船拦截“航迹会命中附近友舰”的鱼雷，需要解决归属与友军误伤。
- **战报 / 统计**：记录拦截数、拦截方、鱼雷型号；当前只有水花表现。
- **空间索引**：鱼雷数量进入数百级后，再考虑分桶。
- **正式基准**：把临时基准固化成 `BenchmarkAntiTorpedoTargeting`，加性能门禁。

## 9. 已知限制

- 只做本舰自卫，不做友舰掩护。
- 只按鱼雷当前航向做直线航迹外推；制导鱼雷急转时可能误判。
- `TorpedoTarget.ID()` 固定为 `"torpedo"`，不是唯一 UID；不用于战报归属。
- 拦截不写 observer / 战报 / 统计，只有水花表现。
- 12.7 / 13mm 机枪也会参与反鱼雷；如果觉得不合适，降低口径上限或改成显式配置。
- 反鱼雷火控冷却挂在 `BattleShip` 的运行时字段上，不参与存档（当前没有存档系统）。
- 潜艇下潜时 `gunsBlocked` 仍生效，反鱼雷炮不会开火。
- 反鱼雷炮弹不会误伤舰船 / 飞机：`TargetObjType == TypeTorpedo` 的分支只结算鱼雷。

## 10. 测试与验收

自动化测试：

- `TestInitDerivesAntiTorpedoForAutocannons`：真实配置下 40mm 炮可反鱼雷、127mm 炮不可，porter 汇总出正射程。
- `TestAntiTorpedoFireControlCooldownCapsVolleys`：两门炮只开一门；1 秒冷却内不重复开火；冷却结束后恢复。
- `TestAntiTorpedoGunTargetsIncomingTorpedo`：来袭鱼雷被选为 `TypeTorpedo` 目标并写入指针。
- `TestAntiTorpedoHitDestroysTorpedo`：命中框相交 + 概率通过时鱼雷被移除。
- `TestAntiTorpedoProbabilityGateKeepsTorpedo`：概率未通过时鱼雷继续航行。
- `TestTorpedoOnCollisionCourse`：直冲、背离、横向偏移三种航迹判定。

命令：

```bash
GOCACHE=/tmp/jutland-gocache go test ./pkg/mission/... -count=1
GOCACHE=/tmp/jutland-gocache go build ./...
GOCACHE=/tmp/jutland-gocache go vet ./pkg/mission/...
```

手动验收：

1. 进入有战列舰 + 驱逐舰的关卡，让驱逐舰向战列舰发射鱼雷。
2. 观察鱼雷接近时是否偶发小水花、鱼雷是否消失。
3. 对比开启 / 不开启拦截时的战列舰掉血；单条鱼雷应大概率命中，不应出现“整轮齐射全被扫掉”。
4. 同时空袭时观察防空火力是否短暂转向海面，确认这个优先级符合预期。

## 11. 代码索引

| 职责 | 位置 |
|---|---|
| 口径推导 | `pkg/mission/object/initialize/initialize.go` `initGunMap` |
| 舰船射程汇总 | `pkg/mission/object/initialize/initialize.go` `initShipMap` |
| 鱼雷目标适配器 | `pkg/mission/object/unit/torpedo.go` `TorpedoTarget` |
| 炮弹目标指针 | `pkg/mission/object/bullet/bullet.go` `Bullet.TargetTorpedo` |
| 火炮类型过滤与指针写入 | `pkg/mission/object/unit/gun.go` `Gun.IsAvailableAntiType` / `Gun.Fire` |
| 整舰火控冷却 | `pkg/mission/object/unit/ship.go` `AntiTorpedoFireInterval` / `AntiTorpedoReady` / `FireAt` |
| 鱼雷候选与择敌 | `pkg/mission/manager/combat.go` `rebuildAntiTorpedoCandidates` / `nearestThreatTorpedo` |
| 命中结算与清理 | `pkg/mission/manager/combat.go` `updateShotBullets` |
| 拦截水花 | `pkg/mission/manager/combat.go` `hitEffect` |
| 测试 | `pkg/mission/manager/anti_torpedo_test.go` |
