package sim

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/narasux/jutland/pkg/mission/object/damage"
)

type shotRecord struct {
	applied   float64
	effective float64
	overkill  float64
	critical  int
	killed    bool
}

// RunShot executes the F0 pure-damage bench: no units, no hit detection, no
// resources. It applies the production ship damage formula to a scenario-defined
// target and records per-shot effective/overkill damage.
func RunShot(sc *Scenario, startedAt time.Time) (*Result, error) {
	if sc == nil || sc.Shot == nil {
		return nil, fmt.Errorf("shot scenario is required")
	}
	if sc.Mode != ModeShot {
		return nil, fmt.Errorf("scenario mode is %q, want shot", sc.Mode)
	}
	spec := sc.Shot
	rng := rand.New(rand.NewSource(sc.Seed))
	start := time.Now()

	records := make([]shotRecord, 0, min(sc.Runs*spec.Count, 100_000))
	shotsToKill := make([]float64, 0, sc.Runs)
	var totalApplied, totalEffective, totalOverkill float64
	criticals, kills, censored := 0, 0, 0
	for run := 0; run < sc.Runs; run++ {
		hp := spec.TargetHP
		killedAt := 0
		for shot := 0; shot < spec.Count; shot++ {
			real, critical := damage.ResolveShip(
				spec.BaseDamage,
				spec.HorizontalReduction, spec.VerticalReduction,
				spec.Plunge, spec.CriticalRate, rng.Float64(),
			)
			effective := math.Min(real, hp)
			overkill := real - effective
			hp = max(0, hp-real)
			records = append(records, shotRecord{
				applied: real, effective: effective, overkill: overkill,
				critical: critical, killed: hp <= 0,
			})
			totalApplied += real
			totalEffective += effective
			totalOverkill += overkill
			if critical > 1 {
				criticals++
			}
			if hp <= 0 {
				killedAt = shot + 1
				kills++
				shotsToKill = append(shotsToKill, float64(killedAt))
				break
			}
		}
		if killedAt == 0 {
			censored++
		}
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("shot produced no samples")
	}
	totalShots := float64(len(records))
	meanApplied := totalApplied / totalShots
	meanEffective := totalEffective / totalShots
	overkillRate := totalOverkill / max(totalApplied, 1e-9)
	critRate := float64(criticals) / totalShots
	killRate := float64(kills) / float64(sc.Runs)
	censoredRate := float64(censored) / float64(sc.Runs)
	medianKill := median(shotsToKill)

	scorecard := []ScorecardItem{
		formulaScorecard(spec, meanApplied),
		{
			Dimension: "样本量",
			Metric:    "shots",
			Value:     fmt.Sprintf("%.0f", totalShots),
			Target:    ">= 1000",
			Status:    statusAtLeast(totalShots, 1000),
		},
		{
			Dimension: "暴击率",
			Metric:    "observed",
			Value:     fmt.Sprintf("%.2f%%", critRate*100),
			Target:    fmt.Sprintf("%.2f%%", spec.CriticalRate*100),
			Status:    statusNear(critRate, spec.CriticalRate, 0.02),
		},
		{
			Dimension: "过杀率",
			Metric:    "overkill / applied",
			Value:     fmt.Sprintf("%.2f%%", overkillRate*100),
			Target:    "< 30%",
			Status:    statusAtMost(overkillRate, 0.30),
		},
		{
			Dimension: "删失率",
			Metric:    "not killed",
			Value:     fmt.Sprintf("%.2f%%", censoredRate*100),
			Target:    "< 30%",
			Status:    statusAtMost(censoredRate, 0.30),
		},
	}
	metrics := []MetricRow{
		{Group: "命中", Name: "mean_applied_damage", Value: round3(meanApplied), Unit: "HP"},
		{Group: "命中", Name: "mean_effective_damage", Value: round3(meanEffective), Unit: "HP"},
		{Group: "伤害", Name: "overkill_rate", Value: round3(overkillRate * 100), Unit: "%"},
		{Group: "伤害", Name: "crit_rate", Value: round3(critRate * 100), Unit: "%"},
		{Group: "节奏", Name: "kill_rate", Value: round3(killRate * 100), Unit: "%"},
		{Group: "节奏", Name: "ttk_median_shots", Value: medianKill, Unit: "shots"},
		{Group: "样本", Name: "total_shots", Value: totalShots, Unit: "shots"},
	}

	findings := make([]Finding, 0)
	if spec.ExpectedDamage != nil {
		delta := meanApplied - *spec.ExpectedDamage
		if math.Abs(delta) > spec.Tolerance {
			findings = append(findings, Finding{
				ID: "F0-DAMAGE-MISMATCH", Kind: "bug", Severity: "high", Status: "new",
				Title:      "伤害台结果与已知答案不一致",
				Summary:    fmt.Sprintf("平均实际伤害 %.3f，期望 %.3f，差 %.3f。", meanApplied, *spec.ExpectedDamage, delta),
				Confidence: 0.98,
				Evidence: []Evidence{{
					Metric: "mean_applied_damage", Value: round3(meanApplied),
					Baseline: *spec.ExpectedDamage, Delta: round3(delta),
					File: "ai/metrics.json#mean_applied_damage",
				}},
				SuggestedExperiment: "检查装甲/落角/暴击公式与弹药字段，确认是否配置或实现回归。",
				Repro:               "jutland-sim shot $JUTLAND_SIM_HOME/scenarios/<file> --seed <seed>",
			})
		}
	}
	if censoredRate > 0.30 {
		findings = append(findings, Finding{
			ID: "F0-CENSORED", Kind: "data", Severity: "medium", Status: "new",
			Title: "删失率偏高", Summary: fmt.Sprintf("%.1f%% 的 run 在 shot.count 内未击沉目标。", censoredRate*100),
			Confidence: 0.95,
		})
	}

	return &Result{
		SchemaVersion: 1,
		Mock:          false,
		Fidelity:      "F0",
		Scenario:      *sc,
		Seed:          sc.Seed,
		Runs:          sc.Runs,
		StartedAt:     startedAt,
		DurationMS:    time.Since(start).Milliseconds(),
		Scorecard:     scorecard,
		Metrics:       metrics,
		Findings:      findings,
		Diagnostics:   nil,
		Charts:        shotCharts(spec, sc.Runs, records, shotsToKill),
		DataQuality: []string{
			"F0 纯伤害台：目标 HP/装甲/弹药来自场景，不包含命中、射击、舰船配置或地图。",
			"结论只用于伤害公式、弹药/装甲回填与已知答案校验，不能直接当实战平衡。",
		},
	}, nil
}

