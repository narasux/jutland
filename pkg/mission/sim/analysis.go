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

// FindingGroup is a finding aggregated across the analysis window.
type FindingGroup struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Count      int    `json:"count"`
	FirstSeen  string `json:"firstSeen"`
	LastSeen   string `json:"lastSeen"`
	LastStatus string `json:"lastStatus"`
	Fixed      bool   `json:"fixed"`
}

// ScoreGroup is a red scorecard row aggregated across the window.
type ScoreGroup struct {
	Dimension string `json:"dimension"`
	Metric    string `json:"metric"`
	Target    string `json:"target"`
	Count     int    `json:"count"`
}

// ScenarioSummary aggregates repeated runs of one scenario.
type ScenarioSummary struct {
	Scenario      string  `json:"scenario"`
	Runs          int     `json:"runs"`
	MeanShots     float64 `json:"meanShots"`
	MeanOwnSunk   float64 `json:"meanOwnSunk"`
	MeanEnemySunk float64 `json:"meanEnemySunk"`
	MeanP99MS     float64 `json:"meanP99Ms"`
}

// AnalysisBrief is a cross-run, LLM-ready brief derived from history/result files.
type AnalysisBrief struct {
	SchemaVersion     int               `json:"schemaVersion"`
	GeneratedAt       time.Time         `json:"generatedAt"`
	Runs              int               `json:"runs"`
	Since             string            `json:"since,omitempty"`
	Until             string            `json:"until,omitempty"`
	Green             int               `json:"green"`
	Yellow            int               `json:"yellow"`
	Red               int               `json:"red"`
	NewFindings       int               `json:"newFindings"`
	RecurringFindings []FindingGroup    `json:"recurringFindings"`
	RedScorecards     []ScoreGroup      `json:"redScorecards"`
	ScenarioSummaries []ScenarioSummary `json:"scenarioSummaries"`
	RecentRuns        []TrendPoint      `json:"recentRuns"`
	Prompt            string            `json:"prompt"`
}

// AnalysisOutput lists generated analysis files.
type AnalysisOutput struct {
	Markdown string
	JSON     string
}

