package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PerfReport is the p99 gate result for one run.
type PerfReport struct {
	SchemaVersion int     `json:"schemaVersion"`
	RunID         string  `json:"runId"`
	Fidelity      string  `json:"fidelity"`
	BudgetMS      float64 `json:"budgetMs"`
	P99MS         float64 `json:"p99Ms"`
	Pass          bool    `json:"pass"`
	Note          string  `json:"note,omitempty"`
}

// CheckPerf reads a run's result.json and compares tick_p99_ms with budget.
// An empty runArg selects the newest run that has tick_p99_ms.
func CheckPerf(paths Paths, runArg string, budgetMS float64) (PerfReport, error) {
	if budgetMS <= 0 {
		budgetMS = 16.7
	}
	runDir, err := resolvePerfRun(paths.Reports, runArg)
	if err != nil {
		return PerfReport{}, err
	}
	data, err := os.ReadFile(filepath.Join(runDir, "result.json"))
	if err != nil {
		return PerfReport{}, err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return PerfReport{}, fmt.Errorf("parse result.json: %w", err)
	}
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	p99, ok := metrics["tick_p99_ms"]
	if !ok {
		return PerfReport{}, fmt.Errorf("run %s has no tick_p99_ms (F3 required)", filepath.Base(runDir))
	}
	report := PerfReport{
		SchemaVersion: 1, RunID: filepath.Base(runDir), Fidelity: result.Fidelity,
		BudgetMS: budgetMS, P99MS: p99, Pass: p99 <= budgetMS,
		Note: "headless simulation tick p99; excludes draw/audio",
	}
	outDir := filepath.Join(paths.Reports, "perf")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return PerfReport{}, err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return PerfReport{}, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "latest.json"), append(payload, '\n'), 0o644); err != nil {
		return PerfReport{}, err
	}
	return report, nil
}

func resolvePerfRun(reportDir, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg != "" {
		return resolveRunDir(reportDir, arg)
	}
	entries, err := os.ReadDir(filepath.Join(reportDir, "runs"))
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(reportDir, "runs", name, "result.json"))
		if err != nil {
			continue
		}
		var result Result
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		for _, row := range result.Metrics {
			if row.Name == "tick_p99_ms" {
				return filepath.Join(reportDir, "runs", name), nil
			}
		}
	}
	return "", fmt.Errorf("no F3 run with tick_p99_ms found")
}
