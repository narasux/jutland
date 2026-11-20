// Package duel runs real-config F1 engagements through the production headless
// update path. It is separate from pkg/mission/sim because unit/config
// initialization still pulls Ebiten resources.
package duel

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/manager"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/observer"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

type trial struct {
	recorder   *observer.Recorder
	status     state.MissionStatus
	ownStart   int
	enemyStart int
	lastTick   int64
	costByUID  map[string]float64
}

// RunDuel executes one or more deterministic F1 fleet duels and aggregates
// outcomes. Each run uses seed+run; unit/RNG/clock hooks are restored after.
func RunDuel(sc *sim.Scenario, startedAt time.Time) (*sim.Result, error) {
	if sc == nil || sc.Duel == nil {
		return nil, fmt.Errorf("duel scenario is required")
	}
	if sc.Mode != sim.ModeDuel {
		return nil, fmt.Errorf("scenario mode is %q, want duel", sc.Mode)
	}
	spec := sc.Duel
	if spec.MaxTicks <= 0 {
		spec.MaxTicks = sc.MaxTicks
	}
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	restoreEnv := sim.ApplyEnvironment(sc)
	defer restoreEnv()
	runs := max(sc.Runs, 1)
	start := time.Now()
	trials := make([]trial, 0, runs)
	for run := 0; run < runs; run++ {
		t, err := runDuelTrial(sc, spec, sc.Seed+int64(run))
		if err != nil {
			return nil, fmt.Errorf("run %d: %w", run, err)
		}
		trials = append(trials, t)
	}
	duration := time.Since(start)

	var shots, hits, friendly, ownSinks, enemySinks, wins, losses int
	var planeShots, planeHits, planeKills, aircraftLosses, subShots, aswShots int
	var penetrationHits, penetratedHits int
	var penetrationRatioSum float64
	var applied, effective, overkill float64
	var ownLostCost, enemyLostCost float64
	ttks := make([]float64, 0, runs)
	for _, t := range trials {
		shots += len(t.recorder.Shots)
		hits += len(t.recorder.Hits)
		for _, shot := range t.recorder.Shots {
			if shot.ShooterIsPlane {
				planeShots++
			}
			if shot.ShooterType == string(objUnit.ShipTypeSubmarine) {
				subShots++
			}
			if shot.AmmoType == string(objBullet.TypeDepthCharge) {
				aswShots++
			}
		}
		for _, hit := range t.recorder.Hits {
			if hit.ShooterIsPlane {
				planeHits++
			}
		}
		aircraftLosses += len(t.recorder.Losses)
		for _, hit := range t.recorder.Hits {
			if hit.FriendlyFire {
				friendly++
			}
			applied += hit.AppliedDamage
			effective += hit.EffectiveDamage
			overkill += hit.Overkill
			// ArmorMM 非零代表这条命中带穿甲信息（observer 只在开关打开时回填）。
			if hit.ArmorMM > 0 {
				penetrationHits++
				penetrationRatioSum += hit.PenetrationRatio
				if hit.Penetrated {
					penetratedHits++
				}
			}
		}
		for _, sink := range t.recorder.Sinks {
			cost := t.costByUID[sink.TargetUID]
			if sink.TargetPlayer == faction.HumanAlpha {
				ownSinks++
				ownLostCost += cost
			} else {
				enemySinks++
				enemyLostCost += cost
			}
			if sink.KillerIsPlane {
				planeKills++
			}
		}
		switch t.status {
		case state.MissionSuccess:
			wins++
		case state.MissionFailed:
			losses++
		}
		if len(t.recorder.Sinks) > 0 && len(t.recorder.Shots) > 0 {
			firstShot := t.recorder.Shots[0].Tick
			for _, shot := range t.recorder.Shots {
				if shot.Tick < firstShot {
					firstShot = shot.Tick
				}
			}
			ttk := t.recorder.Sinks[0].Tick - firstShot
			if ttk >= 0 {
				ttks = append(ttks, float64(ttk))
			}
		}
	}
	hitRate := 0.0
	if shots > 0 {
		hitRate = float64(hits) / float64(shots)
	}
	winRate := float64(wins) / float64(runs)
	exchange := 0.0
	if ownLostCost > 0 {
		exchange = enemyLostCost / ownLostCost
	} else if enemyLostCost > 0 {
		exchange = 999
	}
	ttkMedian := medianFloat(ttks)

	scorecard := []sim.ScorecardItem{
		{
			Dimension: "胜率",
			Metric:    "shooter win",
			Value:     fmt.Sprintf("%.1f%%", winRate*100),
			Target:    ">= 50%",
			Status:    ratioStatus(winRate, 0.5),
		},
		{
			Dimension: "交换比",
			Metric:    "enemy cost / own cost lost",
			Value:     formatRatio(exchange),
			Target:    ">= 0.8",
			Status:    ratioStatus(exchange/1.0, 0.8),
		},
		{
			Dimension: "TTK",
			Metric:    "first sink",
			Value:     fmt.Sprintf("%.0f ticks", ttkMedian),
			Target:    "<= maxTicks",
			Status:    ttkStatus(ttkMedian, float64(spec.MaxTicks)),
		},
		{
			Dimension: "命中",
			Metric:    "hit rate",
			Value:     fmt.Sprintf("%.1f%%", hitRate*100),
			Target:    "n/a",
			Status:    "green",
		},
		{
			Dimension: "友伤",
			Metric:    "friendly hits",
			Value:     fmt.Sprintf("%d", friendly),
			Target:    "0",
			Status:    zeroStatus(friendly),
		},
		{Dimension: "样本", Metric: "runs", Value: fmt.Sprintf("%d", runs), Target: ">= 1", Status: "green"},
	}
	if planeShots > 0 {
		scorecard = append(scorecard, sim.ScorecardItem{
			Dimension: "空中", Metric: "plane shots", Value: fmt.Sprintf("%d", planeShots),
			Target: "> 0", Status: "green",
		})
	}
	metrics := []sim.MetricRow{
		{Group: "样本", Name: "runs", Value: float64(runs), Unit: "runs"},
		{Group: "命中", Name: "shots", Value: float64(shots), Unit: "shots"},
		{Group: "命中", Name: "hits", Value: float64(hits), Unit: "hits"},
		{Group: "命中", Name: "hit_rate", Value: round3(hitRate * 100), Unit: "%"},
		{Group: "伤害", Name: "applied_damage", Value: round3(applied), Unit: "HP"},
		{Group: "伤害", Name: "effective_damage", Value: round3(effective), Unit: "HP"},
		{Group: "伤害", Name: "overkill", Value: round3(overkill), Unit: "HP"},
		{Group: "结果", Name: "wins", Value: float64(wins), Unit: "runs"},
		{Group: "结果", Name: "losses", Value: float64(losses), Unit: "runs"},
		{Group: "结果", Name: "own_sunk", Value: float64(ownSinks), Unit: "ships"},
		{Group: "结果", Name: "enemy_sunk", Value: float64(enemySinks), Unit: "ships"},
		{Group: "成本", Name: "own_lost_cost", Value: round3(ownLostCost), Unit: "funds"},
		{Group: "成本", Name: "enemy_lost_cost", Value: round3(enemyLostCost), Unit: "funds"},
		{Group: "成本", Name: "exchange_ratio", Value: round3(exchange), Unit: "x"},
		{Group: "节奏", Name: "ttk_median_ticks", Value: ttkMedian, Unit: "ticks"},
		{Group: "安全", Name: "friendly_hits", Value: float64(friendly), Unit: "hits"},
		{Group: "空中", Name: "plane_shots", Value: float64(planeShots), Unit: "shots"},
		{Group: "空中", Name: "plane_hits", Value: float64(planeHits), Unit: "hits"},
		{Group: "空中", Name: "plane_kills", Value: float64(planeKills), Unit: "ships"},
		{Group: "空中", Name: "aircraft_losses", Value: float64(aircraftLosses), Unit: "planes"},
		{Group: "水下", Name: "sub_shots", Value: float64(subShots), Unit: "shots"},
		{Group: "水下", Name: "asw_shots", Value: float64(aswShots), Unit: "shots"},
	}
	// 穿甲指标只在开关打开、命中带穿甲信息时出现，避免普通对射报告里全是 0。
	if penetrationHits > 0 {
		metrics = append(
			metrics,
			sim.MetricRow{
				Group: "穿甲", Name: "penetration_hits", Value: float64(penetrationHits), Unit: "hits",
			},
			sim.MetricRow{
				Group: "穿甲", Name: "penetrated_hits", Value: float64(penetratedHits), Unit: "hits",
			},
			sim.MetricRow{
				Group: "穿甲", Name: "penetration_ratio_mean",
				Value: round3(penetrationRatioSum / float64(penetrationHits)), Unit: "x",
			},
		)
	}
	findings := make([]sim.Finding, 0)
	prefix := "F1"
	if fleetHasAircraft(spec.ShooterFleet) || fleetHasAircraft(spec.TargetFleet) ||
		fleetHasSubmarine(spec.ShooterFleet) || fleetHasSubmarine(spec.TargetFleet) ||
		sc.Environment.Fog {
		prefix = "F2"
	}
	if shots == 0 {
		findings = append(findings, sim.Finding{
			ID: prefix + "-NO-SHOTS", Kind: "bug", Severity: "high", Status: "new",
			Title: "对射全程 0 射击", Summary: "双方进入射程但没有任何 Shot 事件。",
			Confidence: 0.95,
		})
	}
	if enemySinks == 0 && shots > 0 {
		findings = append(findings, sim.Finding{
			ID: prefix + "-CENSORED", Kind: "data", Severity: "medium", Status: "new",
			Title: "在 maxTicks 内未击沉敌方", Summary: fmt.Sprintf("runs=%d shots=%d，maxTicks=%d。", runs, shots, spec.MaxTicks),
			Confidence: 0.9,
		})
	}
	if friendly > 0 {
		findings = append(findings, sim.Finding{
			ID: "F1-FRIENDLY-FIRE", Kind: "bug", Severity: "high", Status: "new",
			Title: "F1 出现友军伤害", Summary: fmt.Sprintf("%d 次命中被标记为友伤。", friendly),
			Confidence: 0.98,
		})
	}

	hasAir := fleetHasAircraft(spec.ShooterFleet) || fleetHasAircraft(spec.TargetFleet)
	hasSub := fleetHasSubmarine(spec.ShooterFleet) || fleetHasSubmarine(spec.TargetFleet)
	fidelity := "F1"
	dataQuality := []string{
		"F1 真实配置平台对射：使用生产 ShipMap/GunMap/BulletMap、FireAt 与 updateShotBullets。",
		fmt.Sprintf("doctrine=%s；无 AI 指令、无飞机/潜艇/迷雾/地图机动。", spec.Doctrine),
		"结论只用于单舰/编队/武器对比，不等于完整战斗。",
		"exchange_ratio=999 表示 shooter 方零损失（无穷大）。",
	}
	if hasAir || hasSub || sc.Environment.Fog {
		fidelity = "F2"
		dataQuality = []string{
			"F2 合同作战 smoke：含舰载机/潜艇/迷雾之一，走生产起飞、防空、投弹与探测路径。",
			fmt.Sprintf("doctrine=%s；无 AI 指令、无任务/增援/经济。", spec.Doctrine),
			"结论只用于合同作战 smoke 与归因，不等于完整战斗平衡。",
			"exchange_ratio=999 表示 shooter 方零损失（无穷大）。",
		}
	}
	if hasAir && planeShots == 0 && aircraftLosses == 0 {
		findings = append(findings, sim.Finding{
			ID: "F2-AIR-NO-SORTIE", Kind: "bug", Severity: "high", Status: "new",
			Title: "F2 舰载机未开火", Summary: "场景包含航母舰载机，但全程没有飞机 Shot 事件（可能未出击或未进入投弹/射击）。",
			Confidence: 0.9,
		})
	}

	return &sim.Result{
		SchemaVersion: 1,
		Mock:          false,
		Fidelity:      fidelity,
		Scenario:      *sc,
		Seed:          sc.Seed,
		Runs:          runs,
		StartedAt:     startedAt,
		DurationMS:    duration.Milliseconds(),
		Scorecard:     scorecard,
		Metrics:       metrics,
		Findings:      findings,
		Charts: sim.Charts{
			Survival: survivalCurve(trials, spec.MaxTicks),
		},
		DataQuality: dataQuality,
	}, nil
}

