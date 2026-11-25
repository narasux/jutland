# 撞击与碰撞系统设计（Ramming & Collision）

> 状态：P0（数据/纯函数）、P1（接触 + 撞击伤害）与 P2（敌军分离 + 掉速）已实现。最后更新：2026-11-26。
> 关联文档：[潜艇系统设计](../submarine/README.md)、[仿真与平衡评估](../simulation/README.md)。

**一句话**：把撞击从「射程 1.5 格、装填 0.001 秒的火炮」改成船体上的 `ram` 属性；把「接触检测 → 碰撞响应 → 撞击伤害」拆成三层，先交付伤害层，后续接入真正的船-船碰撞时 `ram` 配置不改。

**已定决策**：水滴用 `canHitPlanes` 保留撞机；冲角数值见 §5.4 的定稿表；友军不检测、不阻挡、不分离；`enableRamming` 默认关闭，控制战列舰 / 巡洋舰 / 驱逐舰 / 护卫舰的撞击伤害，`alwaysOn` 特殊单位不受影响；敌方舰船的碰撞分离与掉速由独立的 `enableShipCollision` 控制，默认开启。

**性能结论**：原型实测当前 57 舰任务每拍额外约 0.01ms（占 16.7ms 帧 0.06%）；200 舰密集约 0.08ms p50 / 0.1ms p99（<0.6% 帧）。碰撞不会造成可感知的帧率下降，风险只在每拍分配 map/slice 带来的 GC。

---

## 1. 改造前的现状与问题

改造前 `waterdrop` 和 `swordfish` 用一门极短射程的主炮模拟撞击（现已迁移为 `ram`）：

| 舰船 | 武器 | 炮弹 | 发数 | 装填 | 射程 | 对空 |
|---|---|---|---|---|---|---|
| `waterdrop` 水滴 | `Impact` | `SE/GB/1/2048`（伤害 99999、暴击 0.5） | 3 | 0.001s | 1.5 格 | 是 |
| `swordfish` 巨大剑鱼 | `SwordfishImpact` | 同上 | 3 | 0.001s | 1.5 格 | 否 |

相关位置：`configs/guns.json5`、`configs/ships.json5`（`waterdrop` / `swordfish`）、`pkg/mission/object/combatpower/config_regression_test.go`。

这套实现的问题：

1. **伤害与航速、吨位、角度无关**。停船和全速冲刺伤害一样，小船撞大船和大船撞小船一样。
2. **和舰体几何脱节**。火炮从舰体中心（`posPercent: 0`）发射，射程 1.5 格是「点对点距离」，与舰长、舰宽、舰艏是否接触没有关系；大舰可能还没碰到就开火，也可能已经重叠还打不中。
3. **一轮三发、每发都是 99999**。伤害数字、战报、音效都像炮击，不像一次撞击。
4. **对空能力语义混乱**。`Impact` 能打飞机，但「水面舰船撞飞机」没有碰撞依据。
5. **战力与造价失真**。`gunDPS` 用 `99999 × 3 / 0.001` 计算，靠后续效能系数压回来；撞击的真实价值（接触、冷却、自身受损）无法进入 `CombatPower` 与费用评估。
6. **无法复用给早期军舰的冲角**。冲角是舰体结构，不是可挂载火炮；每艘冲角舰都要复制一门假炮和一套射界。
7. **没有碰撞响应**。舰船目前可以互相穿过，未来加入碰撞时这套假炮反而要重写。

---

## 2. 目标与非目标

### 目标

- 用船体配置描述「这艘船有冲角 / 会撞击」，不依赖 `guns.json5`。
- 撞击伤害由**接近速度、撞击角度、双方吨位、目标减伤**决定，结果可解释、可复现。
- 先交付「接触即伤害」的可用版本，且与未来的船-船碰撞系统共用同一套接触检测。
- 保留现有战报、观察者、伤害数字、命中特效、统计与击沉归因链路。
- 撞击数值进入 `CombatPower` 与费用评估，AI 能识别撞击舰。
- 2 倍速、迷雾、潜艇、飞机、友军等边界行为有明确规则。

### 非目标（本方案不包含）

- 完整的刚体物理、动量守恒、舰体断裂、进水与倾斜模拟。
- 常规舰船撞击飞机（只有 `canHitPlanes` 的特殊单位例外）、撞击岸防工事、接舷战。
- 重做全部伤害公式或穿甲模型。
- 为航母、货轮、维修船、医疗船、潜艇、鱼雷艇补默认冲角；普通战斗舰的默认撞击按 §4.3 的舰种表推导。
- 存档兼容（当前无存档系统）。

### 玩法定位

撞击是**早期军舰的近战终结手段**：1890 年代前后的铁甲舰 / 前无畏舰主炮射速慢、远距离命中率低，冲角提供了一种「逼近后一次定胜负」的高风险选项。命中收益高，但自伤、掉速和脱离困难让它不适合作为后期主力舰的常规战术。

因此：

- 冲角主要配置在 19 世纪末 ~ 1905 年前后的舰船上（如 `monterey_m_6`，以及后续的 Camperdown、Victoria 级）；
- 二战舰船默认不配置 `ram`，避免所有舰船都变成「自杀式撞击船」；
- `swordfish` / `waterdrop` 作为特殊单位保留「接触即秒杀」，与历史冲角区分开；
- 未来若加入「撞击指令」，也只对 `ram.damage > 0` 的舰船开放。

---

## 3. 术语与分层

| 术语 | 含义 | 是否造成伤害 |
|---|---|---|
| **接触（contact）** | 两艘舰船的舰体轮廓发生重叠或扫掠相交 | 否 |
| **碰撞（collision）** | 接触后的物理响应：阻挡、分离、掉速 | 默认开启（`enableShipCollision`） |
| **撞击（ram）** | 有冲角的舰船以舰艏、足够接近速度撞上敌舰 | 是 |

三层职责：

```text
几何层  detectShipContacts()     纯函数：谁和谁接触、接触点、法线、穿透深度
响应层  resolveShipContact()    分离 + 掉速（敌军已实现；友军跳过）
伤害层  resolveRamImpacts()      撞击判定、伤害公式、冷却、战报与表现
```

