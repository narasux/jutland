package sim

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FrameReport is a rendered-frame p99 summary.
type FrameReport struct {
	Frames int     `json:"frames"`
	P50MS  float64 `json:"p50Ms"`
	P95MS  float64 `json:"p95Ms"`
	P99MS  float64 `json:"p99Ms"`
	MaxMS  float64 `json:"maxMs"`
	Budget float64 `json:"budgetMs"`
	Pass   bool    `json:"pass"`
}

// BuildFrameReport computes percentiles from rendered frame durations.
func BuildFrameReport(times []time.Duration, budgetMS float64) FrameReport {
	report := FrameReport{Frames: len(times), Budget: budgetMS}
	if len(times) == 0 {
		return report
	}
	sorted := append([]time.Duration(nil), times...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	pick := func(q float64) float64 {
		idx := int(float64(len(sorted)-1) * q)
		return float64(sorted[idx].Microseconds()) / 1000
	}
	report.P50MS, report.P95MS, report.P99MS = pick(0.50), pick(0.95), pick(0.99)
	report.MaxMS = float64(sorted[len(sorted)-1].Microseconds()) / 1000
	report.Pass = report.P99MS <= budgetMS
	return report
}

// WriteFrameReport writes a frame report JSON file.
func WriteFrameReport(path string, report FrameReport) error {
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(payload, '\n'), 0o644)
}

// FrameHistoryEntry is one rendered-frame gate result recorded for trend tracking.
type FrameHistoryEntry struct {
	RecordedAt time.Time `json:"recordedAt"`
	Commit     string    `json:"commit"`
	Dirty      bool      `json:"dirty"`
	Frames     int       `json:"frames"`
	P50MS      float64   `json:"p50Ms"`
	P95MS      float64   `json:"p95Ms"`
	P99MS      float64   `json:"p99Ms"`
	MaxMS      float64   `json:"maxMs"`
	BudgetMS   float64   `json:"budgetMs"`
	Pass       bool      `json:"pass"`
}

// NewFrameHistoryEntry converts a gate result into a history record.
func NewFrameHistoryEntry(report FrameReport, commit string, dirty bool, at time.Time) FrameHistoryEntry {
	return FrameHistoryEntry{
		RecordedAt: at, Commit: commit, Dirty: dirty,
		Frames: report.Frames, P50MS: report.P50MS, P95MS: report.P95MS,
		P99MS: report.P99MS, MaxMS: report.MaxMS, BudgetMS: report.Budget, Pass: report.Pass,
	}
}

// FrameHistoryPath returns the rendered-frame history path under the report dir.
func FrameHistoryPath(reportDir string) string {
	return filepath.Join(reportDir, "frame", "history.jsonl")
}

// AppendFrameHistory appends one rendered-frame gate result to frame/history.jsonl.
func AppendFrameHistory(reportDir string, entry FrameHistoryEntry) error {
	path := FrameHistoryPath(reportDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(payload, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// LoadFrameHistory reads frame/history.jsonl in chronological order, skipping malformed lines.
func LoadFrameHistory(reportDir string) ([]FrameHistoryEntry, error) {
	data, err := os.ReadFile(FrameHistoryPath(reportDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []FrameHistoryEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry FrameHistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].RecordedAt.Before(entries[j].RecordedAt)
	})
	return entries, nil
}

// FrameTrend summarizes rendered-frame p99 history for the trend view.
type FrameTrend struct {
	Samples     int     `json:"samples"`
	LatestP99MS float64 `json:"latestP99Ms"`
	LatestPass  bool    `json:"latestPass"`
	AvgP99MS    float64 `json:"avgP99Ms"`
	WorstP99MS  float64 `json:"worstP99Ms"`
	BudgetMS    float64 `json:"budgetMs"`
}

// BuildFrameTrend aggregates rendered-frame gate history; empty input yields zero samples.
func BuildFrameTrend(entries []FrameHistoryEntry) FrameTrend {
	trend := FrameTrend{Samples: len(entries)}
	if len(entries) == 0 {
		return trend
	}
	var total float64
	for _, entry := range entries {
		total += entry.P99MS
		if entry.P99MS > trend.WorstP99MS {
			trend.WorstP99MS = entry.P99MS
		}
	}
	latest := entries[len(entries)-1]
	trend.LatestP99MS = latest.P99MS
	trend.LatestPass = latest.Pass
	trend.BudgetMS = latest.BudgetMS
	trend.AvgP99MS = total / float64(len(entries))
	return trend
}
