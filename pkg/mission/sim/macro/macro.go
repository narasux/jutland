// Package macro runs an AI-vs-AI full mission through the production headless
// update path. It is separate from pkg/mission/sim because mission/config
// initialization still pulls Ebiten resources.
package macro

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/manager"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/observer"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/state"
)

// RunMacro executes one full AI-vs-AI mission and returns a real F3 result.
func RunMacro(sc *sim.Scenario, startedAt time.Time) (*sim.Result, error) {
	if sc == nil || sc.Macro == nil {
		return nil, fmt.Errorf("macro scenario is required")
	}
	if sc.Mode != sim.ModeMacro {
		return nil, fmt.Errorf("scenario mode is %q, want macro", sc.Mode)
	}
	restoreEnv := sim.ApplyEnvironment(sc)
	defer restoreEnv()
	spec := sc.Macro
	oldTick := objUnit.SimTick()
	objUnit.SetSimTick(0)
	defer objUnit.SetSimTick(oldTick)
	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(sc.Seed)))
	defer restoreRand()
	restoreClock := objUnit.SetNowMillis(objUnit.SimClockMillis(1_700_000_000_000))
	defer restoreClock()
	uidSeq := 0
	restoreUID := objUnit.SetPlaneUIDGenerator(func() string {
		uidSeq++
		return fmt.Sprintf("%s/macro/%d/plane-%d", spec.Mission, sc.Seed, uidSeq)
	})
	defer restoreUID()

	m := manager.New(spec.Mission, faction.Side(spec.PlayerSide))
	m.SetSimulationOptions(manager.SimulationOptions{Headless: true, SyncTargeting: true, SkipCommands: false})
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)
	start := time.Now()
	status := state.MissionRunning
	tickDurations := make([]time.Duration, 0, spec.MaxTicks)
	for tick := 0; tick < spec.MaxTicks; tick++ {
		tickStart := time.Now()
		var err error
		status, err = m.UpdateHeadless()
		tickDurations = append(tickDurations, time.Since(tickStart))
		if err != nil {
			return nil, err
		}
		if status != state.MissionRunning {
			break
		}
	}
	duration := time.Since(start)
	ms := m.State()

	shots := len(recorder.Shots)
	hits := len(recorder.Hits)
	var friendly int
	var damage float64
	for _, hit := range recorder.Hits {
		if hit.FriendlyFire {
			friendly++
		}
		damage += hit.AppliedDamage
	}
	ownSunk, enemySunk, ownTonnage, enemyTonnage := 0, 0, 0.0, 0.0
	// 空袭归因：只统计击杀者是飞机的沉没，用来区分舰载机与水面火力的战果。
	ownSunkByAir, enemySunkByAir := 0, 0
	for _, sink := range recorder.Sinks {
		if sink.TargetPlayer == faction.HumanAlpha {
			ownSunk++
			ownTonnage += sink.Tonnage
			if sink.KillerIsPlane {
				ownSunkByAir++
			}
		} else {
			enemySunk++
			enemyTonnage += sink.Tonnage
			if sink.KillerIsPlane {
				enemySunkByAir++
			}
		}
	}
	// 空中火力按阵营拆分：飞机是否真的开火，比只看沉没数更能定位空袭问题。
	ownPlaneShots, ownPlaneHits, enemyPlaneShots, enemyPlaneHits := 0, 0, 0, 0
	for _, shot := range recorder.Shots {
		if !shot.ShooterIsPlane {
			continue
		}
		if shot.ShooterPlayer == faction.HumanAlpha {
			ownPlaneShots++
		} else {
			enemyPlaneShots++
		}
	}
	for _, hit := range recorder.Hits {
		if !hit.ShooterIsPlane {
			continue
		}
		if hit.ShooterPlayer == faction.HumanAlpha {
			ownPlaneHits++
		} else {
			enemyPlaneHits++
		}
	}
	ownPlanesLost, enemyPlanesLost := 0, 0
	for _, loss := range recorder.Losses {
		if loss.TargetPlayer == faction.HumanAlpha {
			ownPlanesLost++
		} else {
			enemyPlanesLost++
		}
	}
	hitRate := 0.0
	if shots > 0 {
		hitRate = float64(hits) / float64(shots)
	}
	tickP50, tickP95, tickP99, tickMax := durationPercentiles(tickDurations)
	missionSuccess, missionFailed, missionRunning := 0.0, 0.0, 0.0
	switch status {
	case state.MissionSuccess:
		missionSuccess = 1
	case state.MissionFailed:
		missionFailed = 1
	default:
		missionRunning = 1
	}
	initialOwn := ownSunk + countPlayerShips(ms, faction.HumanAlpha)
	initialEnemy := enemySunk + countPlayerShips(ms, faction.ComputerAlpha)
	scorecard := []sim.ScorecardItem{
		{Dimension: "任务", Metric: "status", Value: string(status), Target: "success", Status: missionStatus(status)},
		{
			Dimension: "损失",
			Metric:    "own / enemy ships",
			Value:     fmt.Sprintf("%d / %d", ownSunk, enemySunk),
			Target:    "<= enemy",
			Status:    lossStatus(ownSunk, enemySunk),
		},
		{
			Dimension: "吨位",
			Metric:    "own / enemy lost",
			Value:     fmt.Sprintf("%.0f / %.0f", ownTonnage, enemyTonnage),
			Target:    "<= enemy",
			Status:    lossStatus(int(ownTonnage), int(enemyTonnage)),
		},
		{
			Dimension: "样本",
			Metric:    "ticks",
			Value:     fmt.Sprintf("%d", m.SimulationTick()),
			Target:    "<= maxTicks",
			Status:    tickStatus(m.SimulationTick(), spec.MaxTicks),
		},
		{
			Dimension: "友伤",
			Metric:    "friendly hits",
			Value:     fmt.Sprintf("%d", friendly),
			Target:    "0",
			Status:    zeroStatus(friendly),
		},
		{
			Dimension: "性能",
			Metric:    "p99 tick",
			Value:     fmt.Sprintf("%.3f ms", tickP99),
			Target:    "<= 16.7 ms",
			Status:    perfStatus(tickP99, 16.7),
		},
	}
	metrics := []sim.MetricRow{
		{Group: "任务", Name: "ticks", Value: float64(m.SimulationTick()), Unit: "ticks"},
		{Group: "任务", Name: "runs", Value: 1, Unit: "runs"},
		{Group: "任务", Name: "mission_success", Value: missionSuccess, Unit: "runs"},
		{Group: "任务", Name: "mission_failed", Value: missionFailed, Unit: "runs"},
		{Group: "任务", Name: "mission_running", Value: missionRunning, Unit: "runs"},
		{Group: "命中", Name: "shots", Value: float64(shots), Unit: "shots"},
		{Group: "命中", Name: "hits", Value: float64(hits), Unit: "hits"},
		{Group: "命中", Name: "hit_rate", Value: round3(hitRate * 100), Unit: "%"},
		{Group: "伤害", Name: "applied_damage", Value: round3(damage), Unit: "HP"},
		{Group: "损失", Name: "own_sunk", Value: float64(ownSunk), Unit: "ships"},
		{Group: "损失", Name: "enemy_sunk", Value: float64(enemySunk), Unit: "ships"},
		{Group: "损失", Name: "own_tonnage", Value: round3(ownTonnage), Unit: "t"},
		{Group: "损失", Name: "enemy_tonnage", Value: round3(enemyTonnage), Unit: "t"},
		{Group: "空中", Name: "own_aircraft_losses", Value: float64(ownPlanesLost), Unit: "planes"},
		{Group: "空中", Name: "enemy_aircraft_losses", Value: float64(enemyPlanesLost), Unit: "planes"},
		{Group: "空中", Name: "own_plane_shots", Value: float64(ownPlaneShots), Unit: "shots"},
		{Group: "空中", Name: "own_plane_hits", Value: float64(ownPlaneHits), Unit: "hits"},
		{Group: "空中", Name: "enemy_plane_shots", Value: float64(enemyPlaneShots), Unit: "shots"},
		{Group: "空中", Name: "enemy_plane_hits", Value: float64(enemyPlaneHits), Unit: "hits"},
		{Group: "空中", Name: "own_sunk_by_aircraft", Value: float64(ownSunkByAir), Unit: "ships"},
		{Group: "空中", Name: "enemy_sunk_by_aircraft", Value: float64(enemySunkByAir), Unit: "ships"},
		{Group: "经济", Name: "funds", Value: float64(ms.Player.CurFunds), Unit: "funds"},
		{Group: "经济", Name: "funds_spent", Value: float64(ms.Stats.FundsSpent), Unit: "funds"},
		{Group: "经济", Name: "funds_earned", Value: float64(ms.Stats.FundsEarned), Unit: "funds"},
		{Group: "安全", Name: "friendly_hits", Value: float64(friendly), Unit: "hits"},
		{Group: "性能", Name: "tick_p50_ms", Value: tickP50, Unit: "ms"},
		{Group: "性能", Name: "tick_p95_ms", Value: tickP95, Unit: "ms"},
		{Group: "性能", Name: "tick_p99_ms", Value: tickP99, Unit: "ms"},
		{Group: "性能", Name: "tick_max_ms", Value: tickMax, Unit: "ms"},
		{Group: "初始", Name: "own_ships", Value: float64(initialOwn), Unit: "ships"},
		{Group: "初始", Name: "enemy_ships", Value: float64(initialEnemy), Unit: "ships"},
	}
	findings := make([]sim.Finding, 0)
	if shots == 0 {
		findings = append(findings, sim.Finding{
			ID: "F3-NO-SHOTS", Kind: "bug", Severity: "high", Status: "new",
			Title: "F3 完整战斗全程 0 射击", Summary: "AI 对 AI 没有任何 Shot 事件。",
			Confidence: 0.95,
		})
	}
	if status == state.MissionRunning {
		findings = append(findings, sim.Finding{
			ID: "F3-TIMEOUT", Kind: "data", Severity: "medium", Status: "new",
			Title: "F3 在 maxTicks 内未结束", Summary: fmt.Sprintf("ticks=%d，maxTicks=%d。", m.SimulationTick(), spec.MaxTicks),
			Confidence: 0.9,
		})
	}
	if friendly > 0 {
		findings = append(findings, sim.Finding{
			ID: "F3-FRIENDLY-FIRE", Kind: "bug", Severity: "high", Status: "new",
			Title: "F3 出现友军伤害", Summary: fmt.Sprintf("%d 次命中被标记为友伤。", friendly),
			Confidence: 0.98,
		})
	}
	return &sim.Result{
		SchemaVersion: 1,
		Mock:          false,
		Fidelity:      "F3",
		Scenario:      *sc,
		Seed:          sc.Seed,
		Runs:          1,
		StartedAt:     startedAt,
		DurationMS:    duration.Milliseconds(),
		Scorecard:     scorecard,
		Metrics:       metrics,
		Findings:      findings,
		Charts: sim.Charts{
			Survival: survivalCurve(recorder.Sinks, initialOwn, m.SimulationTick()),
		},
		DataQuality: []string{
			"F3 完整战斗：生产任务状态 + 双方 computer controller + 任务/增援/经济。",
			"无人类输入、无 UI/相机/音频；单 seed，不是统计结论。",
			"任务名与玩家侧来自场景；同 seed 走 seeded RNG/稳定 ID/同步 targeting。",
			"headless 寻路为同步执行；F3 同 seed 12000 tick 的指标/记分卡/ Findings 已可复现。",
			"tick_p99_ms 是 headless 模拟 tick 的墙钟 p99，不是游戏帧时间；用于回归门禁，不包含绘制/音频。",
		},
	}, nil
}