关键约束：**伤害层只消费接触事件，不关心接触是怎么算出来的**。因此 P2 把「重叠检测」升级为「扫掠 + 连续碰撞」时，`ram` 的数据和伤害公式不变。

---

## 4. 数据模型

### 4.1 舰船字段

在 `pkg/mission/object/unit/ship.go` 的 `BattleShip` 增加：

```go
// Ram 冲角 / 撞击配置；零值表示没有撞击攻击能力（碰撞时只阻挡，不结算撞击伤害）
Ram RamConfig `json:"ram"`
```

新文件 `pkg/mission/object/unit/ram.go`：

```go
// RamConfig 描述一艘舰船的冲角与撞击能力。
type RamConfig struct {
    // Damage 满速、正角度撞上同吨位目标时的基础伤害（HP）。
    // <= 0 表示该舰没有撞击攻击能力；碰撞仍会阻挡（P2 起）。
    Damage float64 `json:"damage"`
    // Cooldown 同一对舰船两次撞击结算的最小间隔（秒）。默认 3。
    Cooldown float64 `json:"cooldown"`
    // SelfDamageRatio 撞击方按基础伤害承受的反作用比例，0~1。默认 0.1。
    SelfDamageRatio float64 `json:"selfDamageRatio"`
    // MinSpeed 有效撞击的最低接近速度（节）。默认 3；低于该值只接触不结算。
    MinSpeed float64 `json:"minSpeed"`
    // BowArc 舰艏撞击判定的半角（度）。默认 60；舰艏指向目标偏离超过该角度不算撞击。
    BowArc float64 `json:"bowArc"`
    // CanHitPlanes 是否允许撞击飞机。默认 false；仅特殊飞行单位（水滴）开启。
    // 开启后接触检测额外检查敌机，命中走同一套伤害与冷却。
    CanHitPlanes bool `json:"canHitPlanes"`
    // AlwaysOn 无视游戏设置里的撞击总开关，始终启用。特殊单位（水滴 / 巨大剑鱼）使用。
    AlwaysOn bool `json:"alwaysOn"`
    // Derived 按舰种 / 吨位 / 装甲推导出的默认能力；不参与静态战力与费用。
    Derived bool `json:"-"`
}
```

`Normalize()` 补齐默认值并夹取范围（`Cooldown` 最小 0.5、`SelfDamageRatio` 夹到 0~1、`MinSpeed` 最小 0、`BowArc` 夹到 15~90），在 `initialize.go` 加载舰船时调用；`Damage < 0` 直接 `log.Fatalf`，沿用启动期配置错误的处理方式。

### 4.2 JSON5 示例

```json5
// 巨大剑鱼：接触即秒杀的特殊单位，无自伤；alwaysOn 绕过游戏设置开关
{
  name: "swordfish",
  // ...
  ram: {
    damage: 300000,      // 对 Yamato（72500 HP、55% 水平减伤）50 节正撞约 135000
    cooldown: 2,
    selfDamageRatio: 0,
    minSpeed: 5,
    alwaysOn: true,
  },
}

// 水滴：保留现有「可对空」设定，用 canHitPlanes 撞飞机；alwaysOn 绕过游戏设置开关
{
  name: "waterdrop",
  // totalHP: 1, maxSpeed: 1000, 水平/垂直减伤 1.0
  ram: {
    damage: 10000000,   // 对 500 万 HP 的翻车鱼也能全速一击
    cooldown: 2,
    selfDamageRatio: 0,
    minSpeed: 10,
    canHitPlanes: true,
    alwaysOn: true,
  },
}

// 早期铁甲舰 / 冲角舰（示意，尚未入配置）
{
  name: "camperdown",
  // totalHP: 10600, maxSpeed: 16.5, ...
  ram: {
    damage: 4800,        // ≈ 0.45 × totalHP
    cooldown: 4,
    selfDamageRatio: 0.25,
    minSpeed: 4,
  },
}

// 蒙特雷号（1893，已有舰船）：可作为第一艘真实冲角舰试点
// totalHP 4084、maxSpeed 13.6、水平减伤 0.24
{
  name: "monterey_m_6",
  ram: {
    damage: 1800,        // ≈ 0.44 × totalHP
    cooldown: 4,
    selfDamageRatio: 0.25,
    minSpeed: 4,
  },
}
```

没有 `ram` 字段的舰船不造成撞击伤害，但 P2 起仍会参与碰撞分离与掉速；航母 / 货轮 / 潜艇 / 鱼雷艇属于这一类。

### 4.3 游戏设置开关

`config.GameSettings.EnableRamming`（`configs/game_settings.json5` 的 `enableRamming`）默认 `false`，设置页「撞击伤害」可即时切换。规则：

- 普通战舰：显式配置 `ram` 的舰船用配置（存量的战列 / 巡洋 / 驱逐 / 护卫舰均已逐舰写入）；未配置的按舰种 / 吨位 / 装甲推导默认能力（见下表）；
  两者都要求 `alwaysOn=false` 且开关打开才结算；
- 特殊单位：水滴 / 巨大剑鱼配置 `alwaysOn: true`，始终可撞，不受开关影响；
- 仿真场景可通过 `environment.enableRamming` 单独打开，跑完自动恢复；
- 开关关闭时普通战舰的 `ram` 配置仍然保留，只是不结算伤害，打开后立即生效，无需重开任务。
- 默认能力只覆盖战列舰 / 巡洋舰 / 驱逐舰 / 护卫舰，不进静态战力与费用（`ram.Derived=true`），
  避免打开体验开关就改变 AI 估值、图鉴战力和造价。

`config.GameSettings.EnableShipCollision`（`enableShipCollision`）是独立的舰体碰撞检查开关，默认 `true`：

- 只处理**不同阵营**水面舰船；友军仍不检测、不阻挡、不分离；
- 开启时接触会按吨位分离并掉速，关闭时只保留撞击伤害，舰体可以继续重叠；
- 与 `enableRamming` 互不影响，因此支持「只阻挡不伤害」「只伤害不阻挡」和「两者都开」三种组合。