func formulaScorecard(spec *ShotSpec, mean float64) ScorecardItem {
	item := ScorecardItem{Dimension: "伤害公式", Metric: "mean applied", Value: fmt.Sprintf("%.3f", mean)}
	if spec.ExpectedDamage == nil {
		item.Target = "未设预期"
		item.Status = "green"
		item.Note = "场景未提供 expectedDamage；只做描述性统计。"
		return item
	}
	item.Target = fmt.Sprintf("%.3f ± %.3f", *spec.ExpectedDamage, spec.Tolerance)
	delta := math.Abs(mean - *spec.ExpectedDamage)
	switch {
	case delta <= spec.Tolerance:
		item.Status = "green"
	case delta <= spec.Tolerance+0.05*math.Abs(*spec.ExpectedDamage):
		item.Status = "yellow"
	default:
		item.Status = "red"
	}
	return item
}

func shotCharts(spec *ShotSpec, runs int, records []shotRecord, shotsToKill []float64) Charts {
	labels := make([]string, 0, 11)
	hit := make([]HitRatePoint, 0, 11)
	for i := 0; i <= 10; i++ {
		ratio := float64(i) / 10
		labels = append(labels, fmt.Sprintf("%.1f", ratio))
		hit = append(hit, HitRatePoint{
			Range: labels[i],
			AP:    round2(spec.BaseDamage * (1 - spec.HorizontalReduction*ratio)),
			HE:    round2(spec.BaseDamage * (1 - spec.VerticalReduction*ratio)),
		})
	}

	maxShots := min(spec.Count, 30)
	survival := make([]SurvivalPoint, 0, maxShots)
	for n := 1; n <= maxShots; n++ {
		killed := 0
		for _, at := range shotsToKill {
			if int(at) <= n {
				killed++
			}
		}
		survival = append(
			survival,
			SurvivalPoint{Minute: n, Probability: round3(float64(killed) / float64(max(runs, 1)))},
		)
	}
	return Charts{
		HitRate:       hit,
		HitRateLabels: labels,
		Survival:      survival,
	}
}

func median(values []float64) float64 {
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

func statusAtLeast(value, target float64) string {
	if value >= target {
		return "green"
	}
	if value >= target*0.8 {
		return "yellow"
	}
	return "red"
}

func statusAtMost(value, target float64) string {
	if value <= target {
		return "green"
	}
	if value <= target*1.5 {
		return "yellow"
	}
	return "red"
}

func statusNear(value, target, tolerance float64) string {
	switch delta := math.Abs(value - target); {
	case delta <= tolerance:
		return "green"
	case delta <= tolerance*2:
		return "yellow"
	default:
		return "red"
	}
}
