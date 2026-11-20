# pkg/mission/sim

Headless simulation workspace, scenario loading and report rendering.

## Status (P1–P4)

This package currently implements the **report/reporting skeleton** and a synthetic
demo run (`Mock: true`). P2's production observer is in `pkg/mission/observer`;
the headless runner that consumes it lands in P3/P4.

Implemented:

- required `JUTLAND_SIM_HOME` (no default, empty/whitespace is an error);
- `init` lab skeleton, scenario JSON5 parsing/validation;
- run metadata: time, commit, branch, dirty worktree, worktree hash;
- human view: `report.html` (ECharts CDN + static SVG fallback), `summary.md`;
- machine view: `ai/analysis-context.md`, `findings.json`, `metrics.json`,
  `diagnostics.json`, `changes.json`, `llm-prompt.md`;
- `manifest.json`, `result.json`, `replay.json`, `checksums.sha256`;
- append-only `reports/history.jsonl` and `reports/index.html`.

P2 (separate package, already wired into production):

- `pkg/mission/observer` defines `Shot`, `Hit`, `Sink`, `Loss` and a `Recorder`;
- `MissionManager.SetObserver` attaches it; default is nil;
- shot events are emitted from ship/plane fire loops; hit events per target
  (including AoE/friendly fire); sink/loss events from the removal loops;
- observer-disabled benchmark: 10 bullets ~1.4 ns, 0 allocs/op.

P3 hooks (production defaults unchanged; headless runner wires them in P4):

- `unit.SetNowMillis` for carrier takeoff / healing / construction; duel 和 macro runner
  都用 `unit.SimClockMillis` 把它接到 `unit.SimTick()` 上，冷却按模拟拍推进而不是真实耗时；
- `unit.SetPlaneUIDGenerator` for deterministic aircraft IDs;
- `unit.SetRandSource` for seeded fire spread / criticals / AA;
- ordered ship/plane fire and removal loops, ordered `combatIndex` buckets,
  stable `InstructionSet.ExecAll` order, and `SetSyncTargeting` for deterministic plans.

Headless update:

```go
m := manager.New(...)
m.SetSimulationOptions(manager.SimulationOptions{
    Headless: true, SyncTargeting: true, SkipCommands: true,
})
for i := 0; i < ticks; i++ {
    if _, err := m.UpdateHeadless(); err != nil { ... }
}
```

`UpdateHeadless` and the game loop share `advanceSimulationTick`; the game
passes presentation hooks for camera/prewarm/selection, headless passes nil.

## Commands

```bash
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"
go run ./cmd/jutland-sim init
go run ./cmd/jutland-sim demo --open
go run ./cmd/jutland-sim validate demo-report
go run ./cmd/jutland-sim shot shot-f0
go run ./cmd/jutland-duel duel duel-jiujiang
go run ./cmd/jutland-macro macro macro-pearl
go run ./cmd/jutland-sim run --mock demo-report
```

`JUTLAND_SIM_REPORT_DIR` optionally overrides the report directory; `--out` wins.

## Trend (P9, partial)

`jutland-sim trend [--limit N]` reads `reports/history.jsonl` and writes
`trend.html`, `trend.md` and `ai/trend.json`. It shows per-run scorecard counts,
new findings, fidelity and runtime; `index.html` links to it. This is the first
periodic/LLM-analysis building block, not yet a scheduler.

The rendered-frame gate (`make sim-frame-gate` / `jutland-frame`) appends each
result to `reports/frame/history.jsonl` when `JUTLAND_SIM_HOME` is set; trend
adds a 渲染帧门禁 line/card with latest/avg/worst p99 and the pass status, so real
frame-time regressions show up next to the simulation runs.

## Nightly run (optional)