func runDuelTrial(sc *sim.Scenario, spec *sim.DuelSpec, seed int64) (trial, error) {
	oldTick := objUnit.SimTick()
	objUnit.SetSimTick(0)
	defer objUnit.SetSimTick(oldTick)

	restoreRand := objUnit.SetRandSource(rand.New(rand.NewSource(seed)))
	defer restoreRand()
	restoreClock := objUnit.SetNowMillis(objUnit.SimClockMillis(1_700_000_000_000))
	defer restoreClock()
	uidSeq := 0
	restoreUID := objUnit.SetPlaneUIDGenerator(func() string {
		uidSeq++
		return fmt.Sprintf("%s/%d/plane-%d", sc.Name, seed, uidSeq)
	})
	defer restoreUID()

	shooterGroups := expandFleet(spec.ShooterFleet)
	targetGroups := expandFleet(spec.TargetFleet)
	totalShips := len(shooterGroups) + len(targetGroups)
	width := int(spec.Distance) + 40
	height := max(40, totalShips*4+10)
	rows := make(mapcfg.MapData, height)
	if len(spec.Terrain) > 0 {
		height = len(spec.Terrain)
		width = len(spec.Terrain[0])
		rows = append(rows, spec.Terrain...)
	} else {
		for i := 0; i < height; i++ {
			rows[i] = strings.Repeat(".", width)
		}
	}
	ms := &state.MissionState{
		Core: state.MissionCoreState{
			MissionStatus: state.MissionRunning,
			MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{
				Name: "duel-water", Map: rows, Width: width, Height: height,
			}},
		},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{},
			Planes:            map[string]*objUnit.Plane{},
			ShipUidGenerators: map[faction.Player]*objUnit.ShipUidGenerator{},
		},
		UI: state.MissionUIState{GameMarks: map[objMark.ID]*objMark.Mark{}},
	}
	ms.Stats = state.NewBattleStats()
	if sc.Environment.Fog {
		ms.Core.FogOfWar = true
		humanVision := state.NewFactionVision(width, height)
		computerVision := state.NewFactionVision(width, height)
		humanVision.RendersShade = false
		computerVision.RendersShade = false
		ms.Player.Visions = map[faction.Player]*state.FactionVision{
			faction.HumanAlpha:    humanVision,
			faction.ComputerAlpha: computerVision,
		}
	}
	ms.Arena.ShipUidGenerators[faction.HumanAlpha] = objUnit.NewShipUidGenerator(faction.HumanAlpha)
	ms.Arena.ShipUidGenerators[faction.ComputerAlpha] = objUnit.NewShipUidGenerator(faction.ComputerAlpha)

	shooter := placeFleet(ms, shooterGroups, spec, 10, spec.ShooterRotation, faction.HumanAlpha, height)
	target := placeFleet(
		ms,
		targetGroups,
		spec,
		10+int(spec.Distance),
		spec.TargetRotation,
		faction.ComputerAlpha,
		height,
	)
	if spec.ShooterInitialHP > 0 {
		for _, ship := range shooter {
			ship.CurHP = spec.ShooterInitialHP
		}
	}
	if spec.TargetInitialHP > 0 {
		for _, ship := range target {
			ship.CurHP = spec.TargetInitialHP
		}
	}
	if spec.Doctrine == "focus" {
		applyFocusDoctrine(shooter, target)
		applyFocusDoctrine(target, shooter)
	}
	// Give carriers a concrete strike target so aircraft sortie even when the
	// background targeting queue is not populated in a synthetic headless state.
	assignCarrierTargets(shooter, target)
	assignCarrierTargets(target, shooter)

	m := manager.NewWithState(ms)
	m.SetSimulationOptions(manager.SimulationOptions{Headless: true, SyncTargeting: true, SkipCommands: true})
	recorder := &observer.Recorder{}
	m.SetObserver(recorder)
	if os.Getenv("DUEL_DEBUG") != "" {
		for _, ship := range ms.Arena.Ships {
			fmt.Printf(
				"DBG pre %s %s target=%s maxSub=%.1f sonar=%.1f submerged=%t\n",
				ship.Uid,
				ship.Name,
				ship.AttackTarget,
				ship.Weapon.MaxToSubmergedRange,
				ship.SonarRange(),
				ship.Submerged(),
			)
		}
	}

	maxTick := int64(0)
	for tick := 0; tick < spec.MaxTicks; tick++ {
		status, err := m.UpdateHeadless()
		if err != nil {
			return trial{}, err
		}
		if os.Getenv("DUEL_DEBUG") != "" && tick < 5 {
			for _, ship := range ms.Arena.Ships {
				fmt.Printf(
					"DBG t=%d %s hp=%.0f target=%s det=%v shots=%d\n",
					tick,
					ship.Name,
					ship.CurHP,
					ship.AttackTarget,
					ship.Submarine.DetectedBy,
					len(recorder.Shots),
				)
			}
		}
		maxTick = int64(tick + 1)
		if status != state.MissionRunning {
			break
		}
	}
	for _, sink := range recorder.Sinks {
		if sink.Tick > maxTick {
			maxTick = sink.Tick
		}
	}
	return trial{
		recorder: recorder, status: ms.Core.MissionStatus,
		ownStart: len(shooter), enemyStart: len(target),
		lastTick: maxTick, costByUID: costMap(shooter, target),
	}, nil
}