| 舰种 | 伤害系数（× totalHP） | 自伤基数 | 冷却（秒） |
|---|---|---|---|
| battleship | 0.30 | 0.15 | 4 |
| cruiser | 0.24 | 0.22 | 4 |
| destroyer | 0.30 | 0.30 | 5 |
| frigate | 0.26 | 0.28 | 5 |

```text
armor = clamp(max(horizontalDamageReduction, verticalDamageReduction), 0, 0.9)
damage = totalHP × 类型系数 × (1 + 0.5 × armor)
selfDamageRatio = 类型自伤基数 × (1 - 0.35 × armor)
```

重甲同时提高撞击伤害、降低自伤；显式 `ram` 配置优先于默认值。

### 4.4 为什么不用 `guns.json5`

- 冲角没有「射界、装填、炮塔位置」这些概念；硬套会继续产生「从舰体中心发射」「射界挡住舰艏」等错误。
- 撞击伤害必须读取双方航速、吨位、接触点，`Gun.Fire` 的入参只有射手和 `Hurtable`，拿不到接触几何。
- 撞击消耗的是「接触机会」而不是弹药，冷却与 `ReloadTime` 语义不同。
- 图鉴、战力、AI 都需要把撞击当成独立能力统计，而不是一门 DPS 异常高的炮。

为什么不做全局 `rams.json5`：冲角数值与舰体吨位、减伤绑定，天然是单舰属性；目前只有少量舰船会用到，按仓库「不为一次性逻辑加抽象」的原则先内联。等出现 5 艘以上同型冲角舰、确实需要共享调参时，再把 `ram` 扩展成「profile 名 + 单舰覆盖」两种写法。

---

## 5. 运行时设计

### 5.1 Tick 顺序

在 `MissionManager.updateCombatPhase()` 的最前面插入接触结算，位于 `updateShipDiveStates()` 之后、`updateVision()` / 目标规划之前：

```text
advanceSimulationTick
 ├ updateInstructions
 ├ executeInstructions          // 舰船移动在这里
 ├ updateSupportPhase
 └ updateCombatPhase
     ├ updateShipDiveStates     // 先确定本拍水上 / 水下
     ├ updateShipContacts()     // ← 新增：接触检测 + 撞击伤害（+ P2 碰撞响应）
     ├ updateSubmarineDetection
     ├ updateVision
     ├ updateTargetPlanning
     ├ updateShipWeaponFire     // 本拍已被撞沉的舰船不会再开火
     ...
```

先结算撞击再开火，可以保证「撞沉即哑火」，也避免同一拍里撞击伤害和炮弹伤害的先后歧义。

### 5.2 接触检测

**宽相位**：复用 `pkg/mission/manager/spatial.go` 的 `combatIndex`。每拍在移动结束后重建一次，按舰船中心分桶；查询半径取「两舰半长之和 + 余量」，实际可用 `maxHullHitRadius`（3 格）的两倍，覆盖当前最长舰（263m ≈ 2.05 格）。遍历时只处理 `attacker.Uid < target.Uid` 的一侧，避免同一接触对重复检测；注意这只是**接触对去重**，撞击判定仍要按 A→B 和 B→A 各算一次（见 5.3）。

**窄相位**：在 `pkg/utils/geometry` 增加纯函数：

```go
// RotatedRectContact 返回两个旋转矩形的接触信息。
// 无接触时 ok=false；有接触时 depth 为最小平移深度（地图格），
// nx/ny 为从 b 指向 a 的单位法线，(px, py) 为近似接触点。
func RotatedRectContact(
    ax, ay, aLen, aWid, aRot float64,
    bx, by, bLen, bWid, bRot float64,
) (ok bool, depth, nx, ny, px, py float64)
```

实现用分离轴（SAT）：4 条边法线投影求最小重叠；接触点取「两矩形沿法线最近点的中点」。长度、宽度沿用现有的「地图格」口径（`Length / constants.MapBlockSize`），与炮弹命中判定保持一致。为它补表驱动单测：正撞、并排、T 字、角碰、明显分离。

**高速穿透**：`waterdrop` 1000 节在 2 倍速下单拍位移会超过自身舰宽，单点 OBB 检测会穿透。接触检测对快速舰船按本拍位移做子步 OBB 采样（步长不超过双方较窄舰宽），并用本拍起点作为舰艏判定中心，避免接触点落到当前舰艉；普通舰船不需要。

**过滤条件**（检测阶段就跳过，减少无效计算）：

- 任一方 `CurHP <= 0`；
- 任一方 `Submerged()`（水下潜艇从下方通过，不接触）；
- **同阵营直接跳过**：友军不检测、不阻挡、不分离，保持当前可互相穿行的行为；这样也不会让 AI 编队被友军碰撞堵死；
- 飞机只在 5.3 的 `canHitPlanes` 分支里处理，不参与舰-舰接触。

### 5.3 撞击判定

对每个接触对，只有同时满足以下条件才结算（普通战舰还要先通过 `enableRamming` 开关；`alwaysOn` 特殊单位跳过该检查）：

1. 该舰有有效撞击配置：显式 `ram.damage > 0`，或按舰种推导出的默认撞击；
2. 接触点是撞击舰的**舰艏区域**：接触点在撞击舰局部坐标的前 25% 舰长内（`localForward > halfLength * 0.5`，localForward 以舰艏为正）；
3. 撞击舰航向指向目标：`attacker→target` 方位与航向夹角 ≤ `ram.BowArc`；
4. 接近速度 ≥ `ram.MinSpeed`（见 5.4，按节计算）；
5. 冷却已好：`SimTick() - lastRamTick >= ReloadTicks(ram.Cooldown)`，冷却按「撞击舰 + 目标舰」成对记录，存在撞击舰的 `map[string]int64` 里（懒创建，规模等于它撞过的目标数）；
6. 目标必须是敌舰；友军在上一步已被整体跳过，所以不存在友军撞击，`FriendlyFire` 开关不改变这一点。

同一接触对要**双向**判定：A 的舰艏撞 B、B 的舰艏撞 A 是两次独立的撞击。正面对撞时双方都可能吃到对方的撞击伤害，再各自叠加自伤；按 UID 顺序结算，保证可复现。

