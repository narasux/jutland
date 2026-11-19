package sim

import (
	"math"
	"math/rand"
	"time"
)

// DefaultDemoScenario returns the built-in P1 report-mock scenario.
func DefaultDemoScenario() Scenario {
	return Scenario{
		SchemaVersion: 1,
		Name:          "demo-report",
		Description:   "P1 报告版式 mock：合成数据，不代表真实模拟结果。",
		Mode:          ModeDemo,
		Seed:          42,
		Runs:          100,
		MaxTicks:      3600,
		Environment: Environment{
			Map:             "open_sea",
			SpeedMultiplier: 1.0,
		},
		Notes: []string{"合成数据，仅用于确认报告版式、字段和阅读动线。"},
	}
}

// GenerateDemoResult builds deterministic synthetic metrics for the P1 report mock.
func GenerateDemoResult(sc *Scenario, startedAt time.Time) *Result {
	rng := rand.New(rand.NewSource(sc.Seed))
	ranges := []string{"0.1R", "0.2R", "0.3R", "0.4R", "0.5R", "0.6R", "0.7R", "0.8R", "0.9R", "1.0R"}
	ap := []float64{0.86, 0.80, 0.73, 0.66, 0.58, 0.49, 0.39, 0.29, 0.20, 0.12}
	he := []float64{0.88, 0.83, 0.77, 0.70, 0.62, 0.53, 0.44, 0.35, 0.26, 0.18}
	hitRate := make([]HitRatePoint, 0, len(ranges))
	for i, label := range ranges {
		hitRate = append(hitRate, HitRatePoint{
			Range: label,
			AP:    round3(ap[i] + (rng.Float64()-0.5)*0.015),
			HE:    round3(he[i] + (rng.Float64()-0.5)*0.015),
		})
	}

	survivalProb := []float64{1.0, 0.94, 0.84, 0.72, 0.61, 0.52, 0.44, 0.37, 0.31, 0.26, 0.22}
	survival := make([]SurvivalPoint, 0, len(survivalProb))
	for i, p := range survivalProb {
		survival = append(survival, SurvivalPoint{Minute: i, Probability: round3(p)})
	}

	classes := []string{"BB", "CA", "DD", "SS", "CV"}
	values := [][]float64{
		{1.00, 1.12, 1.05, 0.78, 1.08},
		{0.89, 1.00, 1.06, 0.74, 0.95},
		{0.95, 0.94, 1.00, 0.82, 0.90},
		{1.28, 1.35, 1.22, 1.00, 1.18},
		{0.93, 1.05, 1.11, 0.85, 1.00},
	}
	exchange := make([]ExchangeCell, 0, len(classes)*len(classes))
	for y, attacker := range classes {
		for x, defender := range classes {
			exchange = append(exchange, ExchangeCell{
				Attacker: attacker,
				Defender: defender,
				Ratio:    round2(values[y][x] + (rng.Float64()-0.5)*0.03),
			})
		}
	}

	penAP := []float64{690, 655, 620, 585, 550, 515, 480, 445, 410, 375}
	penHE := []float64{88, 86, 84, 82, 80, 78, 76, 74, 72, 70}
	penetration := make([]PenetrationPoint, 0, len(ranges))
	for i, label := range ranges {
		penetration = append(penetration, PenetrationPoint{
			Range: label,
			AP:    penAP[i],
			HE:    penHE[i],
			Armor: 410,
		})
	}

	scorecard := []ScorecardItem{
		{Dimension: "对称胜率", Metric: "5 DD vs 1 BB", Value: "52.0% (49.1~54.9)", Target: "45~55%", Status: "green"},
		{Dimension: "成本交换", Metric: "exchange ratio", Value: "1.07 (0.95~1.19)", Target: "0.8~1.2", Status: "green"},
		{Dimension: "TTK", Metric: "median TTK", Value: "214s (p90 402s)", Target: "120~480s", Status: "green"},
		{
			Dimension: "成本效率",
			Metric:    "damage / 100 funds",
			Value:     "8,420",
			Target:    "median ±15%",
			Status:    "yellow",
			Note:      "略高于巡洋舰中位数，需结合生存率看。",
		},
		{Dimension: "数据质量", Metric: "censored rate", Value: "18%", Target: "<30%", Status: "green"},
	}

	metrics := []MetricRow{
		{Group: "命中", Name: "hit_rate", Value: 52.4, Unit: "%", CI: "49.1~55.7"},
		{Group: "伤害", Name: "damage_per_salvo", Value: 1820, Unit: "HP", CI: "1710~1930"},
		{Group: "伤害", Name: "overkill_rate", Value: 11.0, Unit: "%"},
		{Group: "节奏", Name: "ttk_median", Value: 214, Unit: "s", CI: "198~236"},
		{Group: "成本", Name: "damage_per_100_funds", Value: 8420, Unit: "HP"},
		{Group: "防空", Name: "aircraft_loss_rate", Value: 27.0, Unit: "%"},
		{Group: "鱼雷", Name: "torpedo_hit_rate", Value: 18.0, Unit: "%", CI: "15.2~20.9"},
	}

	findings := []Finding{
		{
			ID: "F-20261119-001", Kind: "bug", Severity: "high", Title: "示例：目标在射程内但 0 射击",
			Summary:    "合成数据示例：目标全程位于主炮射程内，武器已装填，但 shots=0。",
			Confidence: 0.86,
			Evidence: []Evidence{
				{Metric: "shots", Value: 0, Baseline: 420, Delta: -420, File: "metrics.json#shots"},
				{Metric: "target_in_range_ratio", Value: 1.0, File: "metrics.json#target_in_range_ratio"},
			},
			Affected:            []string{"test-gun", "test-target"},
			SuggestedExperiment: "复跑同场景并打开 observer，确认是射界、目标选择还是装填状态问题。",
			Repro:               "jutland-sim run $JUTLAND_SIM_HOME/scenarios/demo-report.json5 --seed 42",
			FirstSeen:           "2026-11-19", LastSeen: "2026-11-19", Status: "new",
		},
		{
			ID: "F-20261119-002", Kind: "balance", Severity: "medium", Title: "示例：5 DD vs 1 BB 交换比接近上限",
			Summary:    "合成数据示例：交换比 1.07，仍在 0.8~1.2 目标带内，但 10km 场景胜率 62%，接近黄色上限。",
			Confidence: 0.72,
			Evidence: []Evidence{
				{
					Metric:   "exchange_ratio",
					Value:    1.07,
					Baseline: 1.02,
					Delta:    0.05,
					CI:       "0.95~1.19",
					File:     "metrics.json#exchange_ratio",
				},
				{
					Metric:   "win_rate_10km",
					Value:    0.62,
					Baseline: 0.55,
					Delta:    0.07,
					File:     "metrics.json#win_rate_10km",
				},
			},
			Affected:            []string{"fletcher", "bismarck", "torpedo"},
			SuggestedExperiment: "同场景把 DD 数量从 5 降到 4，确认是否是数量效应。",
			Repro:               "jutland-sim run $JUTLAND_SIM_HOME/scenarios/demo-report.json5 --seed 42 --runs 500",
			FirstSeen:           "2026-11-19", LastSeen: "2026-11-19", Status: "new",
		},
	}

	diagnostics := []Diagnostic{
		{ID: "MOCK-BUG-01", Severity: "high", Message: "示例诊断：目标在射程内但 0 射击。", Repro: "见 findings.json F-20261119-001"},
		{ID: "MOCK-PERF-01", Severity: "medium", Message: "示例诊断：p99 帧时间回归 0.8%，在门禁内但接近阈值。"},
	}

	dataQuality := []string{
		"这是 P1 合成数据，不是真实战斗结果。",
		"示例删失率 18%；真实报告会按实际 run 统计。",
		"示例 CI 为占位值；真实报告由样本量计算。",
	}

	return &Result{
		SchemaVersion: 1,
		Mock:          true,
		Fidelity:      "mock",
		Scenario:      *sc,
		Seed:          sc.Seed,
		Runs:          sc.Runs,
		StartedAt:     startedAt,
		DurationMS:    12400,
		Scorecard:     scorecard,
		Metrics:       metrics,
		Findings:      findings,
		Diagnostics:   diagnostics,
		Charts: Charts{
			HitRate:        hitRate,
			Survival:       survival,
			Exchange:       exchange,
			Penetration:    penetration,
			HitRateLabels:  ranges,
			ExchangeRows:   classes,
			ExchangeCols:   classes,
			PenetrationMax: 700,
		},
		DataQuality: dataQuality,
	}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round3(v float64) float64 { return math.Round(v*1000) / 1000 }
