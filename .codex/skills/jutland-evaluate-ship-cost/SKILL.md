---
name: jutland-evaluate-ship-cost
description: Evaluate and assign costs for Jutland ships and weapons using a three-layer model (weapon unit cost + hull base cost + aircraft cost). Use when adding new ships, rebalancing costs, or when weapon/ship costs need systematic evaluation.
---

# Jutland 舰船费用评估

## 核心规则

- 在仓库根目录工作。先阅读 `AGENTS.md` 并执行 `git status --short`。
- 三层费用模型：武器单价 + 舰体基础费 + 舰载机费。
- 公式参数在本文件中集中维护；修改时必须同步更新脚本。

## 脚本

- `scripts/evaluate_weapon_costs.py` — 读取 bullets + guns/torpedo/rocket，计算武器费用到 /tmp/weapon_costs.json
- `scripts/evaluate_ship_costs.py` — 读取 ships + weapons + planes，计算舰体费 + 总费，输出 TSV
- `scripts/apply_ship_costs.py` — 套用公式写入 configs/ships.json5
- `scripts/evaluate_ship_costs.sh` — 编排脚本

### 使用方式

```bash
bash .codex/skills/jutland-evaluate-ship-cost/scripts/evaluate_ship_costs.sh
```

## 费用模型

### 武器单价

```
weaponRawCost = damage * bulletCount / reloadTime * weaponTypeFactor
weaponCost    = clamp(round(weaponRawCost / 5) * 5, 1, 200)
```

| 武器类型 | weaponTypeFactor |
|---------|-----------------|
| 舰炮 (gun) | 0.50 |
| 鱼雷 (torpedo) | 1.00 |
| 舰载火箭 (rocket) | 0.70 |

### 舰体基础费

```
hullRawCost = HP^0.45 * typeMultiplier * nationMultiplier * hullScaleFactor
fundsCost   = hullCost + weaponCost/10
timeCost    = clamp(round(fundsCost * 0.35 + 2), 3, 130)
```

| 舰种 | typeMultiplier |
|-----|---------------|
| default | 0.00 |
| torpedo_boat | 0.10 |
| destroyer | 0.18 |
| frigate | 0.15 |
| cruiser | 0.35 |
| battleship | 0.80 |
| aircraft_carrier | 0.65 |
| cargo | 0.30 |
| hospital | 0.50 |

| 国家 | nationMultiplier |
|------|-----------------|
| us | 0.90 |
| jp | 1.00 |
| de | 1.05 |
| uk | 1.00 |
| ru | 1.10 |
| cn | 1.00 |

### 舰载机费（运行时计算）

```
aircraftCost = sum(planeCount * planeFundsCost)
totalCost    = fundsCost + aircraftCost
```

## 当前校准

- hullScaleFactor: 3.6
- WEAPON_SCALE: 10

## 工作流程

1. 运行 `bash scripts/evaluate_ship_costs.sh` 查看费用
2. 确认后运行 `python3 scripts/apply_ship_costs.py` 写入 ships.json5
3. `make build` 验证