**对空撞击（仅 `canHitPlanes`）**：对开启该标记的舰船，接触检测额外遍历敌机 OBB；舰艏朝向、接近速度、冷却都通过后，用 `m.hurtPlane` 结算，伤害取 `base`（飞机没有水平减伤），冷却按「舰 + 飞机」单独记录。飞机只吃这一条伤害分支，不参与舰-舰的碰撞响应。

**接触开始才结算（已实现）**：`BattleShip.ramContactTick` 记录每对「撞击舰 + 目标」最近一次检测到接触的拍号；只有接触刚开始的那一拍（上一拍该对没有接触）才允许结算。持续重叠、被碰撞响应推着摩擦、或同一拍内重复调用检测，都不会再掉血。冷却仍然保留，用于阻止「分开后立刻又贴上来」的连续结算；本次接触结束后，冷却结束且重新建立接触才会再次撞击。这样一次撞击对应一次可见的接触，而不是按冷却对重叠舰体持续掉血。

### 5.4 伤害公式

所有速度都用**节**（`CurSpeedKnots()` / `MaxSpeedKnots()` 同口径），不用内部「格/拍」。这样全局 2 倍速不会把撞击伤害翻倍，和图鉴、战报的速度口径一致。

```text
v1 = 撞击舰航向单位向量 × attacker.CurSpeedKnots()
v2 = 被撞舰航向单位向量 × target.CurSpeedKnots()
n  = 接触法线（从被撞舰指向撞击舰）
closing = (v1 - v2) · (-n)          // 接近速度，可为负

speedF = clamp(closing / attacker.MaxSpeedKnots(), 0, 1.5)²
angleF = clamp(cos(撞击舰航向与 attacker→target 方位夹角), 0, 1)^1.5

base = ram.Damage × speedF × angleF
目标伤害 = base × (1 - target.HorizontalDamageReduction)
自伤     = base × ram.SelfDamageRatio × clamp(target.Tonnage / attacker.Tonnage, 0.1, 1)
```

要点：

- **基础伤害按「满速正撞同吨位」定义**，设计者只需按 `totalHP` 的比例给初值（见下表），不用理解物理量纲。
- **用水平减伤**：撞击视为命中水线附近的舰体，与鱼雷、平射炮弹同一条减伤带；不参与穿甲开关（`PenetrationMM = 0`），不做暴击（撞击结果必须可复现）。
- **自伤按质量比缩放**：大舰撞小舰几乎无感，小舰撞大舰自伤接近满额。`Tonnage` 当前在初始化时等于 `TotalHP`，可直接当质量代理。
- `angleF` 在 60° 时约 0.35，擦碰明显弱于正撞；`speedF` 让低速蹭碰几乎无伤害。
- `closing` 为负（目标跑得更快）时不结算；`MaxSpeedKnots() <= 0` 时按 1 节兜底，避免除零。
- 任一方 `Tonnage <= 0`（测试船 / 未初始化）时质量比按 1 处理，不因缺字段产生 NaN。
- 冷却按游戏时间（`ReloadTicks`）计算，2 倍速下墙钟冷却减半，与火炮装填一致。

以下数值作为实现与仿真的基线（全速正撞、目标静止，因此 `speedF = 1`）：

| 单位 / 模板 | totalHP | 最大航速 | `ram.damage` | `cooldown` | `selfDamageRatio` | `minSpeed` | 同级目标单次伤害 | 自伤 |
|---|---|---|---|---|---|---|---|---|
| 巨大剑鱼 | 100000 | 50 | 300000 | 2 | 0 | 5 | Yamato：135000（186%） | 0 |
| 水滴 | 1 | 1000 | 10000000 | 2 | 0 | 10 | Yamato：4500000（击沉） | 0 |
| Monterey（1893） | 4084 | 13.6 | 1800 | 4 | 0.25 | 4 | 4000 HP / 减伤 0.24：1368（34%） | 450（11%） |
| 铁甲舰模板 | 10600 | 16.5 | 4800 | 4 | 0.25 | 4 | 10000 HP / 减伤 0.40：2880（29%） | 1200（11%） |
| 前无畏模板 | 15000 | 18 | 5250 | 4 | 0.20 | 4 | 15000 HP / 减伤 0.50：2625（17.5%） | 1050（7%） |

按舰种推导的默认值（未显式配置 `ram` 的普通战舰，仅在开关打开时生效）：

| 舰船 | 舰种 | totalHP | 装甲（max H/V） | 推导 damage | 自伤比 | 对同级目标 |
|---|---|---|---|---|---|---|
| Yamato | battleship | 72500 | 0.55 | 27731 | 0.121 | 12479（17.2%） |
| Bismarck | battleship | 52600 | 0.55 | 20120 | 0.121 | 9054（17.2%） |
| Baltimore | cruiser | 17031 | 0.35 | 4803 | 0.193 | 3122（18.3%） |
| Johnston | destroyer | 2500 | 0.00 | 750 | 0.300 | 750（30.0%） |

重甲示例：同为 10000 HP 的战列舰，装甲 0.1 → 0.8 时 `damage` 从 3150 提高到 4200（+33%），自伤比从 0.145 降到 0.108（-25%）。

定值理由：

- 历史冲角舰按 `ram.damage ≈ 0.45 × totalHP`，同级目标挨一次约掉 25%~30% 血：两到三次撞击能决出胜负，但不会一次秒杀，给被撞方转向/撤退窗口。
- Monterey 略高（0.44 × totalHP），因为它是目前唯一的 19 世纪实船，作为冲角玩法试点需要一击有存在感；如果实测过强，先降到 1500。
- 前无畏按 0.35 × totalHP，撞同级约 17.5%，撞驱逐舰仍然毁灭性；符合「保留冲角但不再作为主战术」的历史定位。
- 剑鱼保持 300000，对 Yamato 一发 135000（击沉）；水滴提高到 10000000，对 Yamato 4.5M、对 500 万 HP 翻车鱼 10M，都是整舰动能命中，维持「宇宙探测器」的压迫感；两者自伤为 0。
- 驱逐舰 / 巡洋舰现在通过舰种默认值获得撞击（驱逐舰 0.30 × HP、5s 冷却、自伤 0.30），不再需要逐艘配置；显式 `ram` 仍可覆盖。

