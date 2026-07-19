# 配置文件说明

## 费用与耗时评估

飞机与舰船的 `fundsCost` / `timeCost` 不建议手工长期维护。新增或重平衡单位后，优先使用仓库内的 Codex skill 统一评估，避免同类单位价格口径不一致。

### 飞机费用（planes.json5）

使用 skill：`jutland-evaluate-plane-cost`

目标文件：`configs/planes.json5`

主流程优先使用运行时战力数据：

```bash
bash .codex/skills/jutland-evaluate-plane-cost/scripts/evaluate_plane_costs.sh
```

如果当前环境因 Ebiten 图形初始化等问题无法运行 Go 测试导出，可使用备用配置估算器：

```bash
python3 .codex/skills/jutland-evaluate-plane-cost/scripts/plane_cost_calc.py configs/planes.json5
python3 .codex/skills/jutland-evaluate-plane-cost/scripts/plane_cost_calc.py configs/planes.json5 --apply
```

当前飞机费用公式：

```text
rawFunds  = combatPower * typeMultiplier * scaleFactor
fundsCost = clamp(round(rawFunds), 3, 30)
timeCost  = clamp(round(fundsCost * 0.35 + 2), 3, 10)
```

费用按整数写回，不再做 `$5` 粗粒度分档。备用估算器使用 `fallbackScaleFactor = 0.30`，其 `cpEstimate` 仅用于无法取得 Go 图鉴战力时的替代评估。

### 舰船费用（ships.json5）

使用 skill：`jutland-evaluate-ship-cost`

目标文件：`configs/ships.json5`、`configs/guns.json5`、`configs/torpedo_launchers.json5`、`configs/rocket_launchers.json5`，并依赖 `configs/bullets.json5`、`configs/planes.json5` 与初始化后的运行时战力。

查看建议费用：

```bash
bash .codex/skills/jutland-evaluate-ship-cost/scripts/evaluate_ship_costs.sh
```

确认后写回：

```bash
python3 .codex/skills/jutland-evaluate-ship-cost/scripts/evaluate_weapon_costs.py --apply
python3 .codex/skills/jutland-evaluate-ship-cost/scripts/apply_ship_costs.py
```

武器参考价同时考虑持续输出与首轮爆发：

```text
expectedDamage = damage * (1 + 2.7 * criticalRate)
salvoDamage    = expectedDamage * projectileCount
referenceScore = salvoDamage * (0.70 / actualCycle + 0.30 / referenceCycle)
               * weaponTypeFactor
weaponCost     = clamp(roundTo5(referenceScore / 5), 1, 100)
```

`referenceScore` 只用于比较武器强弱，除以 `5` 后才映射为游戏资金。标准武器统一计算；没有阵营路径的彩蛋武器保留手工价格，但同样限制在 `$1–100`。武器 `fundsCost` 当前不会在运行时被单独购买。

舰船价格以 `combatpower.CalculateShip` 的运行时结果为主，并用有效生命值防止无武装或低输出舰体被低估：

```text
economicPower = HullPower + 0.25 * Burst + 0.10 * Projection
combatCost    = roundTo5(typeBaseCost + economicPower * typePowerFactor)
hullFloor     = roundTo5(EHP^0.45 * hullFloorMultiplier * 3.0)
fundsCost     = max(5, combatCost, hullFloor)
```

航母和航空战列舰的 `Burst` / `Projection` 已混入航空贡献，因此舰体定价只使用 `HullPower`，舰载机资金仍在运行时全额追加：

```text
aircraftCost = sum(planeCount * planeFundsCost)
totalCost    = fundsCost + aircraftCost
```

医疗船的常规战力为零，使用 `roundTo5(hullFloor + 25)`，其中 `$25` 表示治疗设施和支援价值。`nation == special` 的彩蛋舰船保留手工价格。

少数科技树终局舰可以在用户确认后使用显式战略倍率。当前 `satsuma=1.10`，用于让其价格贴近大和；`edo=1.15`，用于确保江户明显高于大和。此类例外只解决科技树层级，不应改动全局战列舰系数。

舰载机不线性增加舰船建造时间；舰船和飞机生产视为可并行，只增加小额飞行队适配时间：

```text
airWingFitPenalty = clamp(
    round(sqrt(aircraftCount) * avgPlaneTime * 0.12 + (aircraftTypes - 1) * 1.5),
    0,
    18,
)

普通舰种：
baseTime = clamp(round(2 + fundsCost * 0.35), 3, 130)

战列舰：
baseTime = clamp(round(40 + fundsCost * 0.13), 85, 180)

timeCost = clamp(
    baseTime + airWingFitPenalty * nationAirFitMultiplier,
    3,
    typeTimeMax,
)
```

