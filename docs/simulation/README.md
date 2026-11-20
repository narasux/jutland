# 仿真与平衡评估（Simulation Lab）

> 不启动游戏窗口，用**生产战斗代码**跑可复现的舰船/飞机对抗实验，输出报告、趋势、门禁与
> LLM 分析输入。目标不是替代试玩，而是把「改一个数值会不会破坏平衡」变成可量化、可回归的问题。

本文面向使用者：说明这套仿真能提供什么能力、怎么跑、报告怎么看、门禁怎么用。实现细节见
[`pkg/mission/sim/README.md`](../../pkg/mission/sim/README.md)。

---

## 1. 能提供什么

### 1.1 四个保真度（F0–F3）

| 保真度 | 跑什么 | 适合回答 |
|---|---|---|
| **F0** | 纯数学契约：伤害公式、命中率曲线、穿甲几何矩阵、已知答案 | 「公式对不对」「穿深/装甲边界在哪」 |
| **F1** | 真实 `ShipMap/GunMap/BulletMap` + 生产 `FireAt`/`updateShotBullets` 的单舰或编队对射 | 「这艘船/这门炮/这个弹药强不强」 |
| **F2** | 合同作战：舰载机起飞/投弹/防空、潜艇下潜/声呐/深弹/反潜鱼雷、迷雾 | 「空袭效果如何」「反潜链路是否打得通」 |
| **F3** | AI vs AI 完整任务：任务状态、增援/经济、双方 computer controller | 「整场战斗/关卡的结果是什么」 |

### 1.2 可观测的指标

- 逐发 `Shot`、逐次 `Hit`、`Sink`、`Loss` 事件（observer，生产默认关闭、仿真开启）。
- 命中率、应用/有效伤害、过杀、TTK、胜率、交换比、成本损失。
- 兵种归因：飞机射击/命中/击沉、飞机损失、潜艇射击、ASW 射击、被空袭击沉的舰船。
- 穿甲（实验开关打开时）：每发命中的穿深/装甲/比值/是否穿透/过穿，duel 汇总为
  `penetration_hits`、`penetrated_hits`、`penetration_ratio_mean`。
- 性能：headless 每 tick 墙钟 p50/p95/p99（用于回归），以及真实渲染帧 p99。

### 1.3 报告与聚合

每次 run 产出：

```
$JUTLAND_SIM_HOME/reports/runs/<runId>/
├── report.html            # 人看：ECharts（CDN）+ 静态 SVG 回退
├── summary.md             # 关键表 + 记分卡
├── result.json            # 机器可读的完整结果（指标/记分卡/findings/图表数据）
├── replay.json            # 复现所需的命令、seed、commit、场景路径
├── manifest.json          # 元数据与文件列表
├── checksums.sha256
└── ai/
    ├── analysis-context.md   # AI 精简视图
    ├── metrics.json / findings.json / diagnostics.json / changes.json
    └── llm-prompt.md
```

套件还会在 `reports/suites/` 下生成 `*-evaluation.html/md/json`：逐 run 红绿、findings 分级
（bug/data/balance）、穿甲矩阵等。`reports/` 下还有：

- `index.html`：所有 run 的入口；
- `trend.html/md` + `ai/trend.json`：记分卡、耗时、渲染帧门禁趋势；
- `ai/analysis-brief.md/json`：跨 run 汇总（反复出现的 finding、红色指标、场景均值与波动、
  样本不足提示、帧门禁、LLM prompt）。

---

## 2. 快速开始

```bash
# 1) 指定私有实验目录（必填；没有默认值）
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"

# 2) 初始化目录骨架（scenarios/suites/golden/baselines/notes/reports）
go run ./cmd/jutland-sim init

# 3) 把仓库自带示例场景/套件拷进 lab（按需）
cp pkg/mission/sim/testdata/{shot-f0,penetration-127mm,duel-jiujiang,duel-aa,duel-carrier,duel-carrier-fog,duel-sub,duel-asw,duel-fleet,macro-pearl}.json5 \
   "$JUTLAND_SIM_HOME/scenarios/"
cp pkg/mission/sim/testdata/suite-full.json5 "$JUTLAND_SIM_HOME/suites/"

# 4) 生成一份合成 demo 报告（不跑仿真，验证报告链路）
go run ./cmd/jutland-sim demo --open

# 5) 跑一个真实 F0 场景
go run ./cmd/jutland-sim shot shot-f0 --runs 100 --open
```