不同接近速度下的实际结果（用于判断手感）：

| 场景 | `ram.damage` | `speedF` | 目标减伤 | 目标伤害 | 自伤 |
|---|---|---|---|---|---|
| Monterey 13 节撞 4000 HP 同级舰 | 1800 | (13/13.6)² = 0.914 | 0.24 | ≈ 1250（31%） | ≈ 403（10%） |
| 铁甲舰模板 14 节撞 10000 HP 同级舰 | 4800 | (14/16.5)² = 0.72 | 0.40 | ≈ 2074（21%） | ≈ 864（8%） |
| 巨大剑鱼 40 节撞 Yamato | 300000 | (40/50)² = 0.64 | 0.55 | ≈ 86400（击沉） | 0 |

对飞机：`目标伤害 = base`（飞机没有水平减伤），仍乘 `speedF × angleF`；水滴的 10000000 对任何飞机都是一击。实现时需要在 `pkg/mission/object/unit/plane.go` 的 `HurtBy` 里对 `TypeRam` 跳过「舰对空 60% 单发上限」，否则会被误判成普通舰炮命中。

### 5.5 结算链路

为了不重写战报、观察者、伤害数字、命中特效，撞击事件复用现有的弹药结算载体：

1. 新建 `objBullet.TypeRam`，但**不进入 `ForwardingBullets`**，不飞行、不受寿命影响；
2. 撞击时构造一枚「撞击弹」：`Type=TypeRam`、`Name="Ram"`、`WeaponName="Ram"`、`Damage=base`、`Plunge=0`、`CriticalRate=0`、`PenetrationMM=0`、`CurPos=接触点`、`Rotation=撞击舰航向`、`Shooter`/`ShooterName`/`ShooterType`/`BelongPlayer`/`TargetUID` 齐全；
3. 舰-舰调用 `m.hurtShip(target, ramBullet)`，舰-机调用 `m.hurtPlane(plane, ramBullet)`，沿用现有伤害、`DamageState`、`LastHit`、战报、观察者；注意 `Plane.HurtBy` 目前把「舰对空」单发伤害封顶在飞机最大 HP 的 60%，撞击要放行 `TypeRam`，否则水滴撞机只能打掉 60% 而不是一击必杀；
4. `RealDamage` 写回后计入 `Stats.RecordDamage`；
5. 把 `updateShotBullets` 尾部「伤害数字 + 命中特效」抽成可复用函数（行为不变），撞击复用它；
6. 自伤单独结算：扣 `attacker.CurHP`、累计 `DamageTaken`、记录 `LastHit`，并写入一条 `BattleLogShipSelfHit` 战况（攻击者与目标都是撞击舰，伤害为反作用值）；若自伤导致沉没，按普通战沉记录，不把击杀归给被撞方。

> 已实现：`pkg/mission/manager/collision.go` 构造 `Bullet{Type: TypeRam}` 并调用 `hurtShip` / `hurtPlane`，`hitEffect` 用专用大闪光，音频走 `PlayRamImpact`。
>
> 取舍：用 `Bullet` 当「撞击事件载体」是刻意复用，不是把它当炮弹。好处是战报 / 观察者 / 特效 / 统计 / 击沉归因全部复用，改动面最小。若后续 P2 想要更干净的领域模型，再把 `Bullet → Impact` 抽成独立事件类型，`ram` 的数据契约不受影响。

`bulletHitInfo` 需要把 `TypeRam` 映射为武器类型 `ram`，战报新增「撞击」文案；`hitEffect` 为 `TypeRam` 返回大型火花（`KindFlash`）而不是小口径弹丸效果。

### 5.6 表现

- **特效**：接触点生成一次大型命中闪光；若接触点靠近水线可叠加水花。复用 `objExplosion.NewHitEffect`，不新增爆炸类型。
- **伤害数字**：沿用 `objMark.NewText`，字号按「本次伤害 / 目标最大 HP」分档（例如 <5% 白、5%~15% 黄、>15% 红），避免大数值直接刷屏。
- **音效**：`pkg/audio/player/weapon.go` 增加 `PlayRamImpact()`，带冷却与镜头内判断，避免多舰同时接触时叠音。
- **战报 / 观察者**：`HitInfo.WeaponType = "ram"`；i18n 增加 `MsgBattleReportWeaponRam`（中 / 英 / 日 / 俄），文案如「撞击」。
- **图鉴**：战力贡献使用稳定 ID `"Ram"`，在四个 `references.*.json5` 中增加 `Ram` 条目（显示名：撞击 / Ramming / 体当たり / Таран），避免贡献列表出现未翻译的英文。

---

## 6. 碰撞系统（已实现）

P2（位置分离 + 掉速）在同一个接触检测结果上做物理响应，当前只实现**不同阵营水面舰**，
友军仍保持穿行。规则如下：

| 接触对 | 检测 | 伤害 | 响应 |
|---|---|---|---|
| 敌方水面舰 × 水面舰 | 是 | 满足 5.3 时撞击 | 分离 + 掉速（已实现） |
| 友军水面舰 × 水面舰 | **否** | 无 | 无（直接穿过，保持现状） |
| 水面舰 × 水下潜艇 | 否 | 无 | 无（从上方通过） |
| 水下潜艇 × 水下潜艇 | 可选 | 无 | 暂不阻挡 |
| `canHitPlanes` 舰船 × 敌机 | 是 | 撞击 | 无（飞机继续飞） |
| 其他舰船 × 飞机 | 否 | 无 | 飞机飞越 |
| 舰船 × 陆地 | 现有 `MoveTo` 逻辑 | 无 | 停船 / 搁浅 |
| `waterdrop`（`CanOnLand`）× 舰船 | 是 | 撞击 | 可穿过陆地，但不穿过舰船 |

响应细节（`pkg/mission/manager/collision.go`）：