当前适配时间国家倍率：`us=0.75`、`uk=0.90`、`jp/de=1.00`、`ru/su=1.05`、`cn=0.50`。

### 验证

写回费用后至少运行：

```bash
make build
```

若只改费用脚本，可额外运行 Python 语法检查：

```bash
PYTHONPYCACHEPREFIX=/tmp/jutland_pycache python3 -m py_compile \
  .codex/skills/jutland-evaluate-plane-cost/scripts/plane_cost_calc.py \
  .codex/skills/jutland-evaluate-ship-cost/scripts/evaluate_weapon_costs.py \
  .codex/skills/jutland-evaluate-ship-cost/scripts/evaluate_ship_costs.py \
  .codex/skills/jutland-evaluate-ship-cost/scripts/apply_ship_costs.py
```

## 弹药配置（bullets.json5）

```json5
[
  {
    // 弹药名称（不可重复）
    // US 表示所属阵营
    // GB (gun bullet) 表示是火炮弹药
    // TB (torpedo bullet) 表示鱼雷弹药
    // RB (rocket bullet) 表示火箭弹
    // 127 表示口径 127mm
    // 1932 表示 1932 年研制（不一定准确，主要是做区分）
    name: "US/GB/127/1932",
    // 弹药类型
    // shell 炮弹
    // torpedo 鱼雷
    // bomb 炸弹
    // rocket 火箭弹
    // laser 镭射
    type: "shell",
    // 口径
    diameter: 305,
    // 弹药命中伤害值
    // 推荐公式：现实炮弹重量（kg) * 3
    damage: 100,
    // 暴击率：造成暴击伤害的几率（击中弹药库之类）
    // 注：超级暴击率固定为暴击率的 1/10
    criticalRate: 0.002
  }
]
```

## 舰炮配置（guns.json5）

```json5
[
  {
    // 舰炮名称（不可重复）
    // US 是所属阵营，127 是口径，38 是倍径，MK45 是型号（可选）
    // 注：如果是多管炮，则追加炮管数量，如：UK/152/50/2
    name: "US/127/38/MK45",
    // 弹药名称（需确保一定存在）
    bulletName: "US/GB/127/1932",
    // 炮管数量
    bulletCount: 1,
    // 装填时间（单位：秒）
    // 推荐值：现实装填速度 / 2
    reloadTime: 1,
    // 射程（地图格数）
    // 推荐值：现实射程（km）
    range: 20,
    // 散布半径
    bulletSpread: 50,
    // 炮弹速度
    // 推荐值：1km/s -> 1000
    bulletSpeed: 1000,
    // 是否对舰
    antiShip: true,
    // 是否对空
    antiAircraft: true
  }
]
```

## 图鉴引用配置（references.json5 / references.LOCALE.json5）

图鉴引用按语言拆分为四个同结构文件：`references.json5` 为简体中文，
`references.en.json5`、`references.ru.json5`、`references.ja.json5` 分别为英文、俄文和日文。
四个文件必须保持完全一致的 `name` 集合和武装条目数量；
`name`、武器数量、口径、型号及素材作者署名属于稳定数据，不应在翻译时改变。
运行时会按“当前语言 → 英文 → 简体中文”的顺序读取对应条目。

```json5
[
  {
    // 对象名称，通常与 ships.json5 中的 name 对应
    name: "duck",
    // 图鉴使用的精细舰种名称；缩写、排水量、速度、费用和减伤从 ships.json5 读取
    type: "特殊",
    // 图鉴武装摘要
    armaments: [
      {label: "主炮", value: "1x8 406mm"},
    ],
    // 历史与来源描述。这里使用单个字符串，长文案由 UI 自动换行。
    description: "小黄鸭是一种利用塑胶或者聚氯乙烯制成雏鸭形状的玩具，中空而且质料轻软。",
    // 素材原作者
    author: "未知",
    // 参考链接
    links: [
      {
        name: "百度百科 - 小黄鸭",
        url: "https://baike.baidu.com/item/%E5%B0%8F%E9%BB%84%E9%B8%AD/3214424"
      }
    ]
  }
]
```

## 鱼雷发射器配置（torpedo_launchers.json5）