`make sim-nightly` runs the `full-evaluation` suite with `--fail-on-red` (missing runs
or smoke red items fail the job), then the rendered-frame gate (skipped when no
graphical session is available), then `trend`/`analyze`; the failing report is still
generated because trend/analyze always run. It requires `JUTLAND_SIM_HOME`.
Set `SIM_NIGHTLY_SUITE=small` for the quick `tools/sim-suite.sh` matrix, and
`SIM_NIGHTLY_ARCHIVE=1` to move runs older than `SIM_KEEP_RUNS` (default 100) to the
archive after the run.

To schedule it daily at 10:00 local time, install the per-user launchd job
explicitly (nothing is installed by default):

```bash
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"
make sim-schedule-install     # writes ~/Library/LaunchAgents/com.jutland.sim-periodic.plist
make sim-schedule-status
make sim-schedule-uninstall
```

Knobs: `SIM_SCHEDULE_HOUR`/`SIM_SCHEDULE_MINUTE` for the schedule,
`SIM_SCHEDULE_FRAME=0` to skip the frame gate. Output goes to
`$JUTLAND_SIM_HOME/reports/scheduler.log`. `tools/install-scheduler.sh --dry-run`
prints the generated plist without touching the system.

### CI gate

`make sim-gate` (or `jutland-suite suite <manifest> --fail-on-red`) turns a suite into
a pass/fail check: missing runs or smoke red items fail the command, while balance reds
stay informational. To gate an existing summary without re-running the suite:

```bash
jutland-sim evaluation "$JUTLAND_SIM_HOME/reports/suites/<suite>.json" --fail-on-red
```

F1/F2/F3 steps need an Ebiten-capable session, so run the gate on a machine with a
graphical login rather than a headless server.

`jutland-sim analyze [--limit N]` writes `ai/analysis-brief.md` and
`ai/analysis-brief.json`: recurring findings, recurring red scorecards,
per-scenario multi-run means with min~max spread and a small-sample warning,
the latest rendered-frame gate, finding lifecycle (first/last/fixed), recent runs
and an LLM prompt.

`jutland-sim llm-package <run>` copies only `analysis-context.md`,
findings/metrics/diagnostics/changes and `llm-prompt.md` into
`reports/llm/<run>/`, plus a token estimate; HTML/CSV are excluded.

`jutland-sim llm-review <run>` calls DeepSeek's OpenAI-compatible endpoint
(`https://api.deepseek.com/chat/completions`, model `deepseek-chat`) using
`DEEPSEEK_API_KEY`. `--dry-run` writes `ai/llm-request.json` without a network
call. Output: `reports/llm/<run>/llm-review.md` + `llm-review.json` (content,
model, token usage). The endpoint is fixed in code; the key is never logged. It is a deterministic aggregation step; no external LLM
call is made.

## Suite manifests (P9)

`jutland-suite suite <manifest>` runs JSON5 steps under the lab `suites/` dir:

```json5
{ name: "smoke", steps: [
  { mode: "shot", scenario: "shot-f0", runs: 100 },
  { mode: "duel", scenario: "duel-jiujiang", seeds: [1, 3] },
  { mode: "macro", scenario: "macro-pearl", seeds: [1, 3], ticks: 12000 },
]}
```

It writes each run through the normal report pipeline, a suite summary and an
aggregate `*-evaluation.html/md/json` under `reports/suites/`. F1/F2/F3 steps need game resources/Ebiten; F0-only suites run
without a simulation window. `make sim-suite` remains the shell-based smoke loop.

## Local CI / periodic analysis

```bash
make sim-ci          # go test + vet + build
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"
make sim-periodic    # trend + ai/analysis-brief
make sim-watch       # refresh trend/analyze every 10m (Ctrl-C to stop)
make sim-archive     # dry-run old-run archive; remove --dry-run after review
make sim-frame       # rendered-frame p99 (opens a temporary window)
make sim-frame-gate  # same, writes reports/frame/latest.json and fails over budget
make sim-perf        # check newest F3 tick_p99_ms against 16.7ms
make sim-suite       # F0 + F1 + F3 (3 seeds), then trend/analyze
# SIM_MACRO_PARALLEL=1 runs the F3 seeds as separate processes
```