func expandFleet(groups []sim.DuelShip) []sim.DuelShip {
	out := make([]sim.DuelShip, 0)
	for _, g := range groups {
		for i := 0; i < max(g.Count, 1); i++ {
			out = append(out, g)
		}
	}
	return out
}

func placeFleet(
	ms *state.MissionState, groups []sim.DuelShip, spec *sim.DuelSpec,
	x int, rotation float64, player faction.Player, height int,
) []*objUnit.BattleShip {
	ships := make([]*objUnit.BattleShip, 0, len(groups))
	spacing := 3
	startY := height/2 - (len(groups)-1)*spacing/2
	for i, group := range groups {
		ship := objUnit.NewShip(
			ms.Arena.ShipUidGenerators[player], group.Ship,
			objPos.New(x, startY+i*spacing), rotation, player,
		)
		if group.InitialHP > 0 {
			ship.CurHP = group.InitialHP
		}
		ms.Arena.Ships[ship.Uid] = ship
		ships = append(ships, ship)
	}
	return ships
}

func assignCarrierTargets(carriers, enemies []*objUnit.BattleShip) {
	if len(enemies) == 0 {
		return
	}
	for _, carrier := range carriers {
		if carrier.Aircraft.HasPlane && carrier.AttackTarget == "" {
			carrier.AttackTarget = enemies[0].Uid
		}
	}
}

