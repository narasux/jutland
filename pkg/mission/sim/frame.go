package sim

import (
	"encoding/json"
	"os"
	"sort"
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
