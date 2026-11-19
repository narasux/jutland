package sim

import (
	"fmt"
	"math"
	"strings"
)

// RunDiagnostics applies cheap invariant checks to a result. Diagnostics are
// candidates for findings, not automatic truth; they never mutate metrics.
func RunDiagnostics(result *Result) []Diagnostic {
	if result == nil {
		return nil
	}
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	out := make([]Diagnostic, 0)
	for _, row := range result.Metrics {
		if math.IsNaN(row.Value) || math.IsInf(row.Value, 0) {
			out = append(out, Diagnostic{
				ID: "DIAG-NONFINITE-" + row.Name, Severity: "high",
				Message: fmt.Sprintf("%s 不是有限数：%v", row.Name, row.Value),
			})
		}
		if row.Value < 0 {
			out = append(out, Diagnostic{
				ID: "DIAG-NEGATIVE-" + row.Name, Severity: "high",
				Message: fmt.Sprintf("%s 为负数：%v", row.Name, row.Value),
			})
		}
	}
	shots, hasShots := metrics["shots"]
	hits, hasHits := metrics["hits"]
	if hasShots && hasHits && hits > shots {
		out = append(out, Diagnostic{
			ID: "DIAG-HITS-GT-SHOTS", Severity: "high",
			Message: fmt.Sprintf("hits %.0f > shots %.0f", hits, shots),
		})
	}
	if hasShots && hasHits && shots > 0 {
		want := hits / shots * 100
		if got, ok := metrics["hit_rate"]; ok && math.Abs(got-want) > 0.2 {
			out = append(out, Diagnostic{
				ID: "DIAG-HIT-RATE-MISMATCH", Severity: "medium",
				Message: fmt.Sprintf("hit_rate %.2f != hits/shots %.2f", got, want),
			})
		}
	}
	if hasShots && shots == 0 {
		switch result.Scenario.Mode {
		case ModeDuel, ModeMicro, ModeMacro, ModeStrike:
			out = append(out, Diagnostic{
				ID: "DIAG-ZERO-SHOTS", Severity: "high",
				Message: "战斗模式没有任何 Shot 事件",
			})
		}
	}
	if friendly, ok := metrics["friendly_hits"]; ok && friendly > 0 {
		out = append(out, Diagnostic{
			ID: "DIAG-FRIENDLY-FIRE", Severity: "high",
			Message: fmt.Sprintf("友伤命中 %.0f 次", friendly),
		})
	}
	if p99, ok := metrics["tick_p99_ms"]; ok && p99 > 16.7 {
		out = append(out, Diagnostic{
			ID: "DIAG-P99-OVER-BUDGET", Severity: "medium",
			Message: fmt.Sprintf("tick_p99_ms %.3f > 16.7", p99),
		})
	}
	if killed, ok := metrics["kill_rate"]; ok && killed == 0 {
		out = append(out, Diagnostic{
			ID: "DIAG-ZERO-KILL-RATE", Severity: "medium",
			Message: "kill_rate=0，目标在样本内未被击沉",
		})
	}
	return out
}

func mergeDiagnostics(existing, computed []Diagnostic) []Diagnostic {
	if len(computed) == 0 {
		return existing
	}
	seen := map[string]bool{}
	for _, diag := range existing {
		seen[diag.ID] = true
	}
	for _, diag := range computed {
		if seen[diag.ID] {
			continue
		}
		existing = append(existing, diag)
	}
	return existing
}

func diagnosticsMarkdown(diags []Diagnostic) string {
	if len(diags) == 0 {
		return "无。\n"
	}
	var b strings.Builder
	for _, diag := range diags {
		fmt.Fprintf(&b, "- [%s] `%s` %s\n", diag.Severity, diag.ID, diag.Message)
	}
	return b.String()
}