func applyFocusDoctrine(attackers, enemies []*objUnit.BattleShip) {
	if len(attackers) == 0 || len(enemies) == 0 {
		return
	}
	best := enemies[0]
	bestCost := fleetCost([]*objUnit.BattleShip{best})
	for _, enemy := range enemies[1:] {
		if cost := fleetCost([]*objUnit.BattleShip{enemy}); cost > bestCost {
			best, bestCost = enemy, cost
		}
	}
	for _, ship := range attackers {
		ship.AttackTarget = best.Uid
	}
}

func fleetCost(ships []*objUnit.BattleShip) float64 {
	total := 0.0
	for _, ship := range ships {
		cost, _ := objUnit.GetShipHullCost(ship.Name)
		total += float64(cost)
	}
	return total
}

func costMap(groups ...[]*objUnit.BattleShip) map[string]float64 {
	out := map[string]float64{}
	for _, ships := range groups {
		for _, ship := range ships {
			cost, _ := objUnit.GetShipHullCost(ship.Name)
			out[ship.Uid] = float64(cost)
		}
	}
	return out
}

func survivalCurve(trials []trial, maxTicks int) []sim.SurvivalPoint {
	last := 0
	for _, t := range trials {
		if int(t.lastTick) > last {
			last = int(t.lastTick)
		}
	}
	if last <= 0 {
		last = min(maxTicks, 60)
	}
	step := 60
	points := make([]sim.SurvivalPoint, 0, last/step+2)
	for tick := 0; tick <= last; tick += step {
		sum := 0.0
		for _, t := range trials {
			alive := t.ownStart
			for _, sink := range t.recorder.Sinks {
				if sink.TargetPlayer == faction.HumanAlpha && int(sink.Tick) <= tick {
					alive--
				}
			}
			if t.ownStart > 0 {
				sum += float64(max(alive, 0)) / float64(t.ownStart)
			}
		}
		points = append(points, sim.SurvivalPoint{
			Minute: tick / 60, Probability: round3(sum / float64(max(len(trials), 1))),
		})
	}
	return points
}

func medianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return round3((sorted[mid-1] + sorted[mid]) / 2)
}

func ratioStatus(value, target float64) string {
	if value >= target {
		return "green"
	}
	if value > 0 {
		return "yellow"
	}
	return "red"
}

func ttkStatus(ttk, maxTicks float64) string {
	if ttk <= 0 {
		return "red"
	}
	if ttk <= maxTicks {
		return "green"
	}
	return "yellow"
}

func zeroStatus(value int) string {
	if value == 0 {
		return "green"
	}
	return "red"
}

func fleetHasAircraft(groups []sim.DuelShip) bool {
	for _, group := range groups {
		if ship, ok := objUnit.ShipMap[group.Ship]; ok && ship.Aircraft.HasPlane {
			return true
		}
	}
	return false
}

func fleetHasSubmarine(groups []sim.DuelShip) bool {
	for _, group := range groups {
		if ship, ok := objUnit.ShipMap[group.Ship]; ok && ship.Type == objUnit.ShipTypeSubmarine {
			return true
		}
	}
	return false
}

func formatRatio(v float64) string {
	if v >= 999 {
		return "inf"
	}
	return fmt.Sprintf("%.2f", v)
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}