```json5
[
  {
    // 鱼雷发射器名称
    // US 是所属阵营，533 表示口径，4 表示管数
    name: "US/533/4",
    // 鱼雷名称
    bulletName: "US/TB/533/1931",
    // 发射器管数
    bulletCount: 4,
    // 发射间隔（单位：秒）
    //（建议至少为 1）
    shotInterval: 1,
    // 装填时间（单位：秒）
    // 推荐值：不要少于 35，否则很破坏平衡
    reloadTime: 50,
    // 射程（地图格数）
    // 推荐值：现实射程（km）
    range: 12,
    // 鱼雷速度
    // 推荐值：实际速度（节）如 45 节 -> 45
    bulletSpeed: 45,
  },
]
```

## 火箭发射器配置（rocket_launchers.json5）

```json5
[
  {
    // 火箭发射器名称
    name: "JP/120/21/Type5/AA",
    // 火箭弹名称
    bulletName: "JP/RB/120/1944",
    // 单轮装填火箭弹数量
    rocketCount: 21,
    // 分组数量，21 发 / 3 组即每组 7 发
    groupCount: 3,
    // 组内单发发射间隔（单位：秒）
    shotInterval: 0.06,
    // 分组发射间隔（单位：秒）
    groupInterval: 0.35,
    // 装填时间（单位：秒）
    reloadTime: 45,
    // 射程（地图格数）
    range: 12,
    // 火箭弹散布
    bulletSpread: 140,
    // 火箭弹速度
    bulletSpeed: 700,
    // 是否具备反舰能力
    antiShip: false,
    // 是否具备防空能力
    antiAircraft: true,
    // 近炸触发半径（地图格数）
    proximityRadius: 0.25,
    // 爆炸伤害半径（地图格数）
    blastRadius: 0.45,
  },
]
```

## 飞机火箭发射器配置（plane_rocket_launchers.json5）

```json5
[
  {
    // 飞机火箭发射器名称
    // 名称建议包含总备弹数量，如 127/4 表示 127mm / 4 发挂载
    name: "US/AIR/RB/127/4",
    // 火箭弹名称
    bulletName: "US/RB/127/1937",
    // 挂载总数；飞机火箭弹不会在空中重装
    rocketCount: 4,
    // 单发发射间隔（单位：秒）
    shotInterval: 0.18,
    // 射程（地图格数）
    range: 8,
    // 火箭弹散布
    bulletSpread: 140,
    // 火箭弹速度
    bulletSpeed: 700,
    // 是否具备反舰能力
    antiShip: true,
    // 是否具备防空能力；第一版实际目标仍由载机类型决定
    antiAircraft: false,
    // 近炸触发半径；实际只有对空目标会进入近炸分支
    proximityRadius: 0.22,
    // 爆炸伤害半径；实际只有对空目标会进入范围伤害分支
    blastRadius: 0.35,
  },
]
```

## 战舰配置（ships.json5）