1. 先按接触法线用 SAT 最小平移向量分离两舰，位移按质量代理（优先 `Tonnage`，缺失时用 `TotalHP`）反比分配，大舰几乎不动；单舰单拍位移上限 `maxContactSeparation = 0.5` 格，深穿透分几拍推开，不会瞬移；同一艘舰每拍最多参与一次分离；
2. 分离后的位置做边界与陆地检查（`HullLandOverlapSamples`）：会把舰体推上岸时只放弃该舰的这次位移，另一艘仍可被推开；边界夹取若把位移方向改反则同样放弃；
3. 掉速只在**接触开始那一拍**施加：把航速朝接触法线方向的分量按 80% 吃掉，正撞保留 `contactSpeedKeep = 0.2`，擦碰只掉少量航速；持续重叠不会逐拍把航速乘到 0；
4. 撞击伤害与掉速共用同一次接触开始判定：先按撞击瞬间的航速结算伤害，再掉速 / 分离，避免正撞先把自己减到低于 `minSpeed` 而漏伤害；
5. 友军不参与：不检测、不阻挡、不分离，避免 AI 编队互相卡死；
6. 碰撞响应由 `enableShipCollision` 控制，不受 `enableRamming` 影响：关闭撞击伤害时敌舰仍会互相阻挡，只是不结算伤害；关闭碰撞检查时只结算伤害，舰体可继续重叠；
7. 连续碰撞（高速、2 倍速）沿用「线段扫掠 + 单拍位置修正」，不引入物理引擎；
8. 碰撞位移同步平移尾流采样锚点，避免把推挤画成自航尾迹。

### 6.1 性能影响（原型实测）

用临时原型（中心分桶 + OBB SAT + 舰艏粗判，每拍新建 map）在 `PearlHarbor1941` 上实测，环境 Apple M1 Pro / Go 1.26.2，headless：

| 场景 | 舰数 | 检测 p50 | 检测 p99 | 同段 tick p50 | 占 16.7ms 帧 |
|---|---|---|---|---|---|
| 珍珠港任务（稀疏） | 57 | 10.4µs | 13.1µs | 250µs | 0.06% |
| 合成密集，cell=2 | 40 | 9.3µs | 15.5µs | — | 0.06% |
| 合成密集，cell=2 | 80 | 18.0µs | 23.9µs | — | 0.11% |
| 合成密集，cell=2 | 120 | 29.2µs | 35.0µs | — | 0.17% |
| 合成密集，cell=2 | 200 | 78.3µs | 97.8µs | — | 0.47% |

方法：任务先跑 2400 拍进入交战节奏，再连续测 300 拍的 tick；检测在同一批舰船位置上重复 500 次取分位。合成场景为 40~200 艘随机分布在 50×50 格、半敌半友、随机航向，舰长 80~260m。原型每拍新建 map，未做复用；P1 实现沿用了这一做法（2 格网格 + 每拍 map），因此上表可视为上界。

对照：如果把 `combatBuckets` 的 8 格直接拿来用，200 艘密集场景 p50 会到 300µs（1.8% 帧），所以接触检测应使用独立的 2~3 格网格，而不是复用炮弹的 8 格桶。

结论：

- **当前任务规模（约 60 艘）碰撞检测每拍约 0.01ms，占 16.7ms 帧预算 0.06%**；即使 200 艘密集同场，p99 也不到 0.1ms（约 0.6% 帧）。
- 碰撞响应只在真正接触的少数对（实测 0~11 对/拍）上做位置修正，量级在 1µs 以内；撞击伤害、伤害数字和特效只在撞击发生的那一拍触发，次数远低于炮弹。
- 因此「引入碰撞」本身不会造成可感知的帧率下降。真正的风险是实现时每拍分配 map / slice 触发 GC：生产代码应复用 scratch 桶和候选切片，实测值还能再降。
- 200 艘全部敌对时，候选对大致翻倍，检测约 0.15ms（<1% 帧），仍在安全范围。
- 唯一需要留意的规模是「数百艘舰船挤在几十格内」的极端场景；按当前任务设计不会出现，若出现再考虑把接触网格与 `combatBuckets` 合并、或给密集区加降频。

---

## 7. 战力、费用与 AI

### 7.1 战力

`pkg/mission/object/combatpower/combatpower.go` 增加撞击贡献：

```text
期望系数 = 0.5                    // 接触机会 × 角度 × 速度的综合折减
AntiShipDPS += ram.Damage × 期望系数 / max(ram.Cooldown, 1)
BurstDamage += ram.Damage × 期望系数
if ram.CanHitPlanes {
    AntiAirDPS += ram.Damage × 期望系数 / max(ram.Cooldown, 1)
}
MaxProjectionRange = max(现有值, 舰长 / MapBlockSize)   // 接触距离，约半舰长
```

- 撞击是接触武器，`Projection` 必然很低；这是符合事实的，不应为了好看虚增。
- 撞击舰不能出现 `AntiShip = 0`，否则 AI 会把它当无武装单位；水滴靠 `CanHitPlanes` 同时保留 `AntiAir > 0`。
- `Weapon.MaxToShipRange` 在初始化时至少设为「舰长 / MapBlockSize + 0.5 格」，让 `shipFireTargets` 能在贴近时选中目标；撞击本身不依赖这个射程。
- 图鉴贡献名用 `"Ram"`，与 5.6 的引用条目对应。
- `ram.Derived=true` 的舰种默认撞击**不进入**战力与费用；只有显式 `ram`（Monterey、水滴、巨大剑鱼）按上式计算。
- 因此打开体验开关不会改变 AI 的舰队估值、图鉴战力或造价；AI 仍按原有火力/生存指标编组，只是在接触时多了一次撞击结算。

### 7.2 费用

按现有 `fundsCost` / `timeCost` 评估流程重算：撞击的 DPS 与爆发进入战力后，参考 `jutland-evaluate-ship-cost` 的方法重新给 `swordfish` / `waterdrop` 定价；早期冲角舰在加入时一并评估。不要把「一次 10000000」直接当 DPS 计价。

### 7.3 AI

