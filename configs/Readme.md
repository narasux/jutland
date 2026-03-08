# 配置文件说明

## 弹药配置（bullets.json5）

```json5
[
  {
    // 弹药名称（不可重复）
    // US 表示所属阵营
    // GB (gun bullet) 表示是火炮弹药
    // TB (torpedo bullet) 表示鱼雷弹药
    // 127 表示口径 127mm
    // 1932 表示 1932 年研制（不一定准确，主要是做区分）
    name: "US/GB/127/1932",
    // 弹药类型
    // shell 炮弹
    // torpedo 鱼雷
    // bomb 炸弹
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

## 战舰配置（ships.json5）

```json5
[
  {
    // 战舰名称（不可重复）
    name: "atlanta",
    // 展示用名称
    displayName: "亚特兰大",
    // 战舰类型，可选值：
    // aircraft_carrier 航空母舰
    // battleship 战列舰
    // cruiser 巡洋舰
    // destroyer 驱逐舰
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
    // 飞机类型，可选值：
    // fighter 战斗机
    // dive_bomber 俯冲轰炸机
    // torpedo_bomber 鱼雷轰炸机
    type: "fighter",
    // 类型缩写
    typeAbbr: "F",
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
    // 飞机描述
    description: [
      "战斗机（美）",
      "4 x 12.7mm 机枪",
    ],
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
  "speedMultiplier": 1.0
}
```

## 任务关卡配置（missions.json5）

```json5
[
  {
    // 任务关卡名称（不可重复）
    name: "default",
    // 展示用名称
    displayName: "默认",
    // 初始资金
    initFunds: 10000,
    // 初始相机视角位置（需在地图范围内）
    initCameraPos: [30, 30],
    // 地图名称（需确保存在）
    mapName: "default",
    // 最大战舰数量（目前不生效）
    maxShipCount: 5,
    // 关卡描述
    description: [
      "默认关卡"
    ],
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