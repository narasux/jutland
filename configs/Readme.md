# 配置文件说明

## 弹药配置（bullets.json5）

```json5
[
  {
    // 弹药名称（不可重复）
    // GB (gun bullet) 表示是火炮弹药
    // TB (torpedo bullet) 表示鱼雷弹药
    // 127 表示口径 127mm，T1 表示 Type1
    "name": "GB127T1",
    // 弹药命中伤害值
    // 推荐公式：现实炮弹重量（kg) * 3
    "damage": 100,
    // 弹药生命期，过期后失效
    // 推荐值：150-200，射程越远越大
    "life": 150
  }
]
```

## 舰炮配置（guns.json5）

```json5
[
  {
    // 舰炮名称（不可重复）
    // MK45 是型号，38 是倍径
    "name": "MK45/38",
    // 弹药名称（需确保一定存在）
    "bulletName": "GB127T1",
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
    // 推荐值：1km/s -> 0.5
    "bulletSpeed": 0.5
  }
]
```

## 鱼雷发射器配置（torpedo_launchers.json5）

```json5
[
  {
    // 鱼雷发射器名称
    // US 表示美国，533 表示口径，4 表示管数
    "name": "US533/4",
    // 鱼雷名称
    "bulletName": "US/TB533T1",
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
    // 推荐值：30 节是 0.1，其他以此类推
    "bulletSpeed": 0.15,
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
    // 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
    "damageReduction": 0.7,
    // 最大速度
    // 推荐值：30 节是 0.1，其他以此类推
    "maxSpeed": 0.1,
    // 加速度
    // 推荐值：最大速度的 1/500 - 1/200，船越大越慢
    "acceleration": 0.01,
    // 转向速度（度）
    "rotateSpeed": 2,
    // 长度（实际长度，单位：米）
    "length": 220,
    // 宽度（实际宽度，单位：米）
    "width": 22,
    // 武器配置
    "weapon": {
      // 舰炮
      "guns": [
        // 前主炮 A
        {
          // 舰炮名称，需保证存在
          "name": "MK45",
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
          "name": "MK45/38",
          "posPercent": -0.5,
          "rightFiringArc": [30, 180],
          "leftFiringArc": [180, 330]
        }
      ],
      // 鱼雷
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