- 撞击舰的目标选择沿用现有逻辑：`MaxToShipRange` 覆盖接触距离后，AI 会照常选敌并沿航路压上；接触即结算，不需要新的「撞击指令」。
- 电脑舰队编组把撞击舰当战斗舰；`CombatPower.AntiShip > 0` 保证它不会被当成货轮。
- 撤退判定不变；撞击舰残血时仍会撤退。
- 友军之间不检测碰撞，编队与现有行为一致，不需要额外的最小间距。
- 玩家侧：不新增按键；选中舰船后右键移动即可撞击。是否增加「撞击」显式指令（指定目标、强制全速）留到 P3 评估。

### 7.4 图鉴与文案

- 舰船 `armaments` 的「攻击方式：舰首撞击（仅对舰）」保留；
- 新增 `Ram` 武器条目（四个语言文件），描述区分「冲角舰」与「特殊单位」；
- 删除 `Impact` / `SwordfishImpact` 武器条目（四个语言文件同步删除，保持 `name` 集合一致）。

---

## 8. 迁移现有单位（已完成）

| 单位 | 迁移前 | 迁移后 |
|---|---|---|
| `waterdrop` | 主炮 `Impact`（可对空） | `ram`：`damage=10000000`、`cooldown=2`、`selfDamageRatio=0`、`canHitPlanes=true`；**保留对空**（撞机） |
| `swordfish` | 主炮 `SwordfishImpact`（仅对舰） | `ram`：`damage=300000`、`cooldown=2`、`selfDamageRatio=0`；仍仅对舰 |

配套改动：

- `configs/ships.json5` 删除两舰的 `mainGuns` 假炮，加 `ram`；
- `configs/guns.json5` 删除 `Impact`、`SwordfishImpact`；
- `configs/references*.json5` 删除两条武器引用，新增 `Ram`；
- `pkg/mission/object/combatpower/config_regression_test.go` 的 `TestWaterDropAndSwordfishRamWeaponsDifferInAntiAir` 改为验证 `ram` 配置：水滴保留对空战力、剑鱼对空为 0；
- 图鉴描述里「撞击武器」的说法改为「冲角 / 撞击」。

**水滴对空的处理**：用 `ram.canHitPlanes` 保留。舰体 OBB 与敌机 OBB 接触、舰艏朝向和接近速度达标时，按同一公式结算；飞机没有水平减伤，10000000 伤害等于一击必杀（需放行 `Plane.HurtBy` 的 60% 舰对空上限，见 §5.5）。该能力只给水滴，不推广到常规舰船。

---

## 9. 分阶段实施

### P0：数据与纯函数（已完成）

- 新增 `RamConfig`、`Normalize`、`BattleShip.Ram` 字段；
- 新增 `damage.ResolveRam(...)` 纯函数（入参：双方航速、吨位、减伤、配置；出参：目标伤害、自伤、接近速度）；
- 新增 `geometry.RotatedRectContact`；
- 补单测（四段式注释）：SAT 五种几何、公式表驱动、边界（0/负速度、NaN、减伤 1、吨位 0）；
- `configs/Readme.md` 增加 `ram` 字段说明。

验收：`go test ./pkg/utils/geometry ./pkg/mission/object/damage ./pkg/mission/object/unit` 通过；游戏行为不变。

### P1：接触 + 撞击伤害（已完成）

- `MissionManager.updateShipContacts()`：宽相位 + 窄相位 + 5.3 判定 + 5.4 伤害 + 冷却；
- `objBullet.TypeRam`、`bulletHitInfo`、战报 i18n、命中特效、音效；
- 抽出可复用的「到达弹药表现」函数；
- 迁移 `swordfish` / `waterdrop`；
- 管理器 headless 测试：两舰对开 → 只结算一次、冷却内不重复、友军完全不检测、潜艇不接触、水滴撞机可一击击落（不被 60% 上限截断）；
- 固定 `damage` 数值先跑 F1 `duel` 场景校准。

验收：两舰相撞有且只有一次伤害；战报显示「撞击」；伤害数字/特效/音效正常；撞击舰的 `CombatPower.AntiShip > 0`，水滴 `AntiAir > 0`、剑鱼 `AntiAir == 0`；`go test ./pkg/mission/...` 通过。

### P2：碰撞响应（敌军分离已实现）

- 已实现：敌军水面舰位置分离、陆地回退、单拍位移上限、按吨位分配、接触开始掉速、尾流锚点同步；
- 未实现：友军分离、连续碰撞的完整修正；
- 独立开关 `enableShipCollision` 默认开启，与 `enableRamming` 互不影响；
- 性能：按 6.1 的原型基线，接触检测占单帧 <0.5%；开关打开后需复测 F3 headless p99。

验收：舰船不再互相穿过；不会把舰船推上岸；撞击后能脱离；F3 宏观场景无卡死。

### P3：早期军舰与 AI 调优

- `monterey_m_6` 已配置冲角，设置页打开后即可实测；
- 战列舰 / 巡洋舰 / 驱逐舰 / 护卫舰已通过舰种默认值获得撞击，不再逐艘写 `ram`；需要特殊数值时仍可显式覆盖；
- 费用重估、图鉴、AI 编组与撤退复测；
- README 增加本设计文档入口。

验收：至少一艘真实历史冲角舰两次撞击可击沉同级舰；AI 不会因为撞击舰卡住；费用与战力排名合理。

### P4：打磨（可选）

- 撞击镜头震动、进水 / 减速 debuff、断裂特效；
- 常规舰船撞机、撞岸工事、接舷；
- `Bullet → Impact` 领域模型重构。

---

## 10. 验证

### 单元测试

实现后已执行 `go test ./...`（全绿）与 `go build ./...`（通过）。关键用例：

- `TestRotatedRectContact`：正撞 / 并排 / T 字 / 45° / 分离；
- `TestResolveRamScalesWithSpeedAngleAndMass`：速度平方、60° 角、目标免疫、质量比、NaN/零速度；
- `TestRamConfigNormalizeDefaultsAndClamps`：默认值、负伤害、非法角度与冷却；
- `TestRamCooldownIsPerTarget`：冷却按目标隔离与 `ReloadTicks` 边界；
- `TestRamSettingOnlyGatesNormalShips`：普通舰默认关闭、开启后生效、`alwaysOn` 不受开关影响；
- `TestRamCooldownBlocksRepeatedContact`：持续重叠只结算一次，过冷却再结算；
- `TestRamSkipsFriendlyAndSubmergedShips`：友军与水下潜艇不参与；
- `TestSpecialShipRamsPlaneThroughWeaponCap`：水滴撞机绕过 60% 舰对空上限，一击击落。