> `suite-f0.json5` / `suite-full.json5` 是套件，必须放 `suites/`，其余场景文件放 `scenarios/`。
> 仓库 `testdata/` 只是参考副本，真正运行时读取的是 lab 目录下的文件。

常用环境变量：

| 变量 | 作用 |
|---|---|
| `JUTLAND_SIM_HOME` | **必填**，lab 根目录；缺失/空/全空白直接报错 |
| `JUTLAND_SIM_REPORT_DIR` | 可选，覆盖 report 目录 |
| `GOCACHE` | Go 构建缓存（受限环境可设 `/tmp/jutland-gocache`） |
| `DEEPSEEK_API_KEY` | `llm-review` 真实调用时需要 |

---

## 3. 命令参考

### 3.1 四个程序

| 程序 | 用途 | 依赖 |
|---|---|---|
| `cmd/jutland-sim` | F0/纯报告/趋势/基线/评测/LLM/归档 | F0 无需图形 |
| `cmd/jutland-duel` | F1/F2 单舰/编队对射 | 需要 Ebiten 图形会话（不开窗口） |
| `cmd/jutland-macro` | F3 AI vs AI 完整任务 | 需要 Ebiten 图形会话 |
| `cmd/jutland-suite` | 按套件串跑多个场景并出评测 | 取决于步骤 |
| `cmd/jutland-frame` | 真实渲染帧 p99 | 需要图形会话 |

### 3.2 常用命令

```bash
# F0：纯伤害台 / 穿甲矩阵
go run ./cmd/jutland-sim shot shot-f0 --runs 100
go run ./cmd/jutland-sim penetration penetration-127mm

# F1：真实配置对射（可覆盖 seed/runs/ticks/基线）
go run ./cmd/jutland-duel duel duel-jiujiang --ticks 1200
go run ./cmd/jutland-duel duel duel-jiujiang --baseline <baseline-name>

# F3：AI vs AI（--runs 顺序多 seed 聚合、--baseline 对比）
go run ./cmd/jutland-macro macro macro-pearl --seed 3 --ticks 12000
go run ./cmd/jutland-macro macro macro-pearl --runs 3 --ticks 12000 --baseline macro-pearl-1941

# 套件：跑完整评测并出门禁结果
go run ./cmd/jutland-suite suite full-evaluation --fail-on-red

# 对已有套件汇总重算评测（不重跑仿真）
go run ./cmd/jutland-sim evaluation "$JUTLAND_SIM_HOME/reports/suites/<suite>.json" --fail-on-red

# 趋势 / 跨 run 分析 / 比对 / 基线
go run ./cmd/jutland-sim trend --limit 50
go run ./cmd/jutland-sim analyze --limit 20
go run ./cmd/jutland-sim compare <runA> <runB>
go run ./cmd/jutland-sim compare-record <recording.jsonl> <run>
go run ./cmd/jutland-sim baseline <run> --name <name> --tolerance "shots=5000,own_sunk=10"

# 其它
go run ./cmd/jutland-sim validate <scenario.json5>
go run ./cmd/jutland-sim perf --budget-ms 16.7     # 最新 F3 run 的 tick p99 门禁
go run ./cmd/jutland-sim watch --once              # 刷新一次 trend/analyze
go run ./cmd/jutland-sim archive --keep 100 --dry-run
go run ./cmd/jutland-sim llm-package <run>         # 打包 AI 输入
go run ./cmd/jutland-sim llm-review <run> --dry-run # 生成请求 JSON，不联网
```

### 3.3 Make 目标

```bash
make sim-suite        # 小套件：shot + duel + 3 个 macro seed，随后 trend/analyze
make sim-gate         # 完整 full-evaluation + --fail-on-red（CI 门禁）
make sim-nightly      # 帧门禁 + 门禁套件 + 覆盖套件 + trend/analyze
make sim-perf         # 最新 F3 run 的 tick p99 门禁
make sim-frame        # 跑 300 帧渲染帧 p99 并打印结果（需图形会话）
make sim-frame-gate   # 渲染帧门禁并写入 frame/history.jsonl
make sim-periodic     # 只刷新 trend/analyze（可被 cron 调用）
make sim-watch        # 按间隔循环刷新
make sim-archive      # 归档旧 run（默认 dry-run）
make sim-schedule-install / -status / -uninstall  # 安装/查看/卸载每日 launchd 任务
make sim-ci           # go test + vet + build
```

