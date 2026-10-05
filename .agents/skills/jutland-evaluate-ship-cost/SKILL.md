---
name: jutland-evaluate-ship-cost
description: Evaluate and assign Jutland ship and weapon fundsCost/timeCost from runtime combat power, burst/projection, effective HP, and aircraft cost. Use when adding ships, rebalancing fleets, or auditing weapon and reinforcement costs.
---

# Jutland 舰船与武器费用评估

## 核心规则

- 舰船费用以初始化后的运行时战力为主，不直接累加武器配置单价。
- 武器与特殊武器统一使用 `$1–100` 资金尺度；当前运行时不会单独购买武器，彩蛋单位和无阵营路径的特殊武器保留手工价格，但同样限制在该范围内。
- 舰载机资金在 `GetShipCost` 中全额追加，舰船配置只保存舰体费用。
- 全量重平衡必须先输出审计结果，与用户确认异常项后再写回配置。

## 文件与脚本

- `export_ship_cost_data.go`：普通 Go 导出器，加载初始化后的全部舰船运行时战力。
- `scripts/evaluate_weapon_costs.py`：评估武器参考价；`--output` 指定结果路径，追加 `--apply` 才写回配置。
- `scripts/evaluate_ship_costs.py`：通过 `--combat-power` 读取运行时战力，通过 `--output` 写出舰船建议价。
- `scripts/apply_ship_costs.py`：通过 `--costs` 读取已确认的建议价并写回 `configs/ships.json5`。
- `scripts/evaluate_ship_costs.sh`：编排完整只读审计流程。

```bash
# 只读审计
AUDIT_DIR="$(mktemp -d)"
bash .agents/skills/jutland-evaluate-ship-cost/scripts/evaluate_ship_costs.sh \
  "$AUDIT_DIR"

# 确认后写回
python3 .agents/skills/jutland-evaluate-ship-cost/scripts/evaluate_weapon_costs.py --apply
python3 .agents/skills/jutland-evaluate-ship-cost/scripts/apply_ship_costs.py \
  --costs "$AUDIT_DIR/ship_costs.json"
```

## 武器参考价

```text
# 单发期望伤害与游戏战力公式一致
expectedDamage = damage × (1 + 2.7 × criticalRate)
salvoDamage    = expectedDamage × projectileCount

# 持续输出与首轮爆发按 70% / 30% 混合
referenceScore = salvoDamage × (
    0.70 / actualCycle
  + 0.30 / referenceCycle
  ) × weaponTypeFactor
weaponCost = clamp(
    quantize(referenceScore / 5),
    1,
    100,
)

quantize(x) = round(x)      # x < 10 时用 $1 步进
            = roundTo5(x)   # x >= 10 时用 $5 步进
```

`referenceScore` 是武器强弱比较分，不直接等同于游戏资金；除以 `5` 后再写入 `fundsCost`，避免一座发射器或炮塔的显示价格接近整艘主力舰。

`$5` 步进在低分段太粗（参考分 0~12.5 会被一起抹成 0，6 英寸单装副炮与机枪同价），因此 `$10` 以下改用 `$1` 步进（`COST_FINE_STEP` / `COST_FINE_STEP_LIMIT`）。三管 25mm 机炮参考分只有 `1.6`，仍会与 7.7mm 机枪同价，所以单独为 25mm 机炮设置最低参考价 `$2`（见 `MIN_GUN_COST_BY_CALIBER`）；机枪（参考分 < 0.5，取整后仍为 `0`）保持 `$1`。

| 武器 | actualCycle | referenceCycle | weaponTypeFactor |
|---|---|---:|---:|
| 舰炮 | `reloadTime` | 20s | 0.50 |
| 鱼雷 | `reloadTime + (count - 1) × shotInterval` | 60s | 1.00 |
| 舰载火箭 | 装填、组内间隔、组间间隔组成的完整周期 | 60s | 0.70 |

带 `/` 的标准武器统一计算。`RailGun`、`RushYa`、`RunYa`、`FlyYa`、`LeiYa`、`Impact`、`YaLei`、`ShanDa`、`DuckRocket` 等特殊武器没有阵营路径，保留手工价格。

## 舰船费用

脚本通过 `combatpower.CalculateShip` 的结果定价，已经考虑水平/垂直减伤与有效生命值、持续对舰与对空输出、暴击期望、武器散布/射界/射程/命中效率、舰船速度/转向/加速度、鱼雷与火箭完整发射周期。对不带打击型舰载机（含无舰载机）的舰船：

```text
economicPower = HullPower + 0.25 × Burst + 0.10 × Projection
```

