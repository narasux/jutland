---
name: jutland-evaluate-plane-cost
description: Evaluate and assign fundsCost and timeCost for Jutland aircraft from combat power data. Use when adding planes, deriving plane costs from combat power statistics, or rebalancing aircraft cost.
---

# Jutland 飞机花费评估

## 核心规则

- 花费评估必须基于脚本提取的实际战力数据，不做凭空估算。
- 公式参数集中维护在本文件；修改公式时必须同步更新脚本与本文档。

## 目标文件

- 飞机配置：`configs/planes.json5`（`fundsCost` 与 `timeCost` 字段）
- Go 辅助函数：`pkg/mission/object/unit/plane.go`（`GetPlaneCost` 函数）
- 战力计算：`pkg/mission/object/combatpower/combatpower.go`（`CalculatePlane` 函数）

## 脚本

- `scripts/evaluate_plane_costs.sh`：编排脚本，提取战力数据、套用花费公式、输出建议值。
- `export_plane_cost_data.go`：普通 Go 导出器，加载初始化后的飞机并输出结构化战力数据；不向业务包复制临时测试。
- `scripts/plane_cost_calc.py`：备用配置估算器，Go 导出器因本机图形环境触发 Ebiten 初始化失败时使用；追加 `--apply` 才写回 `configs/planes.json5`。

流程：运行 `go run export_plane_cost_data.go` 加载初始化数据，解析导出 JSON 的 `name`、`type`、`nation`、`combatPower`、`tonnage`，套用下方公式后输出 TSV，列依次为 `name`、`type`、`nation`、`combatPower`、`tonnage`、`fundsCost`、`timeCost`。

```bash
bash .agents/skills/jutland-evaluate-plane-cost/scripts/evaluate_plane_costs.sh
```

- 主路径依赖 Ebiten 图形环境：本机 `go run` 常在 `internal/ui` 的 `currentMouseLocation` 空指针处 panic（无图形会话时 `go test` 的 Ebiten 包同样 panic）。**失败一次就直接改用备用估算器**，不要翻 Ebiten 源码找原因，也不要反复重跑主路径。
- 备用估算器默认只读建议值：`python3 .agents/skills/jutland-evaluate-plane-cost/scripts/plane_cost_calc.py configs/planes.json5`。
- 交付前与既有配置对表：多数机型与仓库现值一致；新机型建议值若与同类既有机型（同 type、同年代、战力接近）明显冲突，按同类既有分层取值并在最终说明里记录差异。

## 花费公式

```
rawFunds  = combatPower * typeMultiplier * scaleFactor
fundsCost = clamp(round(rawFunds), 3, 120)
timeCost  = clamp(round(fundsCost * 0.35 + 2), 3, 50)
```

| 参数 | 值 | 说明 |
|---|---|---|
| `typeMultiplier` (fighter) | 1.00 | 战斗机单位战力费用最低 |
| `typeMultiplier` (dive_bomber) | 1.15 | 俯冲轰炸机携带炸弹，费用略高 |
| `typeMultiplier` (attacker) | 1.20 | 攻击机同时挂机炮、火箭与小型炸弹，且装甲减伤高 |
| `typeMultiplier` (level_bomber) | 1.15 | 水平轰炸机同样携带炸弹，费用与俯冲机相同 |
| `typeMultiplier` (torpedo_bomber) | 1.30 | 鱼雷轰炸机挂载最重，费用最高 |
| `scaleFactor` | 0.10 / 0.30 | 主路径 scaleFactor=0.10，Python 备用估算器 fallbackScaleFactor=0.30 |
| `fundsCost` 范围 | 3–120 | 最便宜飞机不低于 $3；重型轰炸机战力与载弹量高，允许显著超过 $30（如 B-17G ≈ $114） |
| `timeCost` 范围 | 3–50 | 建造时间随资金线性增长，钳制在 3–50 秒 |
| `nation == special` | 手工 | 彩蛋飞机保留手工价格，`--apply` 不覆盖（与舰船惯例一致） |
| 机枪战力口径 | 按枪管数 | 备用估算器用 `guns.json5` 的 `bulletCount` 统计机枪管数（双联 `US/12.7/2` 计 2），与游戏 `gunDPS` 口径一致；单装飞机不受影响 |

## 比较基准与校准

- 最便宜的作战舰船（小型鱼雷艇）约 $3–5 资金、3–8 秒；最贵的舰船（战列舰）约 $575–1200 资金、60–130 秒。飞机费用应处于舰船费用最下端；重型战略轰炸机（如 B-17 家族）战力与载弹量远超普通战斗机，允许明显更高，`fundsCost` 120 / `timeCost` 50 的上限已覆盖这类机型。
- 首次运行脚本后检查 `combatPower` 列：最低战力飞机的 `rawFunds` 应约等于或略低于 3，最高战力的普通（非 special）飞机 `fundsCost` 应接近但不超过 120；若最低战力被钳制到远低于 3，或最高战力远低于 120（浪费了区间），调整 `scaleFactor` 后重跑。
- 校准公式：`scaleFactor = targetMinFunds / (minCombatPower * typeMultiplier)`，其中 `targetMinFunds ≈ 3`。
- 当前参数：scaleFactor 0.10（初始估算，首次运行后校准）；fallbackScaleFactor 0.30（仅用于 Python 配置估算器，其 `cpEstimate` 尺度低于 Go 图鉴战力）；fallbackFundsMin 3。
- `nation == special` 的彩蛋飞机只显示评估价，不作为写回依据（保留手工价格）。

## 工作流程与验证

评估所有现有飞机：运行 `bash .agents/skills/jutland-evaluate-plane-cost/scripts/evaluate_plane_costs.sh`，检查费用分层（同一型号系列费用连贯，如 A6M2 → A6M3 → A7M2 递增；鱼雷轰炸机 > 俯冲轰炸机 > 战斗机，同年代比较；所有飞机落在 `$3–120 / 3–50s` 区间内），据此更新 `configs/planes.json5` 中每架飞机的 `fundsCost` 与 `timeCost`，移除条目中的 `// TODO 确认资金`、`// TODO 确认时间` 注释。

添加或修改单架飞机：先完成武器、性能等全部配置，再运行脚本获取完整战力数据，从输出中提取该机建议费用填入 `planes.json5` 对应条目。

验证：`make build` 检查 JSON5 解析与字段映射，`go test ./pkg/mission/object/unit/` 检查 `GetPlaneCost` 行为，并手动确认同机型系列费用连贯、类型间梯度合理、费用在钳制范围内。

## 备选方案

若战力分布过窄导致费用分层不明显（大多数飞机的 `fundsCost` 落入同一档），改用基于吨位的公式：

```
rawFunds  = tonnage * typeMultiplier * tonnageScaleFactor
fundsCost = clamp(round(rawFunds), 3, 30)
timeCost  = clamp(round(fundsCost * 0.35 + 2), 3, 10)
```

`tonnageScaleFactor` 需根据实际吨位范围校准，使 `rawFunds` 覆盖 3–30 区间。
