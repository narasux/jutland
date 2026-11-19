package sim

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Baseline is a lab-owned reference run used to detect metric drift.
type Baseline struct {
	SchemaVersion int                `json:"schemaVersion"`
	Name          string             `json:"name"`
	Scenario      string             `json:"scenario,omitempty"`
	Fidelity      string             `json:"fidelity,omitempty"`
	Commit        string             `json:"commit,omitempty"`
	CreatedAt     time.Time          `json:"createdAt"`
	Metrics       map[string]float64 `json:"metrics"`
	Tolerance     map[string]float64 `json:"tolerance,omitempty"`
}

// ResolveBaselinePath resolves an absolute/relative baseline file or a name
// under the lab's baselines directory.
func ResolveBaselinePath(paths Paths, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("baseline path is empty")
	}
	candidates := []string{arg}
	if filepath.Ext(arg) == "" {
		candidates = append(candidates, arg+".json")
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				return "", err
			}
			return abs, nil
		}
	}
	for _, candidate := range candidates {
		full := filepath.Join(paths.Baselines, candidate)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			return full, nil
		}
	}
	return "", fmt.Errorf("baseline %q not found (tried cwd and %s)", arg, paths.Baselines)
}

// LoadBaseline reads and validates a baseline JSON file.
func LoadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("parse baseline %s: %w", path, err)
	}
	if len(b.Metrics) == 0 {
		return nil, fmt.Errorf("baseline %s has no metrics", path)
	}
	if b.SchemaVersion == 0 {
		b.SchemaVersion = 1
	}
	return &b, nil
}

// ApplyBaseline fills MetricRow.Baseline/Delta and adds a finding for each
// metric outside its tolerance. It returns the number of matched metrics.
func ApplyBaseline(result *Result, b *Baseline) (int, error) {
	if result == nil || b == nil {
		return 0, fmt.Errorf("result and baseline are required")
	}
	if b.Scenario != "" && b.Scenario != result.Scenario.Name {
		return 0, fmt.Errorf("baseline scenario %q does not match %q", b.Scenario, result.Scenario.Name)
	}
	if b.Fidelity != "" && b.Fidelity != result.Fidelity {
		return 0, fmt.Errorf("baseline fidelity %q does not match %q", b.Fidelity, result.Fidelity)
	}
	matched := 0
	for i := range result.Metrics {
		row := &result.Metrics[i]
		base, ok := b.Metrics[row.Name]
		if !ok {
			continue
		}
		matched++
		row.Baseline = base
		row.Delta = roundBaseline3(row.Value - base)
		tolerance := b.Tolerance[row.Name]
		if tolerance > 0 && math.Abs(row.Delta) > tolerance {
			result.Findings = append(result.Findings, Finding{
				ID:         "BASELINE-DRIFT-" + row.Name,
				Kind:       "balance",
				Severity:   driftSeverity(row.Delta, tolerance),
				Status:     "new",
				Title:      fmt.Sprintf("基线漂移：%s", row.Name),
				Summary:    fmt.Sprintf("当前 %.3f，基线 %.3f，delta %.3f，容差 %.3f。", row.Value, base, row.Delta, tolerance),
				Confidence: 0.9,
				Evidence: []Evidence{{
					Metric: row.Name, Value: row.Value, Baseline: base, Delta: row.Delta,
					File: "baselines/" + b.Name + ".json",
				}},
			})
		}
	}
	if matched == 0 {
		return 0, fmt.Errorf("baseline %q has no metric names shared with result", b.Name)
	}
	result.DataQuality = append(result.DataQuality,
		fmt.Sprintf("baseline: %s，matched=%d，commit=%s。", b.Name, matched, b.Commit))
	return matched, nil
}

// WriteBaselineFromRun snapshots result.json into the lab baselines directory.
// runArg is a run ID under reports/runs or a path to a run dir/result.json.
func WriteBaselineFromRun(paths Paths, runArg, name string, tolerance map[string]float64) (string, error) {
	runDir, err := resolveRunDir(paths.Reports, runArg)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(runDir, "result.json"))
	if err != nil {
		return "", err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse result.json: %w", err)
	}
	if strings.TrimSpace(name) == "" {
		name = result.Scenario.Name
	}
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("baseline name is required")
	}
	metrics := make(map[string]float64, len(result.Metrics))
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	if len(metrics) == 0 {
		return "", fmt.Errorf("run has no numeric metrics")
	}
	git := CollectGitInfo("")
	baseline := Baseline{
		SchemaVersion: 1, Name: name, Scenario: result.Scenario.Name,
		Fidelity: result.Fidelity, Commit: git.CommitShort,
		CreatedAt: time.Now(), Metrics: metrics, Tolerance: tolerance,
	}
	if err := os.MkdirAll(paths.Baselines, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(paths.Baselines, name+".json")
	payload, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(out, append(payload, '\n'), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func resolveRunDir(reportDir, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("run id or path is required")
	}
	candidates := []string{arg, filepath.Join(reportDir, "runs", arg)}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			return candidate, nil
		}
		if filepath.Base(candidate) == "result.json" {
			return filepath.Dir(candidate), nil
		}
	}
	return "", fmt.Errorf("run %q not found", arg)
}

// ParseTolerances parses "metric=value,metric2=value2".
func ParseTolerances(raw string) (map[string]float64, error) {
	out := map[string]float64{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("tolerance %q must be metric=value", part)
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return nil, fmt.Errorf("tolerance %q: %w", part, err)
		}
		out[strings.TrimSpace(key)] = parsed
	}
	return out, nil
}

func driftSeverity(delta, tolerance float64) string {
	if math.Abs(delta) > tolerance*2 {
		return "high"
	}
	return "medium"
}

func roundBaseline3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