```json5
[
  {
    // 战舰名称（不可重复）
    name: "atlanta",
    // 国籍，用于图鉴筛选，可选值：
    // us 美国 / jp 日本 / de 德国 / uk 英国
    // su 苏联 / cn 中国 / special 特殊或中立单位
    nation: "us",
    // 战舰类型，可选值：
    // aircraft_carrier 航空母舰
    // battleship 战列舰
    // cruiser 巡洋舰
    // destroyer 驱逐舰
    // frigate 护卫舰
    // cargo 货轮
    // torpedo_boat 快艇
    // submarine 潜艇（未来可期）
    type: "cruiser",
    // 类型缩写
    typeAbbr: "CL",
    // 初始生命值
    // 推荐值：满载排水量（单位：吨）
    totalHP: 1000,
    // 伤害减免比例（必须 <= 1）
    // 0.7 -> (1-0.7=0.3) -> 仅受到击中的弹药的 30% 伤害
    // -0.5 -> (1+0.5=1.5) -> 会收到击中的弹药的 150% 伤害
    // 水平：计算鱼雷 / 直射炮弹用
    horizontalDamageReduction: 0.5,
    // 垂直：计算曲射炮弹用，一般要大于水平的值
    verticalDamageReduction: 0.25,
    // 最大速度
    // 推荐值：实际速度（节）如 30 节 -> 30
    maxSpeed: 30,
    // 加速度
    // 推荐值：最大速度的 1/500 - 1/200，船越大越慢
    acceleration: 0.3,
    // 转向速度（度）
    rotateSpeed: 2,
    // 长度（实际长度，单位：米）
    length: 220,
    // 宽度（实际宽度，单位：米）
    width: 22,
    // 资金消耗
    fundsCost: 150,
    // 增援时间
    timeCost: 12,
    // 吨位
    tonnage: 8000,
    // 可选：俯视逐帧动画；静止时使用 idleTopFrame
    animation: {
      topFrames: ["swordfish_01", "swordfish_02", "swordfish_03"],
      idleTopFrame: "swordfish_03",
      frameTicks: 6,
    },
    // 战舰描述（建议不多于 4 行）
    description: [
      "导弹巡洋舰（美）",
      "主炮：2 座单装 127mm/38",
    ],
    // 武器配置
    weapon: {
      // 主炮
      mainGuns: [
        // 前主炮 A
        {
          // 舰炮名称，需保证存在
          name: "US/127/38/MK45",
          // 相对位置
          // 0.35 -> 从中心往舰首 35% 舰体长度
          // -0.3 -> 从中心往舰尾 30% 舰体长度
          posPercent: 0.3,
          // 右侧射界（顺时针，12 点钟为 0，3 点钟为 90，6 点钟为 180，以此类推）
          rightFiringArc: [0, 150],
          // 左侧射界（若某侧射界不存在，则使用 right: [0, 0] 或 left: [360, 360]）
          leftFiringArc: [210, 360]
        },
        // 后主炮 B
        {
          name: "US/127/38/MK45",
          posPercent: -0.5,
          rightFiringArc: [30, 180],
          leftFiringArc: [180, 330]
        }
      ],
      // 副炮（参数与主炮相同）
      secondaryGuns: [
        // 副炮 A
        {
          name: "US/76/50",
          posPercent: 0.3,
          rightFiringArc: [0, 120],
          leftFiringArc: [240, 360]
        },
      ],
      // 防空炮（参数与主炮相同）
      antiAircraftGuns: [
        // 防空炮 A
        {
          name: "US/20/70",
          posPercent: 0.5,
          rightFiringArc: [0, 180],
          leftFiringArc: [180, 360]
        },
      ],
      // 鱼雷发射器（参数与主炮相同）
      torpedoes: [
        // 中轴鱼雷 A
        {
          name: "US/533/4",
          posPercent: 0.1,
          rightFiringArc: [30, 150],
          leftFiringArc: [210, 330]
        },
      ],
      // 火箭发射器（参数与主炮相同）
      rockets: [
        // 右防空火箭炮 A
        {
          name: "JP/120/21/Type5/AA",
          posPercent: 0,
          rightFiringArc: [0, 180],
          leftFiringArc: [360, 360]
        },
      ]
    },
    // 舰载机联队（仅航空母舰需要配置）
    aircraft: {
      // 起飞间隔（单位：秒）
      takeOffTime: 1,
      // 飞机编组
      groups: [
        {
          // 飞机名称（需确保在 planes.json5 中存在）
          name: "F",
          // 最大数量
          maxCount: 12
        },
        {
          name: "B",
          maxCount: 12
        },
        {
          name: "A",
          maxCount: 12
        }
      ]
    }
  }
]
```

## 飞机配置（planes.json5）

```json5
[
  {
    // 飞机名称（不可重复）
    name: "F4F-3",
    // 展示用名称
    displayName: "野猫",
    // 国籍，取值与舰船配置一致
    nation: "us",
    // 飞机类型，可选值：
    // fighter 战斗机
    // dive_bomber 俯冲轰炸机
    // torpedo_bomber 鱼雷轰炸机
    type: "fighter",
    // 类型缩写
    typeAbbr: "F",
    // 服役年份
    year: 1940,
    // 初始生命值
    totalHP: 35,
    // 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
    damageReduction: 0.5,
    // 最大速度
    maxSpeed: 533,
    // 加速度
    acceleration: 30,
    // 转向速度（度）
    rotateSpeed: 12,
    // 飞机长度（实际长度，单位：米）
    length: 9,
    // 飞机宽度（翼展，单位：米）
    width: 12,
    // 造价
    fundsCost: 10,
    // 耗时
    timeCost: 5,
    // 吨位
    tonnage: 3.367,
    // 总航程
    range: 1360,
    // 武器配置
    weapon: {
      // 机炮（参数与舰炮武器挂载点相同）
      guns: [
        {
          // 舰炮名称，需保证存在
          name: "US/12.7",
          // 相对位置（与战舰武器位置参数含义相同）
          posPercent: 0.5,
          // 右侧射界
          rightFiringArc: [0, 20],
          // 左侧射界
          leftFiringArc: [340, 360]
        },
      ],
      // 炸弹（仅俯冲轰炸机使用，参数与机炮挂载点相同）
      bombs: [
        {
          // 释放器名称（需确保在 releasers.json5 中存在）
          name: "US/BB/907",
          posPercent: 0.3,
          rightFiringArc: [0, 20],
          leftFiringArc: [340, 360]
        },
      ],
      // 鱼雷（仅鱼雷轰炸机使用，参数与机炮挂载点相同）
      torpedoes: [
        {
          // 释放器名称（需确保在 releasers.json5 中存在）
          name: "US/AST/570",
          posPercent: 0.3,
          rightFiringArc: [0, 20],
          leftFiringArc: [340, 360]
        },
      ],
      // 火箭弹（参数与机炮挂载点相同，名称需确保在 plane_rocket_launchers.json5 中存在）
      rockets: [
        {
          name: "US/AIR/RB/127/4",
          posPercent: 0.25,
          rightFiringArc: [0, 25],
          leftFiringArc: [335, 360]
        },
      ],
      // 释放间隔（单位：秒，炸弹/鱼雷两次投放之间的最小间隔）
      releaseInterval: 5
    }
  }
]
```

