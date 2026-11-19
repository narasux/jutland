package sim

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// SuiteStep is one scenario execution inside a suite.
type SuiteStep struct {
	Mode     Mode    `json:"mode"`
	Scenario string  `json:"scenario"`
	Seeds    []int64 `json:"seeds,omitempty"`
	Runs     int     `json:"runs,omitempty"`
	Ticks    int     `json:"ticks,omitempty"`
}

// SuiteSpec is a deterministic list of simulation runs.
type SuiteSpec struct {
	SchemaVersion int         `json:"schemaVersion"`
	Name          string      `json:"name"`
	Description   string      `json:"description,omitempty"`
	Steps         []SuiteStep `json:"steps"`
}

// LoadSuite parses and validates a suite JSON5 file.
func LoadSuite(path string) (*SuiteSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var suite SuiteSpec
	if err := json5.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("parse suite %s: %w", path, err)
	}
	suite.ApplyDefaults()
	if err := suite.Validate(); err != nil {
		return nil, fmt.Errorf("suite %s: %w", path, err)
	}
	return &suite, nil
}

// ApplyDefaults fills safe defaults.
func (s *SuiteSpec) ApplyDefaults() {
	if s.SchemaVersion <= 0 {
		s.SchemaVersion = 1
	}
	for i := range s.Steps {
		if s.Steps[i].Runs <= 0 {
			s.Steps[i].Runs = 1
		}
	}
}

// Validate checks supported modes/scenarios.
func (s *SuiteSpec) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(s.Steps) == 0 {
		return fmt.Errorf("steps must not be empty")
	}
	for i, step := range s.Steps {
		if strings.TrimSpace(step.Scenario) == "" {
			return fmt.Errorf("steps[%d].scenario is required", i)
		}
		switch step.Mode {
		case ModeShot, ModePenetration, ModeDuel, ModeMacro:
		default:
			return fmt.Errorf("steps[%d].mode %q is not supported by suite runner", i, step.Mode)
		}
		if step.Mode == ModeShot && step.Runs < 1 {
			return fmt.Errorf("steps[%d].runs must be >= 1", i)
		}
	}
	return nil
}

// ResolveSuitePath resolves an absolute/relative suite or a name under lab/suites.
func ResolveSuitePath(paths Paths, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("suite path is empty")
	}
	candidates := []string{arg}
	if filepath.Ext(arg) == "" {
		candidates = append(candidates, arg+".json5", arg+".json")
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
		full := filepath.Join(paths.Suites, candidate)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			return full, nil
		}
	}
	return "", fmt.Errorf("suite %q not found (tried cwd and %s)", arg, paths.Suites)
}

// SuiteRunResult is one completed step run.
type SuiteRunResult struct {
	Step       int    `json:"step"`
	Mode       Mode   `json:"mode"`
	Scenario   string `json:"scenario"`
	Seed       int64  `json:"seed"`
	RunID      string `json:"runId"`
	Fidelity   string `json:"fidelity"`
	ReportHTML string `json:"reportHtml"`
}

// SuiteSummary is written after a suite finishes.
type SuiteSummary struct {
	SchemaVersion int              `json:"schemaVersion"`
	Name          string           `json:"name"`
	StartedAt     string           `json:"startedAt"`
	Command       string           `json:"command"`
	GoVersion     string           `json:"goVersion"`
	Runs          []SuiteRunResult `json:"runs"`
}

// WriteSuiteSummary writes a suite summary JSON.
func WriteSuiteSummary(
	paths Paths,
	suite *SuiteSpec,
	runs []SuiteRunResult,
	command string,
	startedAt time.Time,
) (string, error) {
	dir := filepath.Join(paths.Reports, "suites")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	summary := SuiteSummary{
		SchemaVersion: 1, Name: suite.Name, StartedAt: startedAt.Format(time.RFC3339),
		Command: command, GoVersion: runtime.Version(), Runs: runs,
	}
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, suite.Name+"-"+startedAt.Format("20060102-150405")+".json")
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