`sim-perf` reads the newest F3 `result.json`; `tick_p99_ms` is headless
simulation wall time, not the rendered game frame. It is a regression proxy.

`make sim-frame` uses `cmd/jutland-frame` to open a temporary 1280x720 window,
run `manager.Update` + `Draw` and report rendered-frame p50/p95/p99/max. It
requires a working window/graphics session and is not a CI gate.

`tools/sim-periodic.sh` is suitable for cron/launchd; it does not schedule itself.

## Full evaluation

```bash
jutland-suite suite full-evaluation
# or regenerate an overview
jutland-sim evaluation reports/suites/full-evaluation-<stamp>.json
```

`testdata/suite-full.json5` is the reference F0/F1/F2/F3 matrix. The aggregate
page shows per-run green/yellow/red counts, findings/diagnostics, shots,
hit%, mission success and enemy ships sunk, plus links to each report. F2 smoke
reds are counted separately from balance reds.

## Previous-run diff (P9, first pass)

Every report now tries to find the previous same-scenario run before writing
history. If found, `result.json.previousDiff` is embedded and the HTML/Markdown
reports show metric deltas, scorecard changes, added findings and gone findings.
The first run of a scenario has no diff section.

## In-game debug recording (P8, partial)

```bash
JUTLAND_DEBUG_RECORD_DIR="$JUTLAND_SIM_HOME/reports/recordings" ./jutland
```

The game writes `<mission>/record-<timestamp>.jsonl` with `shot/hit/sink/loss`
envelopes. Default is off; the game keeps the observer nil. Compare a recording
to a simulation run:

```bash
go run ./cmd/jutland-sim compare-record <record.jsonl> <run-id>
```

This is an event-level calibration aid, not a full replay.

## Archive (P9)

`jutland-sim archive --keep N` moves older `reports/runs/*` into
`reports/archive/<timestamp>/runs/`, writes the archived `history.jsonl`, trims
the live history and rewrites `index.html`. It is opt-in and non-destructive
(no delete); `make sim-archive` always starts with `--dry-run`.

## Penetration matrix (P10, opt-in)

`jutland-sim penetration penetration-127mm` runs a pure geometric AP/HE matrix
(`penetrationMM` vs angle-adjusted armor). It writes the usual F0 report and a
`穿透包线` chart.

Production `HurtBy` has an **opt-in** experimental hook: set
`enablePenetration: true` in `configs/game_settings.json5`. Defaults to false;
when false, damage is byte-for-byte the old `ResolveShip` path.

## Baselines (P8)

A lab-owned baseline is a JSON metric snapshot under `$JUTLAND_SIM_HOME/baselines/`:

```json
{"schemaVersion":1,"name":"f0-127mm-vs-belt","scenario":"f0-127mm-vs-belt","fidelity":"F0","metrics":{"mean_applied_damage":40},"tolerance":{"mean_applied_damage":1}}
```

Pass `--baseline <name-or-path>` to `shot`, `duel` or `macro`; matching metrics get
`baseline`/`delta` in the report, and values outside tolerance add a
`BASELINE-DRIFT-<metric>` finding. The baseline must match scenario/fidelity.

Compare two runs (metrics delta, scorecard changes, findings added/gone):

```bash
go run ./cmd/jutland-sim compare <runA> <runB>
go run ./cmd/jutland-sim compare <runB>       # vs previous same-scenario run
```

Output goes to `reports/compare/<a>__<b>.md` and `.json`.

Snapshot a run without hand-editing JSON:

```bash
go run ./cmd/jutland-sim baseline <run-id> --name f0-auto --tolerance 'mean_applied_damage=1,kill_rate=1'
```

## F0 damage bench (P4)

`shot` mode uses the pure `pkg/mission/object/damage` formula and does not import
Ebiten or load game resources. A scenario defines target HP/armor, base damage,
critical rate, plunge and expected damage:

