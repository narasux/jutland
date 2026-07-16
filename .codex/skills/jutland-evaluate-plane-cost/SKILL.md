---
name: jutland-evaluate-plane-cost
description: Evaluate and assign fundsCost and timeCost for Jutland aircraft based on combat power data. Use when adding new planes, rebalancing costs, or when plane costs need to be derived from combat power statistics.
---

# Jutland 飞机花费评估

## 核心规则

- 在仓库根目录工作。先阅读 `AGENTS.md` 并执行 `git status --short`，保留所有无关用户改动。
- 所有项目路径以仓库根目录为基准；脚本不得写死机器绝对路径，中间产物使用动态临时路径。
- 花费评估必须基于实际战力数据（通过脚本提取），不做凭空估算。
- 公式参数在本文件中集中维护；修改公式时必须同步更新脚本与本文档。
- 不删除、不替换、不清理用户未要求处理的现有配置。

## 目标文件

- 飞机配置：`configs/planes.json5`（`fundsCost` 与 `timeCost` 字段）
- Go 辅助函数：`pkg/mission/object/unit/plane.go`（`GetPlaneCost` 函数）
- 战力计算：`pkg/mission/object/combatpower/combatpower.go`（`CalculatePlane` 函数）

## 脚本

- `scripts/evaluate_plane_costs.sh`：编排脚本，负责提取战力数据、套用花费公式、输出建议值。
- `export_plane_cost_data.go`：普通 Go 导出器，加载初始化后的飞机并输出结构化战力数据；不向业务包复制临时测试。
- `scripts/plane_cost_calc.py`：备用配置估算器；当 Go 导出器因本机图形环境触发 Ebiten 初始化失败时使用。追加 `--apply` 可用备用估算器写回 `configs/planes.json5`。

### 使用方式

```bash
bash .codex/skills/jutland-evaluate-plane-cost/scripts/evaluate_plane_costs.sh
```

脚本输出格式为 TSV（制表符分隔），列依次为：`name`、`type`、`nation`、`combatPower`、`tonnage`、`fundsCost`、`timeCost`。

### 脚本内部流程

1. 运行 `go run export_plane_cost_data.go`，加载游戏初始化数据
2. 解析导出的 JSON，提取 `name`、`type`、`nation`、`combatPower`、`tonnage`
3. 套用花费公式计算 `fundsCost` 与 `timeCost`
4. 输出 TSV 结果表

## 花费公式

```
rawFunds  = combatPower * typeMultiplier * scaleFactor
fundsCost = clamp(round(rawFunds), 3, 30)
timeCost  = clamp(round(fundsCost * 0.35 + 2), 3, 10)
```

### 参数说明

| 参数 | 值 | 说明 |
|---|---|---|
| `typeMultiplier` (fighter) | 1.00 | 战斗机单位战力费用最低 |
| `typeMultiplier` (dive_bomber) | 1.15 | 俯冲轰炸机携带炸弹，费用略高 |
| `typeMultiplier` (torpedo_bomber) | 1.30 | 鱼雷轰炸机挂载最重，费用最高 |
| `scaleFactor` | 0.10 | 将战力值映射到 3–30 资金区间的缩放系数（初始估算值） |
| `fundsCost` 范围 | 3–30 | 最便宜飞机不低于 $3，最贵飞机不超过 $30；不做粗粒度分档 |
| `timeCost` 范围 | 3–10 | 建造时间随资金线性增长，钳制在 3–10 秒 |

### 比较基准

- 最便宜的作战舰船（小型鱼雷艇）约 $3–5 资金、3–8 秒建造时间
- 最贵的舰船（战列舰）约 $575–1200 资金、60–130 秒建造时间
- 飞机费用应处于舰船费用的最下端，反映单架飞机的低造价和快速补充

## 校准 scaleFactor

首次运行脚本后，检查输出的 `combatPower` 列：

1. 找到 `combatPower` 最低的飞机，其 `rawFunds` 应约等于或略低于 3
2. 找到 `combatPower` 最高的飞机，其 `fundsCost` 应钳制在 ≤30
3. 若最低战力飞机的 `fundsCost` 远低于 3（被钳制），或最高战力飞机的 `fundsCost` 远低于 30（浪费了区间），则调整 `scaleFactor` 并重新运行

校准公式：`scaleFactor = targetMinFunds / (minCombatPower * typeMultiplier)`，其中 `targetMinFunds ≈ 3`。

### 当前校准状态

- scaleFactor: 0.10（初始估算，首次运行后校准）
- fallbackScaleFactor: 0.30（仅用于 Python 配置估算器；其 `cpEstimate` 尺度低于 Go 图鉴战力）
- fallbackFundsMin: 3

## 工作流程

### 评估所有现有飞机

1. 运行 `bash .codex/skills/jutland-evaluate-plane-cost/scripts/evaluate_plane_costs.sh`
2. 检查输出表，确认费用分层合理：
   - 同一型号系列的费用应连贯（如 A6M2 → A6M3 → A7M2 递增）
   - 鱼雷轰炸机费用 > 俯冲轰炸机 > 战斗机（同年代比较）
   - 无任何飞机超出 $3–30 / 3–10s 范围
3. 根据脚本输出更新 `configs/planes.json5` 中每架飞机的 `fundsCost` 与 `timeCost`
4. 移除 `planes.json5` 中各飞机条目中的 `// TODO 确认资金` 与 `// TODO 确认时间` 注释
5. 运行 `make build` 验证配置解析正确

### 添加或修改单架飞机时

1. 先完成飞机的所有配置（武器、性能参数等）
2. 运行脚本获取新的完整战力数据
3. 从脚本输出中提取该飞机的建议费用
4. 填写到 `planes.json5` 对应条目

## 验证

- `make build`：验证 JSON5 解析无误、字段映射正确
- `go test ./pkg/mission/object/unit/`：验证 `GetPlaneCost` 函数行为
- 手动检查：同机型系列费用连贯，类型间费用梯度合理，费用在钳制范围内

## 备选方案

若战力分布过窄导致费用分层不明显（大多数飞机的 `fundsCost` 落入同一档），改用基于吨位的公式：

```
rawFunds  = tonnage * typeMultiplier * tonnageScaleFactor
fundsCost = clamp(round(rawFunds), 3, 30)
timeCost  = clamp(round(fundsCost * 0.35 + 2), 3, 10)
```

`tonnageScaleFactor` 需根据实际吨位范围校准，使 `rawFunds` 覆盖 3–30 区间。