## 4. 场景（scenario）格式

场景是带注释的 JSON5，放在 `$JUTLAND_SIM_HOME/scenarios/`。最小结构：

```json5
{
  schemaVersion: 1,
  name: "duel-jiujiang",
  mode: "duel",              // shot | penetration | duel | macro
  seed: 7,
  runs: 1,
  maxTicks: 1800,
  environment: {
    map: "open_sea",
    fog: false,
    friendlyFire: false,
    speedMultiplier: 1.0,    // 0.5 | 1.0 | 2.0，作用到本局全局配置
    enablePenetration: false,// 按局打开实验穿甲模型（默认关闭）
  },
  duel: {
    shooterShip: "jiujiang",
    targetShip: "jiujiang",
    distance: 4,
    targetInitialHP: 500,
    doctrine: "auto",        // auto | focus
  },
}
```

要点：

- `environment.speedMultiplier` / `enablePenetration` 会在 run 期间写入全局配置，结束后恢复，
  所以不同场景可以互不污染地做 A/B。
- `duel` 支持 `shooterFleet`/`targetFleet`（多舰）、`terrain`（ASCII 行）、`maxTicks`。
- `macro` 支持 `mission`、`playerSide`（P1/P2）、`maxTicks`，以及可选的
  `objective: { surviveTicks: N }`：在引擎「全歼」胜负之外，判断本方能否坚持到第 N 拍
  （仿真层叠加，不改变任务状态）。
- `penetration` 是纯几何矩阵：`penetrationMM`、`armorMin/Max`、`angleMin/Max`、`steps`、
  `referenceArmor/Angle`。

## 5. 套件（suite）与门禁

套件是场景列表，放在 `$JUTLAND_SIM_HOME/suites/`：

```json5
{
  schemaVersion: 1,
  name: "full-evaluation",
  description: "F0/F1/F2/F3 full combat evaluation",
  steps: [
    { mode: "shot", scenario: "shot-f0", runs: 100 },
    { mode: "penetration", scenario: "penetration-127mm" },
    { mode: "duel", scenario: "duel-jiujiang", seeds: [1, 3, 5] },
    { mode: "macro", scenario: "macro-pearl", seeds: [1, 3, 5, 7, 11], ticks: 12000 },
  ],
}
```

`--fail-on-red` 的门禁语义：

- **失败**：缺 run、smoke 红项、`bug` finding；
- **只记录不阻断**：`data` finding（如 F3 超时）、`balance` finding、balance 红项。

这套规则保证「引擎坏了」会拦住提交，「需要重新平衡」不会误伤日常流程。门的实现见
[`evaluation.go`](../../pkg/mission/sim/evaluation.go)。

## 6. 定期运行与夜间任务

`make sim-nightly` 的执行顺序：

1. 渲染帧门禁（先跑，需要干净的图形会话）；
2. 门禁套件 `full-evaluation`（`--fail-on-red`）；
3. 覆盖套件 `SIM_NIGHTLY_EXTRA_SUITES`（默认 `f3-coverage`，不参与门禁）；
4. `trend` + `analyze`；
5. 可选归档（`SIM_NIGHTLY_ARCHIVE=1`）。

可调环境变量：

| 变量 | 默认 | 说明 |
|---|---|---|
| `SIM_NIGHTLY_SUITE` | `full-evaluation` | 门禁套件；`small` 走 `tools/sim-suite.sh` |
| `SIM_NIGHTLY_EXTRA_SUITES` | `f3-coverage` | 非门禁覆盖套件，空格分隔 |
| `SIM_NIGHTLY_FRAME` | `1` | 是否跑渲染帧门禁 |
| `SIM_NIGHTLY_ARCHIVE` | `0` | 是否归档旧 run |
| `SIM_KEEP_RUNS` | `100` | 归档保留数量 |
| `SIM_NIGHTLY_ATTEMPTS` / `SIM_NIGHTLY_RETRY_WAIT` | `3` / `15` | Ebiten 初始化崩溃时的重试次数/间隔 |