### 仿真

- F0：`damage.ResolveRam` 已知答案表；
- F1：新增 `duel-ram` 场景（两艘冲角舰对开），检查首撞时机、单次伤害、自伤、TTK；
- F2：确认普通舰船不撞机、水滴可撞机、潜艇水下不被撞；
- F3：`macro` 场景里加一艘撞击舰，确认 AI 能接敌且不阻塞编队；
- 性能：headless p50/p95/p99 与当前基线对比。

### 手动验收

- 打开测试分类的「撞击测试场」（`RamTest`）：P1 四类普通战舰 + Monterey 对 P2 四种靶舰 + 翻车鱼；
  先在设置里打开撞击开关，再全速撞击，观察不同舰种/装甲的单次伤害与自伤；
- 开窗口，让 Monterey 全速撞驱逐舰：有撞击音效、闪光、伤害数字、战报「撞击」；
- 2 倍速下重复一次，伤害数值不变（只有发生频率变快）；
- 停船贴住敌舰：不结算伤害；
- 友军相撞：直接穿过，不掉血、不阻挡（保持现状）；
- 图鉴中撞击贡献显示为「撞击 / Ramming / 体当たり / Таран」；
- 水滴撞敌机：有闪光与伤害数字，飞机被一击击落。

---

## 11. 风险与后续问题

1. **一击必杀的手感**：剑鱼 / 水滴改成 `ram` 后，99999×3 的暴击波动消失，伤害变为确定值（剑鱼 300000、水滴 10000000）；对 Yamato 仍是击沉，但需要确认战报与伤害数字不会显得「过于干净」。
2. **HP = 吨位代理**：`Tonnage` 当前等于 `TotalHP`，不是真实吨位；用它做质量比在数值上自洽，但语义上要写清楚，避免以后接入真实吨位时公式漂移。
3. **多次接触 / 多舰同时撞击**：同一目标被多舰同拍撞击时，按 UID 顺序结算，后撞的可能只吃到溢出伤害；战报与击沉归因要测。
4. **水下 / 水面边界**：下潜过渡中（`Diving` / `Surfacing`）按水面处理，还是允许被撞？当前建议按水面处理，等 P2 再定。
5. **撞机的表现**：水滴撞飞机时，飞机有高度和翻滚，舰体 OBB 与飞机 OBB 的接触点可能看起来「差一点」；只给特殊单位开启，优先保证判定一致，表现后续再调。
6. **配重与造价**：撞击战力系数 `0.5` 是起点，需用 F1 对射与 F3 编队校准；否则 AI 会高估或低估撞击舰。
7. **模组 / 旧配置**：`ram` 是可选字段，旧配置不受影响；删除 `Impact` / `SwordfishImpact` 会让引用它们的自定义配置失效，需要在 `configs/Readme.md` 标注。
8. **性能**：原型实测当前规模约 0.01ms/拍（0.06% 帧），200 艘密集约 0.1ms p99（0.6% 帧），不构成帧率风险；但实现若每拍新建 map / 切片会引入 GC，必须复用 scratch 桶。
9. **碰撞响应的寻路副作用**：敌方舰船互相阻挡后，可能把对方顶上岸或卡在狭窄水道；位置分离必须做陆地回退，并在 F3 场景里检查「撞完能否脱离」。
10. **数值定稿后的平衡**：历史冲角舰两次撞击可击沉同级舰，可能让 1890 年代对局过度偏向撞击；先按表中数值进 F1，若 TTK 过短再把 `ram.damage` 降到 0.30~0.35 × totalHP。
11. **默认撞击与 AI 估值脱节**：`Derived` 撞击不进战力/费用是有意为之，避免体验开关改变整局平衡；代价是 AI 不会因为能撞而主动贴身。若以后要让 AI 主动撞击，必须单独加行为开关，不能复用静态战力。

---

## 12. 代码索引

| 位置 | 作用 |
|---|---|
| `pkg/config/config.go` / `configs/game_settings.json5` / `pkg/game/settings/ui.go` | `enableRamming`（撞击伤害）与 `enableShipCollision`（舰体碰撞检查）两个独立开关，`enableShipCollision` 默认开启、`enableRamming` 默认关闭 |
| `pkg/mission/object/unit/ship.go` | `BattleShip` 增加 `Ram` 字段与撞击冷却状态 |
| `pkg/mission/object/unit/ram.go`（新） | `RamConfig`、默认值、撞击能力查询 |
| `pkg/mission/object/damage/ram.go`（新） | 纯伤害公式，供 F0 与运行时共用 |
| `pkg/utils/geometry/geometry.go` | `RotatedRectContact`（SAT） |
| `pkg/mission/manager/collision.go`（新） | 接触检测、撞击判定、冷却、P2 碰撞响应 |
| `pkg/mission/manager/manager.go` | `updateCombatPhase` 插入接触结算 |
| `pkg/mission/sim/environment.go` / `pkg/mission/sim/scenario.go` | 场景级 `enableRamming` 覆盖与恢复 |
| `pkg/mission/manager/battle_log.go` | 撞击命中信息、自伤记录 |
| `pkg/mission/manager/combat.go` | 复用伤害数字 / 特效逻辑 |
| `pkg/mission/object/bullet/bullet.go` | `TypeRam` 事件载体 |
| `pkg/mission/object/unit/plane.go` | `HurtBy` 对 `TypeRam` 放行舰对空 60% 单发上限 |
| `pkg/mission/object/combatpower/combatpower.go` | 撞击战力贡献 |
| `pkg/mission/object/initialize/initialize.go` | 加载时 `Normalize` / 校验 |
| `pkg/audio/player/weapon.go` | `PlayRamImpact` |
| `pkg/i18n/messages.go` + `pkg/i18n/locales/*.toml` | 战报「撞击」文案 |
| `configs/ships.json5` / `configs/guns.json5` / `configs/references*.json5` | 迁移与数据 |
| `configs/Readme.md` | `ram` 字段文档 |
