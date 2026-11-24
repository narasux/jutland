package macro

import (
	"fmt"
	"sort"
	"time"

	"github.com/narasux/jutland/pkg/mission/sim"
)

// RunMacroSeeds runs the same mission for seed..seed+runs-1 sequentially and
// aggregates numeric metrics. Each trial is independently deterministic.
func RunMacroSeeds(sc *sim.Scenario, runs int, startedAt time.Time) (*sim.Result, error) {
	if runs <= 1 {
		return RunMacro(sc, startedAt)
	}
	results := make([]*sim.Result, 0, runs)
	for i := 0; i < runs; i++ {
		trial := *sc
		trial.Seed = sc.Seed + int64(i)
		result, err := RunMacro(&trial, startedAt)
		if err != nil {
			return nil, fmt.Errorf("seed %d: %w", trial.Seed, err)
		}
		results = append(results, result)
	}
	return aggregateResults(sc, results), nil
}

func aggregateResults(sc *sim.Scenario, results []*sim.Result) *sim.Result {
	if len(results) == 0 {
		return nil
	}
	base := results[len(results)-1]
	totalDuration := int64(0)
	metrics := map[string]sim.MetricRow{}
	metricOrder := make([]string, 0)
	scorecard := map[string]sim.ScorecardItem{}
	scoreOrder := make([]string, 0)
	findings := map[string]sim.Finding{}
	findingOrder := make([]string, 0)
	for _, result := range results {
		totalDuration += result.DurationMS
		for _, row := range result.Metrics {
			key := row.Name
			if _, ok := metrics[key]; !ok {
				metricOrder = append(metricOrder, key)
				metrics[key] = row
				continue
			}
			existing := metrics[key]
			existing.Value += row.Value
			metrics[key] = existing
		}
		for _, row := range result.Scorecard {
			key := row.Dimension + "|" + row.Metric
			if _, ok := scorecard[key]; !ok {
				scoreOrder = append(scoreOrder, key)
				scorecard[key] = row
				continue
			}
			existing := scorecard[key]
			existing.Value = row.Value
			if statusRank(row.Status) > statusRank(existing.Status) {
				existing.Status = row.Status
			}
			scorecard[key] = existing
		}
		for _, finding := range result.Findings {
			if _, ok := findings[finding.ID]; !ok {
				findingOrder = append(findingOrder, finding.ID)
				findings[finding.ID] = finding
			}
		}
	}
	for _, key := range metricOrder {
		row := metrics[key]
		row.Value = roundSuite3(row.Value / float64(len(results)))
		metrics[key] = row
	}
	outMetrics := make([]sim.MetricRow, 0, len(metricOrder)+1)
	for _, key := range metricOrder {
		outMetrics = append(outMetrics, metrics[key])
	}
	outMetrics = append(
		outMetrics,
		sim.MetricRow{Group: "样本", Name: "runs", Value: float64(len(results)), Unit: "runs"},
	)
	if _, ok := metrics["mission_success"]; ok {
		successRate := metrics["mission_success"].Value * 100
		outScorecardForSuite := sim.ScorecardItem{
			Dimension: "胜率", Metric: "success", Value: fmt.Sprintf("%.1f%%", successRate),
			Target: ">= 50%", Status: suiteWinStatus(successRate),
		}
		scorecard["胜率|success"] = outScorecardForSuite
		scoreOrder = append(scoreOrder, "胜率|success")
	}
	outScorecard := make([]sim.ScorecardItem, 0, len(scoreOrder))
	for _, key := range scoreOrder {
		outScorecard = append(outScorecard, scorecard[key])
	}
	outFindings := make([]sim.Finding, 0, len(findingOrder))
	for _, key := range findingOrder {
		outFindings = append(outFindings, findings[key])
	}
	sort.Slice(outFindings, func(i, j int) bool { return outFindings[i].ID < outFindings[j].ID })
	battleReport := base.BattleReport
	if battleReport != nil {
		copied := *battleReport
		copied.Note = fmt.Sprintf(
			"multi-seed 聚合：战报明细来自最后一局 seed %d。", base.Seed,
		)
		battleReport = &copied
	}
	return &sim.Result{
		SchemaVersion: base.SchemaVersion,
		Mock:          base.Mock,
		Fidelity:      base.Fidelity,
		Scenario:      *sc,
		Seed:          sc.Seed,
		Runs:          len(results),
		StartedAt:     base.StartedAt,
		DurationMS:    totalDuration,
		Scorecard:     outScorecard,
		Metrics:       outMetrics,
		Findings:      outFindings,
		Charts:        aggregateCharts(results),
		DataQuality: append(append([]string{}, base.DataQuality...),
			fmt.Sprintf("multi-seed: %d sequential seeds from %d；数值与生存率曲线为均值。", len(results), sc.Seed)),
		BattleReport: battleReport,
		BattleLog:    base.BattleLog,
	}
}

func aggregateCharts(results []*sim.Result) sim.Charts {
	if len(results) == 0 {
		return sim.Charts{}
	}
	maxLen := 0
	for _, result := range results {
		if len(result.Charts.Survival) > maxLen {
			maxLen = len(result.Charts.Survival)
		}
	}
	if maxLen == 0 {
		return sim.Charts{}
	}
	points := make([]sim.SurvivalPoint, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		sum, count := 0.0, 0
		minute := 0
		for _, result := range results {
			if i >= len(result.Charts.Survival) {
				continue
			}
			point := result.Charts.Survival[i]
			sum += point.Probability
			minute = point.Minute
			count++
		}
		if count > 0 {
			points = append(points, sim.SurvivalPoint{Minute: minute, Probability: roundSuite3(sum / float64(count))})
		}
	}
	return sim.Charts{Survival: points}
}

func suiteWinStatus(rate float64) string {
	if rate >= 50 {
		return "green"
	}
	if rate > 0 {
		return "yellow"
	}
	return "red"
}

func statusRank(status string) int {
	switch status {
	case "red":
		return 3
	case "yellow":
		return 2
	case "green":
		return 1
	default:
		return 0
	}
}

func roundSuite3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}