以舰载机承担打击任务的单位（`aircraft_carrier`，以及搭载非 `scout` 机型的航空战列舰等混合舰）其 `Burst` / `Projection` 已混入航空贡献，为避免与运行时飞机资金重复计价，只使用 `HullPower`。判定依据是舰载机机型（`planes.json5` 的 `type`）而不是运行时航空战力分：仅带侦察机（`scout`）的舰船其航空贡献可忽略，仍按完整口径计价，`aviation` 四舍五入到 0 还是 1 不影响结论。

战力价与耐久保底：

```text
combatCost = roundTo5(typeBaseCost + economicPower × typePowerFactor)
hullFloor  = roundTo5(EHP^0.45 × hullFloorMultiplier × 3.0)
fundsCost  = max(5, combatCost, hullFloor)
```

| 舰种 | typeBaseCost | typePowerFactor | hullFloorMultiplier |
|---|---:|---:|---:|
| torpedo_boat | 5 | 1.20 | 0.10 |
| destroyer | 20 | 0.85 | 0.18 |
| frigate | 15 | 0.60 | 0.15 |
| cruiser | 40 | 0.65 | 0.35 |
| battleship | 230 | 0.55 | 0.80 |
| aircraft_carrier | 80 | 0.55 | 0.65 |
| cargo | 20 | 0.80 | 0.30 |
| repair | 20 | 0.80 | 0.30 |
| hospital | — | — | 0.50 |

医疗船没有武器，常规战力为零，价格用 `fundsCost = roundTo5(hullFloor + 25)`；固定 `$25` 表示治疗能力、医疗设施和非战斗支援价值。`nation == special` 的彩蛋舰船保留手工价格与耗时。

战略层级修正：静态战力不能完整表达科技树位置、终局稀缺度和设计层级，仅在用户确认后允许为极少数舰船添加显式倍率，按 `fundsCost = roundTo5(fundsCost × strategicMultiplier)` 计算，并在脚本与本文档同时记录。

| 舰船 | strategicMultiplier | 原因 |
|---|---:|---|
| satsuma | 1.10 | 与大和保持接近，避免仅因当前持续火力配置而明显掉档 |
| edo | 1.15 | 日本终局舰溢价，确保价格明显高于大和 |
| illinois | 1.10 | 衣阿华级大型舰体改装成本与战略稀缺性，避免按普通舰队航母低估 |
| clemenceau_pa28 | 1.25 | 战后快速射击高平两用炮显著提高舰体战力，避免因较低 EHP 被航母保底价低估 |

不要通过调整全局战列舰系数来解决单艘舰的科技树层级问题，否则会连带扭曲其他国家战列舰。

舰载机：

```text
aircraftCost = sum(maxCount × planeFundsCost)
totalCost    = fundsCost + aircraftCost

airWingFitPenalty = clamp(
    round(sqrt(aircraftCount) × avgPlaneTime × 0.12
      + (aircraftTypes - 1) × 1.5),
    0,
    18,
)
```

飞机与舰体视为可并行生产，舰载机只增加小额适配时间。适配时间国家倍率：`us=0.75`、`uk=0.90`、`jp/de/fr=1.00`、`ru/su=1.05`、`cn=0.50`。

## 增援耗时

```text
baseTime = clamp(round(2 + fundsCost × 0.35), 3, 130)     # 普通舰种
baseTime = clamp(round(40 + fundsCost × 0.13), 85, 180)   # 战列舰：更宽区间，避免全挤在 130 秒上限

timeCost = clamp(
    baseTime + airWingFitPenalty × nationAirFitMultiplier,
    3,
    typeTimeMax,
)
```

## 审计与验证

全量评估后至少检查：

1. 同类型舰船的费用是否随运行时战力、爆发和投送能力合理分层。
2. 同系列武器是否随弹数、伤害和周期单调变化；只有终局级或特殊武器可以达到 `$100` 上限。
3. 航母必须同时看舰体费与含满编舰载机的总费用。
4. 医疗船、彩蛋单位、特殊武器不得被通用公式无意覆盖。
5. 鱼雷巡洋舰、航空战列舰等极端配置需检查其历史版本和实际挂载，不能只按舰名比较。

```bash
# 写回前
PYCACHE_DIR="$(mktemp -d)"
PYTHONPYCACHEPREFIX="$PYCACHE_DIR" python3 -m py_compile \
  .agents/skills/jutland-evaluate-ship-cost/scripts/evaluate_weapon_costs.py \
  .agents/skills/jutland-evaluate-ship-cost/scripts/evaluate_ship_costs.py \
  .agents/skills/jutland-evaluate-ship-cost/scripts/apply_ship_costs.py
rm -rf "$PYCACHE_DIR"

# 写回后
make build
go test ./pkg/mission/object/combatpower/ ./pkg/mission/object/unit/
```
