package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CompareMetric is one aligned metric delta between two runs.
type CompareMetric struct {
	Group    string  `json:"group"`
	Name     string  `json:"name"`
	A        float64 `json:"a"`
	B        float64 `json:"b"`
	Delta    float64 `json:"delta"`
	DeltaPct float64 `json:"deltaPct"`
	Unit     string  `json:"unit"`
}

// CompareFinding is one finding difference between two runs.
type CompareFinding struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	StatusA  string `json:"statusA,omitempty"`
	StatusB  string `json:"statusB,omitempty"`
}

// CompareReport is a run-to-run calibration diff.
type CompareReport struct {
	SchemaVersion int              `json:"schemaVersion"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	ARun          string           `json:"aRun"`
	BRun          string           `json:"bRun"`
	AMeta         string           `json:"aScenario"`
	BMeta         string           `json:"bScenario"`
	Metrics       []CompareMetric  `json:"metrics"`
	FindingsAdded []CompareFinding `json:"findingsAdded"`
	FindingsGone  []CompareFinding `json:"findingsGone"`
	ScoreChanges  []string         `json:"scoreChanges"`
}

// CompareOutput lists generated comparison files.
type CompareOutput struct {
	Markdown string
	JSON     string
}

// WriteComparison reads two runs and writes compare/<a>__<b>.{md,json}.
func WriteComparison(paths Paths, aArg, bArg string) (CompareOutput, error) {
	a, aID, err := loadComparisonRun(paths, aArg)
	if err != nil {
		return CompareOutput{}, err
	}
	b, bID, err := loadComparisonRun(paths, bArg)
	if err != nil {
		return CompareOutput{}, err
	}
	report := CompareReport{
		SchemaVersion: 1, GeneratedAt: time.Now(), ARun: aID, BRun: bID,
		AMeta: a.Scenario.Name, BMeta: b.Scenario.Name,
	}
	aMetrics := metricMap(a.Metrics)
	bMetrics := metricMap(b.Metrics)
	names := map[string]bool{}
	for name := range aMetrics {
		names[name] = true
	}
	for name := range bMetrics {
		names[name] = true
	}
	for name := range names {
		am := aMetrics[name]
		bm := bMetrics[name]
		metric := CompareMetric{
			Group: firstNonEmpty(bm.Group, am.Group), Name: name,
			A: am.Value, B: bm.Value, Delta: roundCompare(bm.Value - am.Value),
			Unit: firstNonEmpty(bm.Unit, am.Unit),
		}
		if am.Value != 0 {
			metric.DeltaPct = roundCompare((bm.Value - am.Value) / am.Value * 100)
		}
		report.Metrics = append(report.Metrics, metric)
	}
	sort.Slice(report.Metrics, func(i, j int) bool {
		if report.Metrics[i].Group != report.Metrics[j].Group {
			return report.Metrics[i].Group < report.Metrics[j].Group
		}
		return report.Metrics[i].Name < report.Metrics[j].Name
	})
	aFindings := findingMap(a.Findings)
	bFindings := findingMap(b.Findings)
	for id, f := range bFindings {
		if _, ok := aFindings[id]; !ok {
			report.FindingsAdded = append(report.FindingsAdded, findingDiff(f, "", f.Status))
		}
	}
	for id, f := range aFindings {
		if _, ok := bFindings[id]; !ok {
			report.FindingsGone = append(report.FindingsGone, findingDiff(f, f.Status, ""))
		}
	}
	sort.Slice(
		report.FindingsAdded,
		func(i, j int) bool { return report.FindingsAdded[i].ID < report.FindingsAdded[j].ID },
	)
	sort.Slice(
		report.FindingsGone,
		func(i, j int) bool { return report.FindingsGone[i].ID < report.FindingsGone[j].ID },
	)
	for _, item := range b.Scorecard {
		for _, prev := range a.Scorecard {
			if item.Dimension == prev.Dimension && item.Metric == prev.Metric && item.Status != prev.Status {
				report.ScoreChanges = append(
					report.ScoreChanges,
					fmt.Sprintf("%s/%s: %s → %s", item.Dimension, item.Metric, prev.Status, item.Status),
				)
			}
		}
	}
	sort.Strings(report.ScoreChanges)
	dir := filepath.Join(paths.Reports, "compare")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return CompareOutput{}, err
	}
	base := aID + "__" + bID
	mdPath := filepath.Join(dir, base+".md")
	if err := writeFile(mdPath, []byte(compareMarkdown(report))); err != nil {
		return CompareOutput{}, err
	}
	jsonPath := filepath.Join(dir, base+".json")
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return CompareOutput{}, err
	}
	if err := writeFile(jsonPath, append(payload, '\n')); err != nil {
		return CompareOutput{}, err
	}
	return CompareOutput{Markdown: mdPath, JSON: jsonPath}, nil
}

func loadComparisonRun(paths Paths, arg string) (*Result, string, error) {
	runDir, err := resolveRunDir(paths.Reports, arg)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(filepath.Join(runDir, "result.json"))
	if err != nil {
		return nil, "", err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, "", err
	}
	return &result, filepath.Base(runDir), nil
}

func metricMap(rows []MetricRow) map[string]MetricRow {
	out := make(map[string]MetricRow, len(rows))
	for _, row := range rows {
		out[row.Name] = row
	}
	return out
}

func findingMap(rows []Finding) map[string]Finding {
	out := make(map[string]Finding, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out
}

func findingDiff(f Finding, statusA, statusB string) CompareFinding {
	return CompareFinding{ID: f.ID, Title: f.Title, Severity: f.Severity, StatusA: statusA, StatusB: statusB}
}

func compareMarkdown(report CompareReport) string {
	var b strings.Builder
	b.WriteString("# Run comparison\n\n")
	fmt.Fprintf(&b, "- A: `%s` (%s)\n", report.ARun, report.AMeta)
	fmt.Fprintf(&b, "- B: `%s` (%s)\n", report.BRun, report.BMeta)
	fmt.Fprintf(&b, "- generated: %s\n\n", report.GeneratedAt.Format(time.RFC3339))
	b.WriteString("## Metrics\n\n| group | metric | A | B | delta | delta% |\n|---|---|---:|---:|---:|---:|\n")
	for _, m := range report.Metrics {
		fmt.Fprintf(
			&b,
			"| %s | %s | %.3f | %.3f | %+.3f | %+.1f%% |\n",
			m.Group,
			m.Name,
			m.A,
			m.B,
			m.Delta,
			m.DeltaPct,
		)
	}
	b.WriteString("\n## Scorecard changes\n\n")
	if len(report.ScoreChanges) == 0 {
		b.WriteString("无。\n")
	} else {
		for _, change := range report.ScoreChanges {
			fmt.Fprintf(&b, "- %s\n", change)
		}
	}
	b.WriteString("\n## Findings added\n\n")
	if len(report.FindingsAdded) == 0 {
		b.WriteString("无。\n")
	} else {
		for _, f := range report.FindingsAdded {
			fmt.Fprintf(&b, "- [%s/%s] `%s` %s\n", f.Severity, f.StatusB, f.ID, f.Title)
		}
	}
	b.WriteString("\n## Findings gone\n\n")
	if len(report.FindingsGone) == 0 {
		b.WriteString("无。\n")
	} else {
		for _, f := range report.FindingsGone {
			fmt.Fprintf(&b, "- [%s/%s] `%s` %s\n", f.Severity, f.StatusA, f.ID, f.Title)
		}
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func roundCompare(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

// WriteComparisonToPrevious compares a run to the most recent earlier run of
// the same scenario in history.jsonl. It returns an error when no predecessor
// exists (for example the first run of a scenario).
func WriteComparisonToPrevious(paths Paths, runArg string) (CompareOutput, error) {
	target, targetID, err := loadComparisonRun(paths, runArg)
	if err != nil {
		return CompareOutput{}, err
	}
	history, err := LoadHistory(paths.Reports)
	if err != nil {
		return CompareOutput{}, err
	}
	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]
		if entry.RunID == targetID {
			continue
		}
		prev, prevID, err := loadComparisonRun(paths, entry.RunID)
		if err != nil {
			continue
		}
		if prev.Scenario.Name != target.Scenario.Name {
			continue
		}
		if prev.StartedAt.After(target.StartedAt) {
			continue
		}
		return WriteComparison(paths, prevID, targetID)
	}
	return CompareOutput{}, fmt.Errorf("no previous run for scenario %q", target.Scenario.Name)
}
