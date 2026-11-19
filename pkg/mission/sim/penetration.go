package sim

import (
	"fmt"
	"math"
	"time"

	"github.com/narasux/jutland/pkg/mission/object/damage"
)

// RunPenetration builds a pure geometric AP/HE penetration matrix. It does not
// change the production HurtBy path; the outcome is an analysis contract for a
// future gameplay mechanic.
func RunPenetration(sc *Scenario, startedAt time.Time) (*Result, error) {
	if sc == nil || sc.Penetration == nil {
		return nil, fmt.Errorf("penetration scenario is required")
	}
	spec := sc.Penetration
	start := time.Now()
	steps := max(spec.Steps, 2)
	labels := make([]string, 0, steps)
	points := make([]PenetrationPoint, 0, steps)
	apPenetrated, hePenetrated := 0, 0
	penetrated := 0
	maxArmor := 0.0
	for i := 0; i < steps; i++ {
		armor := spec.ArmorMin + (spec.ArmorMax-spec.ArmorMin)*float64(i)/float64(steps-1)
		ap := damage.ResolvePenetration(spec.PenetrationMM, armor, spec.AngleMin)
		he := damage.ResolvePenetration(spec.PenetrationMM, armor, spec.AngleMax)
		if ap.Penetrated {
			apPenetrated++
			maxArmor = armor
		}
		if he.Penetrated {
			hePenetrated++
		}
		if ap.Penetrated && he.Penetrated {
			penetrated++
		}
		label := fmt.Sprintf("%.0fmm", armor)
		labels = append(labels, label)
		points = append(points, PenetrationPoint{
			Range: label, AP: roundPen(spec.PenetrationMM), HE: roundPen(spec.PenetrationMM / 2), Armor: armor,
		})
	}
	reference := damage.ResolvePenetration(spec.PenetrationMM, spec.ReferenceArmor, spec.ReferenceAngle)
	yMax := math.Max(spec.PenetrationMM, spec.ArmorMax) * 1.1
	score := "green"
	if !reference.Penetrated {
		score = "red"
	}
	scorecard := []ScorecardItem{
		{
			Dimension: "穿甲",
			Metric:    "reference penetrated",
			Value:     fmt.Sprintf("%t", reference.Penetrated),
			Target:    "true",
			Status:    score,
		},
		{
			Dimension: "穿甲",
			Metric:    "reference ratio",
			Value:     fmt.Sprintf("%.2f", reference.Ratio),
			Target:    ">= 1.0",
			Status:    ratioPenStatus(reference.Ratio),
		},
		{
			Dimension: "矩阵",
			Metric:    "AP penetrated armors",
			Value:     fmt.Sprintf("%d/%d", apPenetrated, steps),
			Target:    "> 0",
			Status:    countPenStatus(apPenetrated),
		},
		{
			Dimension: "矩阵",
			Metric:    "HE penetrated armors",
			Value:     fmt.Sprintf("%d/%d", hePenetrated, steps),
			Target:    "> 0",
			Status:    countPenStatus(hePenetrated),
		},
	}
	metrics := []MetricRow{
		{Group: "穿甲", Name: "penetration_mm", Value: spec.PenetrationMM, Unit: "mm"},
		{Group: "穿甲", Name: "reference_armor", Value: spec.ReferenceArmor, Unit: "mm"},
		{Group: "穿甲", Name: "reference_angle", Value: spec.ReferenceAngle, Unit: "deg"},
		{Group: "穿甲", Name: "reference_ratio", Value: roundPen(reference.Ratio), Unit: "x"},
		{Group: "穿甲", Name: "reference_effective_armor", Value: roundPen(reference.EffectiveArmorMM), Unit: "mm"},
		{Group: "矩阵", Name: "ap_penetrated", Value: float64(apPenetrated), Unit: "points"},
		{Group: "矩阵", Name: "he_penetrated", Value: float64(hePenetrated), Unit: "points"},
		{Group: "矩阵", Name: "max_penetrated_armor", Value: roundPen(maxArmor), Unit: "mm"},
	}
	findings := make([]Finding, 0)
	if apPenetrated == 0 {
		findings = append(findings, Finding{
			ID: "PEN-AP-NO-PENETRATION", Kind: "balance", Severity: "high", Status: "new",
			Title: "AP 在整个装甲区间都无法穿透",
			Summary: fmt.Sprintf(
				"penetration=%.0fmm armor=[%.0f,%.0f] angle=%.0f°",
				spec.PenetrationMM,
				spec.ArmorMin,
				spec.ArmorMax,
				spec.AngleMin,
			),
			Confidence: 0.95,
		})
	}
	if !reference.Penetrated {
		findings = append(findings, Finding{
			ID: "PEN-REFERENCE-NO-PEN", Kind: "balance", Severity: "medium", Status: "new",
			Title: "参考点未穿透",
			Summary: fmt.Sprintf(
				"armor=%.0f angle=%.0f° ratio=%.2f",
				spec.ReferenceArmor,
				spec.ReferenceAngle,
				reference.Ratio,
			),
			Confidence: 0.9,
		})
	}
	return &Result{
		SchemaVersion: 1,
		Mock:          false,
		Fidelity:      "F0",
		Scenario:      *sc,
		Seed:          sc.Seed,
		Runs:          1,
		StartedAt:     startedAt,
		DurationMS:    time.Since(start).Milliseconds(),
		Scorecard:     scorecard,
		Metrics:       metrics,
		Findings:      findings,
		Charts: Charts{
			Penetration:    points,
			HitRateLabels:  labels,
			PenetrationMax: roundPen(yMax),
		},
		DataQuality: []string{
			"F0 纯几何穿深矩阵：penetration vs angle-adjusted armor。",
			"该模型不在生产 HurtBy 路径中；用于设计评审/契约，不是当前游戏平衡结论。",
			fmt.Sprintf(
				"AP angle=%.0f°，HE angle=%.0f°，armor=%.0f..%.0fmm。",
				spec.AngleMin,
				spec.AngleMax,
				spec.ArmorMin,
				spec.ArmorMax,
			),
		},
	}, nil
}

func ratioPenStatus(ratio float64) string {
	if ratio >= 1 {
		return "green"
	}
	if ratio >= 0.75 {
		return "yellow"
	}
	return "red"
}

func countPenStatus(count int) string {
	if count > 0 {
		return "green"
	}
	return "red"
}

func roundPen(v float64) float64 {
	if math.IsInf(v, 1) {
		return v
	}
	return math.Round(v*1000) / 1000
}