```bash
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"
cp pkg/mission/sim/testdata/shot-f0.json5 "$JUTLAND_SIM_HOME/scenarios/"
go run ./cmd/jutland-sim shot shot-f0 --runs 100
```

The report is `mock: false` with `fidelity: F0`; F0 only validates damage formulas
and known answers, not hit rate, ship configuration or map/AI behavior.

### F1 real-config duel (P4)

`jutland-duel` runs one single-ship duel through the production `FireAt` /
`updateShotBullets` path using real `ShipMap`/`GunMap`/`BulletMap`:

```bash
cp pkg/mission/sim/testdata/duel-jiujiang.json5 "$JUTLAND_SIM_HOME/scenarios/"
go run ./cmd/jutland-duel duel duel-jiujiang --ticks 1200
```

F1 imports the game's unit/config initialization, so it needs an Ebiten-capable
environment (no game window is opened). `jutland-sim` itself stays resource-free
for F0/report generation. Merging F1 into one binary waits for a resource-free
unit factory.

Multi-ship/fleet sample: `testdata/duel-fleet.json5` (2v1, focus doctrine, 5 seeds).
F1 reports include win rate, exchange ratio, TTK, cost lost and a fleet-survival
chart; AI context remains table-oriented.

`environment.speedMultiplier` and `environment.enablePenetration` are written into the
global game settings for the duration of a run and restored afterwards, so a scenario
can A/B a mechanic (e.g. the opt-in penetration model) without touching
`configs/game_settings.json5`.

With `enablePenetration` on, observer `Hit` events carry the same geometric result
(`penetrationMm`, `armorMm`, `effectiveArmorMm`, `penetrationRatio`, `penetrated`,
`overmatch`), and duel reports add `penetration_hits`, `penetrated_hits` and
`penetration_ratio_mean` so the A/B can be attributed to individual hits.

### F2 combined-arms smoke (P6, partial)

`testdata/duel-carrier.json5` exercises carrier sortie/aircraft weapons,
`testdata/duel-sub.json5` a submerged submarine vs a surface ship,
`testdata/duel-asw.json5` sonar + depth-charge ASW, and
`testdata/duel-aa.json5` AA fire shooting down aircraft.
`testdata/duel-carrier-fog.json5` enables fog. Reports switch to `fidelity: F2`
and add plane/sub metrics. Map support is available via optional `duel.terrain`
ASCII rows.

### F3 full AI-vs-AI mission (P7, partial)

`testdata/macro-pearl.json5` runs `PearlHarbor1941` with both sides on the
production computer controller, full mission state, task/economy and headless
update:

```bash
go run ./cmd/jutland-macro macro macro-pearl --ticks 12000
go run ./cmd/jutland-macro macro macro-pearl --runs 3 --ticks 12000
```

`--runs N` executes seeds sequentially in one process and averages numeric
metrics; multi-seed results add a success-rate scorecard row. The CLI suite uses
separate processes for parallel seeds.

F3 reports add mission status, ship/tonnage/aircraft losses, funds and a fleet
survival chart. The 12k-tick sample ends in a real mission success; single seed,
so it is not yet a statistical balance conclusion. Headless pathfinding runs
inline; F3 same-seed 12000-tick metrics are reproducible (covered by macro tests).

## Report layout

```text
$JUTLAND_SIM_HOME/reports/
├── index.html
├── history.jsonl
├── assets/
└── runs/<run-id>/
    ├── report.html
    ├── summary.md
    ├── result.json
    ├── manifest.json
    ├── checksums.sha256
    ├── replay.json
    ├── ai/
    └── charts/
```

## Notes

- `report.html` uses ECharts 6.1.0 from jsDelivr with SRI; if CDN fails, the static
  SVG fallback is shown via the `no-echarts` CSS class.
- Archived runs will default to a local ECharts asset in a later phase; P1 uses CDN.
- Scenario/golden/baseline/history data belongs in the private lab repo, not this one.
  The game repo only keeps tool code plus small smoke testdata.
- Metrics in `demo` mode are synthetic. Do not use them for balance decisions.