## 释放器配置（releasers.json5）

```json5
[
  {
    // 释放器名称（不可重复）
    // US 是所属阵营，BB 表示炸弹，AST 表示航空鱼雷
    name: "US/BB/907",
    // 弹药名称（需确保在 bullets.json5 中存在）
    bulletName: "US/BB/2000/907",
    // 射程（地图格数）
    range: 1.5,
    // 弹药速度
    bulletSpeed: 30,
  }
]
```

## 地图配置（maps.json5）

```json5
[
  {
    // 地图名称（不可重复）
    name: "darwin",
    // 展示用名称
    displayName: "达尔文港",
    // 英文展示用名称；英文界面使用
    displayNameEn: "Darwin Harbour",
    // 俄文、日文展示用名称；缺失时按 ru/ja → en → zh-Hans 回退
    displayNameRu: "Порт-Дарвин",
    displayNameJa: "ダーウィン港",
    // 地图资源名称
    source: "darwin"
  }
]
```

## 游戏设置配置（game_settings.json5）

```json5
{
  // 全局速度倍率
  // 影响战舰、炮弹、鱼雷、飞机等移动/转向速度
  // 范围: 0.25 ~ 4.0，默认值: 1.0
  "speedMultiplier": 1.0,
  // 游戏界面语言；当前正式启用 zh-Hans / en / ru / ja
  "language": "zh-Hans"
}
```

## 任务关卡配置（missions.json5）

```json5
[
  {
    // 任务关卡名称（不可重复）
    name: "default",
    // 任务分类：classic（经典）或 test（测试）；缺省时按 classic 处理
    category: "classic",
    // 分类内显示顺序沿用本数组中的书写顺序
    // 展示用名称
    displayName: "默认",
    // 英文展示用名称；英文界面使用
    displayNameEn: "Default",
    // 俄文、日文展示用名称；缺失时按 ru/ja → en → zh-Hans 回退
    displayNameRu: "По умолчанию",
    displayNameJa: "デフォルト",
    // 初始资金
    initFunds: 10000,
    // 初始相机视角位置（需在地图范围内）
    initCameraPos: [30, 30],
    // 地图名称（需确保存在）
    mapName: "default",
    // 最大战舰数量（目前不生效）
    maxShipCount: 5,
    // 关卡描述；中文界面使用
    description: "默认关卡",
    // 英文关卡描述；英文界面使用
    descriptionEn: "Default mission",
    // 俄文、日文关卡描述；缺失时按 ru/ja → en → zh-Hans 回退
    descriptionRu: "Стандартная миссия",
    descriptionJa: "デフォルト作戦",
    // 增援点信息
    initReinforcePoints: [
      {
        // 位置
        pos: [1, 20],
        // 方向
        rotation: 90,
        // 集结点
        rallyPos: [25, 35],
        // 所属方
        belongPlayer: "HA",
        // 最大队列数量
        maxOncomingShip: 10,
        // 可选择的战舰名称
        providedShipNames: [
          "south_dakota",
          "astoria",
          "atlanta",
          "porter",
          "PT_791",
          "liberty",
        ],
      },
    ],
    // 石油平台
    initOilPlatforms: [
      {
        // 位置
        pos: [10, 10],
        // 半径
        radius: 3,
        // 单次产生资金
        yield: 10
      },
      {
        pos: [40, 68],
        radius: 4,
        yield: 25
      }
    ],
    initShips: [
      // 己方初始战舰
      {
        name: "default",
        pos: [40, 33],
        rotation: 90,
        // 指定所属玩家为本人（humanAlpha）
        belongPlayer: "HA"
      },
      // 敌人初始战舰
      {
        name: "default",
        pos: [70, 35],
        rotation: 90,
        // 指定所属玩家为电脑（computerAlpha）      
        belongPlayer: "CA"
      }
    ]
  }
]
```
