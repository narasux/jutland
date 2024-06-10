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
    "name": "US/GB/127/1932",
    // 弹药类型
    // shell 炮弹
    // torpedo 鱼雷
    "type": "shell",
    // 口径
    "diameter": 305,
    // 弹药命中伤害值
    // 推荐公式：现实炮弹重量（kg) * 3
    "damage": 100,
    // 暴击率：造成暴击伤害的几率（击中弹药库之类）
    // 注：超级暴击率固定为暴击率的 1/10
    "criticalRate": 0.002
  }
]
```

## 舰炮配置（guns.json5）

```json5
[
  {
    // 舰炮名称（不可重复）
    // US 是所属阵营，127 是口径，38 是倍径，MK45 是型号
    "name": "US/127/38/MK45",
    // 弹药名称（需确保一定存在）
    "bulletName": "US/GB/127/1932",
    // 炮管数量
    "bulletCount": 1,
    // 装填时间（单位：秒）
    // 推荐值：现实装填速度 / 2
    "reloadTime": 1,
    // 射程（地图格数）
    // 推荐值：现实射程（km）
    "range": 20,
    // 散布半径
    // TODO 提供折算公式
    "bulletSpread": 50,
    // 炮弹速度
    // 推荐值：1km/s -> 1000
    "bulletSpeed": 1000
  }
]
```

## 鱼雷发射器配置（torpedo_launchers.json5）

```json5
[
  {
    // 鱼雷发射器名称
    // US 是所属阵营，533 表示口径，4 表示管数
    "name": "US/533/4",
    // 鱼雷名称
    "bulletName": "US/TB/533/1931",
    // 发射器管数
    "bulletCount": 4,
    // 发射间隔（单位：秒）
    //（建议至少为 1）
    "shotInterval": 1,
    // 装填时间（单位：秒）
    // 推荐值：不要少于 35，否则很破坏平衡
    "reloadTime": 50,
    // 射程（地图格数）
    // 推荐值：现实射程（km）
    "range": 12,
    // 鱼雷速度
    // 推荐值：实际速度（节）如 45 节 -> 45
    "bulletSpeed": 45,
  },
]
```

## 战舰配置（ships.json5）

```json5
[
  {
    // 战舰名称（不可重复）
    "name": "default",
    // 战舰类型，可选值：
    // carrier 航空母舰（未来可期）
    // battleship 战列舰
    // cruiser 巡洋舰
    // destroyer 驱逐舰
    // frigate 护卫舰
    // speedboat 快艇
    // submarine 潜艇（未来可期）
    "type": "cruiser",
    // 初始生命值
    // 推荐值：满载排水量（单位：吨）
    "totalHP": 1000,
    // 伤害减免
    // 0.7 -> 仅受到击中的弹药的 70% 伤害
    // 1.5 -> 会收到击中的弹药的 150% 伤害
    // 水平：计算鱼雷 / 直射炮弹用
    "horizontalDamageReduction": 0.25,
    // 垂直：计算曲射炮弹用，一般要大于水平的值
    "verticalDamageReduction": 0.25,
    // 最大速度
    // 推荐值：实际速度（节）如 30 节 -> 30
    "maxSpeed": 30,
    // 加速度
    // 推荐值：最大速度的 1/500 - 1/200，船越大越慢
    "acceleration": 0.3,
    // 转向速度（度）
    "rotateSpeed": 2,
    // 长度（实际长度，单位：米）
    "length": 220,
    // 宽度（实际宽度，单位：米）
    "width": 22,
    // 武器配置
    "weapon": {
      // 主炮
      "mainGuns": [
        // 前主炮 A
        {
          // 舰炮名称，需保证存在
          "name": "US/127/38/MK45",
          // 相对位置
          // 0.35 -> 从中心往舰首 35% 舰体长度
          // -0.3 -> 从中心往舰尾 30% 舰体长度
          "posPercent": 0.3,
          // 右侧射界（顺时针，12 点钟为 0，3 点钟为 90，6 点钟为 180，以此类推）
          "rightFiringArc": [0, 150],
          // 左侧射界（若某侧射界不存在，则使用 right: [0, 0] 或 left: [360, 360]）
          "leftFiringArc": [210, 360]
        },
        // 后主炮 B
        {
          "name": "US/127/38/MK45",
          "posPercent": -0.5,
          "rightFiringArc": [30, 180],
          "leftFiringArc": [180, 330]
        }
      ],
      // 副炮（参数与主炮相同）
      "secondaryGuns": [],
      // 鱼雷发射器（参数与主炮相同）
      "torpedoes": []
    }
  }
]
```

## 任务关卡配置（missions.json5）

```json5
[
  {
    // 任务关卡名称（不可重复）
    "name": "default",
    // 初始相机视角位置（需在地图范围内）
    "initCameraPos": [30, 30],
    // 地图名称（需确保存在）
    "mapName": "default",
    // 最大战舰数量（目前不生效）
    "maxShipCount": 5,
    "initShips": [
      // 己方初始战舰
      {
        "name": "default",
        "pos": [40, 33],
        "rotation": 90,
        "belongPlayer": "humanAlpha"
      },
      // 敌人初始战舰
      {
        "name": "default",
        "pos": [70, 35],
        "rotation": 90,
        // 指定所属玩家为电脑       
        "belongPlayer": "computerAlpha"
      }
    ]
  }
]
```