# 配置文件说明

## 弹药配置（bullets.json）

```json5
[
  {
    // 弹药名称（不可重复）
    // GB (gun bullet) 表示是火炮弹药
    // TB (torpedo bullet) 表示鱼雷弹药
    // 127 表示口径 127mm，T1 表示 Type1
    "name": "GB127T1",
    // 弹药命中伤害值
    "damage": 100,
    // 弹药生命期，过期后失效
    // TODO 提供折算公式
    "life": 150
  }
]
```

## 舰炮配置（guns.json）

```json5
[
  {
    // 舰炮名称（不可重复）
    "name": "MK45",
    // 弹药名称（需确保一定存在）
    "bulletName": "GB127T1",
    // 炮管数量
    "bulletCount": 1,
    // 装填时间（单位：秒）
    "reloadTime": 1,
    // 射程（地图格数）
    "range": 20,
    // 散布半径
    "bulletSpread": 50,
    // 炮弹速度
    // TODO 提供折算公式
    "bulletSpeed": 0.5
  }
]
```

## 鱼雷配置（torpedoes.json）

TODO 待补充

## 战舰配置（ships.json）

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
    "totalHP": 1000,
    // 伤害减免（0.7 -> 仅受到击中的 70% 伤害)
    "damageReduction": 0.5,
    // 最大速度
    // TODO 提供折算公式
    "maxSpeed": 0.1,
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
        {
          // 舰炮名称，需保证存在
          "name": "MK45",
          // 相对位置
          // 0.35 -> 从中心往舰首 35% 舰体长度
          // -0.3 -> 从中心往舰尾 30% 舰体长度
          "posPercent": 0.3
        },
        {
          "name": "MK45",
          "posPercent": -0.3
        }
      ],
      // 鱼雷
      "torpedoes": []
    }
  }
]
```

## 任务关卡配置（missions.json）

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