func countPlayerShips(ms *state.MissionState, player faction.Player) int {
	count := 0
	for _, ship := range ms.Arena.Ships {
		if ship.BelongPlayer == player {
			count++
		}
	}
	return count
}

func survivalCurve(sinks []observer.Sink, initial int, maxTick int64) []sim.SurvivalPoint {
	if initial <= 0 {
		return nil
	}
	points := make([]sim.SurvivalPoint, 0, int(maxTick)/60+2)
	for tick := int64(0); tick <= maxTick; tick += 60 {
		alive := initial
		for _, sink := range sinks {
			if sink.TargetPlayer == faction.HumanAlpha && sink.Tick <= tick {
				alive--
			}
		}
		points = append(points, sim.SurvivalPoint{
			Minute: int(tick / 60), Probability: round3(float64(max(alive, 0)) / float64(initial)),
		})
	}
	return points
}

func missionStatus(status state.MissionStatus) string {
	switch status {
	case state.MissionSuccess:
		return "green"
	case state.MissionFailed:
		return "red"
	default:
		return "yellow"
	}
}

func lossStatus(own, enemy int) string {
	if own <= enemy {
		return "green"
	}
	return "red"
}

func tickStatus(ticks int64, max int) string {
	if ticks <= int64(max) {
		return "green"
	}
	return "yellow"
}

func durationPercentiles(durations []time.Duration) (p50, p95, p99, maxMS float64) {
	if len(durations) == 0 {
		return 0, 0, 0, 0
	}
	sorted := append([]time.Duration(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	pick := func(q float64) float64 {
		idx := int(float64(len(sorted)-1) * q)
		return float64(sorted[idx].Microseconds()) / 1000
	}
	return pick(0.50), pick(0.95), pick(0.99), float64(sorted[len(sorted)-1].Microseconds()) / 1000
}

func perfStatus(p99, budget float64) string {
	if p99 <= budget {
		return "green"
	}
	return "red"
}

func zeroStatus(value int) string {
	if value == 0 {
		return "green"
	}
	return "red"
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}
