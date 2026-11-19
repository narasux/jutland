package sim

import "time"

// ScorecardItem is one row of the balance scorecard.
type ScorecardItem struct {
	Dimension string `json:"dimension"`
	Metric    string `json:"metric"`
	Value     string `json:"value"`
	Target    string `json:"target"`
	Status    string `json:"status"` // green | yellow | red
	Note      string `json:"note,omitempty"`
}

// Evidence is a single numeric clue attached to a finding.
type Evidence struct {
	Metric   string  `json:"metric"`
	Value    float64 `json:"value"`
	Baseline float64 `json:"baseline,omitempty"`
	Delta    float64 `json:"delta,omitempty"`
	CI       string  `json:"ci,omitempty"`
	File     string  `json:"file,omitempty"`
}

// Finding is one structured bug/perf/balance/design/noise candidate.
type Finding struct {
	ID                  string     `json:"id"`
	Kind                string     `json:"kind"`
	Severity            string     `json:"severity"`
	Title               string     `json:"title"`
	Summary             string     `json:"summary"`
	Confidence          float64    `json:"confidence"`
	Evidence            []Evidence `json:"evidence"`
	Affected            []string   `json:"affected,omitempty"`
	SuggestedExperiment string     `json:"suggestedExperiment,omitempty"`
	Repro               string     `json:"repro,omitempty"`
	FirstSeen           string     `json:"firstSeen,omitempty"`
	LastSeen            string     `json:"lastSeen,omitempty"`
	Status              string     `json:"status"` // new | persisting | fixed | noise
}

// Diagnostic is a rule-engine candidate, separate from LLM analysis.
type Diagnostic struct {
	ID       string     `json:"id"`
	Severity string     `json:"severity"`
	Message  string     `json:"message"`
	Repro    string     `json:"repro,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

// MetricRow is one human-readable aggregate metric.
type MetricRow struct {
	Group    string  `json:"group"`
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
	CI       string  `json:"ci,omitempty"`
	Baseline float64 `json:"baseline,omitempty"`
	Delta    float64 `json:"delta,omitempty"`
}

// HitRatePoint is a hit-rate curve point for one ammo type.
type HitRatePoint struct {
	Range string  `json:"range"`
	AP    float64 `json:"ap"`
	HE    float64 `json:"he"`
}

// SurvivalPoint is one point on the kill-probability curve.
type SurvivalPoint struct {
	Minute      int     `json:"minute"`
	Probability float64 `json:"probability"`
}

// ExchangeCell is one class-vs-class exchange ratio.
type ExchangeCell struct {
	Attacker string  `json:"attacker"`
	Defender string  `json:"defender"`
	Ratio    float64 `json:"ratio"`
}

// PenetrationPoint is one range point on the penetration envelope.
type PenetrationPoint struct {
	Range string  `json:"range"`
	AP    float64 `json:"ap"`
	HE    float64 `json:"he"`
	Armor float64 `json:"armor"`
}

// Charts is the chart data shared by the human and AI views.
type Charts struct {
	HitRate        []HitRatePoint     `json:"hitRate"`
	Survival       []SurvivalPoint    `json:"survival"`
	Exchange       []ExchangeCell     `json:"exchange"`
	Penetration    []PenetrationPoint `json:"penetration"`
	HitRateLabels  []string           `json:"hitRateLabels"`
	ExchangeRows   []string           `json:"exchangeRows"`
	ExchangeCols   []string           `json:"exchangeCols"`
	PenetrationMax float64            `json:"penetrationMax"`
}

// Result is the canonical machine-readable output of a run.
type Result struct {
	SchemaVersion int             `json:"schemaVersion"`
	Mock          bool            `json:"mock"`
	Fidelity      string          `json:"fidelity"`
	Scenario      Scenario        `json:"scenario"`
	Seed          int64           `json:"seed"`
	Runs          int             `json:"runs"`
	StartedAt     time.Time       `json:"startedAt"`
	DurationMS    int64           `json:"durationMs"`
	Scorecard     []ScorecardItem `json:"scorecard"`
	PreviousDiff  *CompareReport  `json:"previousDiff,omitempty"`
	Metrics       []MetricRow     `json:"metrics"`
	Findings      []Finding       `json:"findings"`
	Diagnostics   []Diagnostic    `json:"diagnostics"`
	Charts        Charts          `json:"charts"`
	DataQuality   []string        `json:"dataQuality"`
}

// Manifest is the run's provenance and path record.
type Manifest struct {
	SchemaVersion  int      `json:"schemaVersion"`
	RunID          string   `json:"runId"`
	Mock           bool     `json:"mock"`
	Fidelity       string   `json:"fidelity"`
	StartedAt      string   `json:"startedAt"`
	FinishedAt     string   `json:"finishedAt"`
	DurationMS     int64    `json:"durationMs"`
	SimHome        string   `json:"simHome"`
	ReportDir      string   `json:"reportDir"`
	ScenarioPath   string   `json:"scenarioPath"`
	ScenarioHash   string   `json:"scenarioHash"`
	ConfigHash     string   `json:"configHash"`
	Seed           int64    `json:"seed"`
	Runs           int      `json:"runs"`
	MaxTicks       int      `json:"maxTicks"`
	Git            GitInfo  `json:"git"`
	LabCommit      string   `json:"labCommit,omitempty"`
	EChartsSource  string   `json:"echartsSource"`
	EChartsVersion string   `json:"echartsVersion"`
	GoVersion      string   `json:"goVersion"`
	GOOS           string   `json:"goos"`
	GOARCH         string   `json:"goarch"`
	Command        string   `json:"command"`
	Files          []string `json:"files"`
}

// ScorecardCounts is the compact scorecard summary used by history/trend.
type ScorecardCounts struct {
	Green  int `json:"green"`
	Yellow int `json:"yellow"`
	Red    int `json:"red"`
}

// FindingCounts is the compact finding summary used by history/trend.
type FindingCounts struct {
	New        int `json:"new"`
	Persisting int `json:"persisting"`
	Fixed      int `json:"fixed"`
	Noise      int `json:"noise"`
}

// HistoryEntry is one append-only line in reports/history.jsonl.
type HistoryEntry struct {
	RunID        string          `json:"runId"`
	Date         string          `json:"date"`
	StartedAt    string          `json:"startedAt"`
	Suite        string          `json:"suite"`
	Mock         bool            `json:"mock"`
	Fidelity     string          `json:"fidelity"`
	Commit       string          `json:"commit"`
	Branch       string          `json:"branch"`
	Dirty        bool            `json:"dirty"`
	WorktreeHash string          `json:"worktreeHash"`
	Config       string          `json:"config"`
	Scorecard    ScorecardCounts `json:"scorecard"`
	Findings     FindingCounts   `json:"findings"`
	RuntimeMS    int64           `json:"runtimeMs"`
	Report       string          `json:"report"`
}