安装每日 10:00 的 launchd 任务（**需要显式执行**，默认不会安装）：

```bash
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"
make sim-schedule-install
make sim-schedule-status
make sim-schedule-uninstall
```

## 7. 性能门禁

- **headless tick p99**：`make sim-perf` / `jutland-sim perf --budget-ms 16.7`，读取最新 F3 run 的
  `tick_p99_ms`，预算 16.7ms。它衡量模拟逻辑，不包含绘制/音频。
- **真实渲染帧 p99**：`make sim-frame-gate`，打开临时窗口跑 300 帧，结果追加到
  `reports/frame/history.jsonl` 并进入 trend/分析简报。预算默认 16.7ms；需要可用的图形会话，
  锁屏或初始化失败时夜间任务会跳过并记录日志。

## 8. 基线、对比与复现

- **基线**：`jutland-sim baseline <run> --name <name> --tolerance "metric=tol,..."` 把一次 run 固化为
  参考；之后跑场景时加 `--baseline <name>` 会填充 Delta，并在超容差时产生 `BASELINE-DRIFT` finding。
- **两次 run 对比**：`jutland-sim compare <runA> <runB>`（或只给一个 run，对比同场景上一次）。
- **游戏内录制 vs 仿真**：`jutland-sim compare-record <recording.jsonl> <run>`。
- **复现**：每个 run 的 `replay.json` 记录命令、seed、场景路径、commit；`reports/runs/<runId>` 不可变。
  F3 使用 seeded RNG、稳定飞机 UID、同步寻路与按拍推进的仿真时钟，同 seed 可复现。

## 9. 局限与注意事项

- **F1/F2/F3 需要可用的图形会话**：会初始化 Ebiten（不开窗口）。无 GUI 的 CI 需要 xvfb 等虚拟显示；
  本机锁屏状态下偶发 Ebiten 初始化崩溃，夜间脚本会重试并跳过帧门禁。
- **单 seed 不等于统计结论**：多 seed 只是样本；简报会对 `runs < 3` 的场景给出「样本不足」提示。
- **F2/F3 的 smoke 场景是机制冒烟，不是平衡结论**：例如珍珠港是历史性单方面袭击，正确的评估口径
  是「进攻方能否达成全歼」或「防守方能否坚持 N 拍」，见 `plans/` 下的设计记录与 lab notes。
- **穿甲模型是实验性的**：默认关闭，只有场景显式 `enablePenetration: true` 时才参与伤害；
  公式/数值仍需定稿。
- **AI 收尾行为**：`ASWTest`/`SubmarineTest` 这类给玩家设计的 test 任务，在 AI vs AI 下会因丢失接触
  而僵持到超时（数据类 finding，不阻塞门禁）。
- **报告图表用 CDN ECharts**：离线时自动回退到静态 SVG。

## 10. 扩展方式

- **加场景**：在 `scenarios/` 写 JSON5，`jutland-sim validate` 校验后即可被 suite 引用。
- **加套件步骤**：在 `suites/*.json5` 的 `steps` 里加 `{ mode, scenario, seeds/runs, ticks }`。
- **加基线/门禁**：`baseline` 固化为参考，`--fail-on-red` 接入 CI/夜间任务。
- **加指标/finding**：遵循 `pkg/mission/sim` 现有 `MetricRow` / `Finding{Kind: bug|data|balance}`
  约定；bug 会触发门禁，data/balance 只作文档信号。

---

## 11. 当前参考结果（示例，随实验更新）

截至 2026-11-20 的 lab 状态：

- `full-evaluation`：34 runs，green 204 / yellow 0 / red 0 / findings 0，门禁通过；
- `f3-coverage`：5 runs（ASW/潜艇僵持 + 珍珠港防守目标），yellow 4 / balance red 8 / data finding 4；
- 渲染帧门禁：300 帧 p99 约 2–6ms（预算 16.7ms），受机器负载影响；
- 珍珠港进攻方（P2）5 个 seed 均可达成全歼；防守方（P1）在约 8967–11180 拍被全灭，
  「坚持 12000 拍」目标未达成。