// WriteAnalysisBrief reads the latest history window and writes ai/analysis-brief.*.
func WriteAnalysisBrief(paths Paths, limit int) (AnalysisOutput, error) {
	entries, err := LoadHistory(paths.Reports)
	if err != nil {
		return AnalysisOutput{}, err
	}
	if limit <= 0 || limit > 500 {
		limit = 20
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	brief := AnalysisBrief{
		SchemaVersion: 1,
		GeneratedAt:   time.Now(),
		Runs:          len(entries),
		Prompt:        analysisPrompt,
	}
	if len(entries) > 0 {
		brief.Since = entries[0].StartedAt
		brief.Until = entries[len(entries)-1].StartedAt
	}
	findings := map[string]*FindingGroup{}
	scores := map[string]*ScoreGroup{}
	type scenarioAgg struct {
		runs, shots, ownSunk, enemySunk, p99 float64
	}
	scenarios := map[string]*scenarioAgg{}
	for _, entry := range entries {
		brief.Green += entry.Scorecard.Green
		brief.Yellow += entry.Scorecard.Yellow
		brief.Red += entry.Scorecard.Red
		brief.NewFindings += entry.Findings.New
		brief.RecentRuns = append(brief.RecentRuns, TrendPoint{
			RunID: entry.RunID, StartedAt: entry.StartedAt, Fidelity: entry.Fidelity,
			Commit: entry.Commit, Dirty: entry.Dirty,
			Green: entry.Scorecard.Green, Yellow: entry.Scorecard.Yellow, Red: entry.Scorecard.Red,
			NewFinding: entry.Findings.New, RuntimeMS: entry.RuntimeMS,
		})
		result, err := loadRunResult(paths.Reports, entry)
		if err != nil {
			continue
		}
		for _, finding := range result.Findings {
			if finding.Status == "fixed" || finding.Status == "noise" {
				continue
			}
			group := findings[finding.ID]
			if group == nil {
				group = &FindingGroup{
					ID: finding.ID, Kind: finding.Kind, Severity: finding.Severity, Title: finding.Title,
				}
				findings[finding.ID] = group
			}
			group.Count++
			if group.FirstSeen == "" {
				group.FirstSeen = entry.StartedAt
			}
			group.LastSeen = entry.StartedAt
			group.LastStatus = finding.Status
		}
		if result.Scenario.Name != "" {
			agg := scenarios[result.Scenario.Name]
			if agg == nil {
				agg = &scenarioAgg{}
				scenarios[result.Scenario.Name] = agg
			}
			agg.runs++
			for _, row := range result.Metrics {
				switch row.Name {
				case "shots":
					agg.shots += row.Value
				case "own_sunk":
					agg.ownSunk += row.Value
				case "enemy_sunk":
					agg.enemySunk += row.Value
				case "tick_p99_ms":
					agg.p99 += row.Value
				}
			}
		}
		for _, item := range result.Scorecard {
			if item.Status != "red" {
				continue
			}
			key := item.Dimension + "|" + item.Metric
			group := scores[key]
			if group == nil {
				group = &ScoreGroup{Dimension: item.Dimension, Metric: item.Metric, Target: item.Target}
				scores[key] = group
			}
			group.Count++
		}
	}
	if len(entries) > 0 {
		latest := entries[len(entries)-1]
		for id, group := range findings {
			if group.LastSeen != latest.StartedAt {
				group.Fixed = true
				findings[id] = group
			}
		}
	}
	for _, group := range findings {
		brief.RecurringFindings = append(brief.RecurringFindings, *group)
	}
	sort.Slice(brief.RecurringFindings, func(i, j int) bool {
		if brief.RecurringFindings[i].Count != brief.RecurringFindings[j].Count {
			return brief.RecurringFindings[i].Count > brief.RecurringFindings[j].Count
		}
		return brief.RecurringFindings[i].ID < brief.RecurringFindings[j].ID
	})
	for _, group := range scores {
		brief.RedScorecards = append(brief.RedScorecards, *group)
	}
	sort.Slice(brief.RedScorecards, func(i, j int) bool {
		if brief.RedScorecards[i].Count != brief.RedScorecards[j].Count {
			return brief.RedScorecards[i].Count > brief.RedScorecards[j].Count
		}
		return brief.RedScorecards[i].Metric < brief.RedScorecards[j].Metric
	})
	for name, agg := range scenarios {
		if agg.runs == 0 {
			continue
		}
		brief.ScenarioSummaries = append(brief.ScenarioSummaries, ScenarioSummary{
			Scenario: name, Runs: int(agg.runs), MeanShots: roundBaseline3(agg.shots / agg.runs),
			MeanOwnSunk:   roundBaseline3(agg.ownSunk / agg.runs),
			MeanEnemySunk: roundBaseline3(agg.enemySunk / agg.runs),
			MeanP99MS:     roundBaseline3(agg.p99 / agg.runs),
		})
	}
	sort.Slice(brief.ScenarioSummaries, func(i, j int) bool {
		return brief.ScenarioSummaries[i].Scenario < brief.ScenarioSummaries[j].Scenario
	})
	if len(brief.RecentRuns) > 10 {
		brief.RecentRuns = brief.RecentRuns[len(brief.RecentRuns)-10:]
	}
	aiDir := filepath.Join(paths.Reports, "ai")
	if err := os.MkdirAll(aiDir, 0o755); err != nil {
		return AnalysisOutput{}, err
	}
	mdPath := filepath.Join(aiDir, "analysis-brief.md")
	if err := writeFile(mdPath, []byte(analysisMarkdown(brief))); err != nil {
		return AnalysisOutput{}, err
	}
	jsonPath := filepath.Join(aiDir, "analysis-brief.json")
	payload, err := json.MarshalIndent(brief, "", "  ")
	if err != nil {
		return AnalysisOutput{}, err
	}
	if err := writeFile(jsonPath, append(payload, '\n')); err != nil {
		return AnalysisOutput{}, err
	}
	return AnalysisOutput{Markdown: mdPath, JSON: jsonPath}, nil
}

func loadRunResult(reportDir string, entry HistoryEntry) (*Result, error) {
	runDir := filepath.Dir(filepath.Join(reportDir, filepath.FromSlash(entry.Report)))
	data, err := os.ReadFile(filepath.Join(runDir, "result.json"))
	if err != nil {
		return nil, err
	}
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func analysisMarkdown(brief AnalysisBrief) string {
	var b strings.Builder
	b.WriteString("# Jutland simulation analysis brief\n\n")
	fmt.Fprintf(&b, "- generated: %s\n", brief.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- window: %s → %s（%d runs）\n", brief.Since, brief.Until, brief.Runs)
	fmt.Fprintf(&b, "- scorecard: green %d / yellow %d / red %d\n", brief.Green, brief.Yellow, brief.Red)
	fmt.Fprintf(&b, "- new findings: %d\n\n", brief.NewFindings)
	b.WriteString("## 反复出现的 Findings\n\n")
	if len(brief.RecurringFindings) == 0 {
		b.WriteString("无。\n\n")
	} else {
		b.WriteString("| count | severity | id | title | first | last | fixed |\n|---|---|---|---|---|---|---|\n")
		for _, f := range brief.RecurringFindings {
			fmt.Fprintf(&b, "| %d | %s | `%s` | %s | %s | %s | %t |\n", f.Count, f.Severity, f.ID, f.Title, f.FirstSeen, f.LastSeen, f.Fixed)
		}
		b.WriteString("\n")
	}
	b.WriteString("## 反复出现的红色指标\n\n")
	if len(brief.RedScorecards) == 0 {
		b.WriteString("无。\n\n")
	} else {
		b.WriteString("| count | dimension | metric | target |\n|---|---|---|---|\n")
		for _, s := range brief.RedScorecards {
			fmt.Fprintf(&b, "| %d | %s | %s | %s |\n", s.Count, s.Dimension, s.Metric, s.Target)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Scenario 汇总\n\n")
	if len(brief.ScenarioSummaries) == 0 {
		b.WriteString("无。\n\n")
	} else {
		b.WriteString("| scenario | runs | mean shots | mean own sunk | mean enemy sunk | mean p99 ms |\n|---|---|---|---|---|---|\n")
		for _, s := range brief.ScenarioSummaries {
			fmt.Fprintf(&b, "| %s | %d | %.1f | %.1f | %.1f | %.3f |\n", s.Scenario, s.Runs, s.MeanShots, s.MeanOwnSunk, s.MeanEnemySunk, s.MeanP99MS)
		}
		b.WriteString("\n")
	}
	b.WriteString("## 最近 Runs\n\n| started | fidelity | green | red | findings | ms |\n|---|---|---|---|---|---|\n")
	for _, run := range brief.RecentRuns {
		fmt.Fprintf(
			&b,
			"| %s | %s | %d | %d | %d | %d |\n",
			run.StartedAt,
			run.Fidelity,
			run.Green,
			run.Red,
			run.NewFinding,
			run.RuntimeMS,
		)
	}
	b.WriteString("\n## Prompt\n\n")
	b.WriteString(brief.Prompt)
	b.WriteString("\n")
	return b.String()
}

const analysisPrompt = `请基于本文件与同目录的 analysis-brief.json / findings.json / result.json，输出：
1. 最高优先级的 bug 或回归（带 runId 与证据）；
2. 可复现的平衡问题（指标、基线 delta、影响范围）；
3. 建议的下一轮实验（场景、seed、指标、预期方向）；
4. 明确区分「数据/实现 bug」「平衡问题」「样本不足」。
不要仅复述表格；每个结论都要引用具体 runId 或 metric。`
