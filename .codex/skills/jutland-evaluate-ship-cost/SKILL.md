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
hullTime    = clamp(round(fundsCost * 0.35 + 2), 3, 130)
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
| ru / su | 1.10 |
| cn | 1.00 |

### 舰载机费（运行时计算）

```
aircraftCost = sum(planeCount * planeFundsCost)
totalCost    = fundsCost + aircraftCost
```

舰载机资金全额计入总费用，但不线性计入舰船建造时间。舰体与舰载机生产在设定上可并行，航母只承担小额的“飞行队适配时间”，用于表达机库、甲板调度、备件、弹药与飞行队磨合成本。

```
aircraftCount = sum(planeCount)
aircraftTypes = count(unique planeName)
avgPlaneTime  = weightedAverage(planeTimeCost, planeCount)

airWingFitPenalty = clamp(
    round(sqrt(aircraftCount) * avgPlaneTime * 0.12 + (aircraftTypes - 1) * 1.5),
    0,
    18,
)
timeCost = clamp(round(hullTime + airWingFitPenalty * nationTimeMultiplier), 3, 130)
```

| 国家 | nationTimeMultiplier | 说明 |
|------|----------------------|------|
| us | 0.75 | 工业化建造与大规模舰载机整备优势 |
| uk | 0.90 | 成熟海军工业，略快 |
| jp | 1.00 | 基准 |
| de | 1.00 | 基准 |
| ru / su | 1.05 | 略慢 |
| cn | 0.50 | 建造/适配时间显著优惠 |

### 年份性价比（后续校准目标）

年份越新的单位应有更高目标性价比。当前脚本先保留 HP / 武器 / 舰载机三层费用模型；下一版若接入 `CombatPower.Total`，推荐改为以目标性价比反推费用：

```
targetEfficiency = baseEfficiency(type) * yearEfficiencyBonus(year) * nationEfficiencyBonus(nation)
fundsCost        = combatPower / targetEfficiency
```

这样年份和国家工业能力会成为显式平衡参数，而不是依赖取整、上限或配置偶然性。

## 当前校准

- hullScaleFactor: 3.6
- WEAPON_SCALE: 10

## 工作流程

1. 运行 `bash scripts/evaluate_ship_costs.sh` 查看费用
2. 确认后运行 `python3 scripts/apply_ship_costs.py` 写入 ships.json5
3. `make build` 验证
