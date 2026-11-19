package sim

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/narasux/jutland/pkg/mission/observer"
)

// RecordingSummary is the aggregate of a debug JSONL recording.
type RecordingSummary struct {
	Path            string  `json:"path"`
	Shots           int     `json:"shots"`
	Hits            int     `json:"hits"`
	Sinks           int     `json:"sinks"`
	PlaneLosses     int     `json:"planeLosses"`
	AppliedDamage   float64 `json:"appliedDamage"`
	EffectiveDamage float64 `json:"effectiveDamage"`
	FriendlyHits    int     `json:"friendlyHits"`
	FirstTick       int64   `json:"firstTick"`
	LastTick        int64   `json:"lastTick"`
}

// SummarizeRecording reads the env-gated game JSONL stream.
func SummarizeRecording(path string) (RecordingSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return RecordingSummary{}, err
	}
	defer file.Close()
	summary := RecordingSummary{Path: path}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var event observer.StreamEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return RecordingSummary{}, fmt.Errorf("parse recording: %w", err)
		}
		switch event.Type {
		case "shot":
			var e observer.Shot
			if err := remarshal(event.Event, &e); err != nil {
				return RecordingSummary{}, err
			}
			summary.Shots++
			updateTick(&summary, e.Tick)
		case "hit":
			var e observer.Hit
			if err := remarshal(event.Event, &e); err != nil {
				return RecordingSummary{}, err
			}
			summary.Hits++
			summary.AppliedDamage += e.AppliedDamage
			summary.EffectiveDamage += e.EffectiveDamage
			if e.FriendlyFire {
				summary.FriendlyHits++
			}
			updateTick(&summary, e.Tick)
		case "sink":
			var e observer.Sink
			if err := remarshal(event.Event, &e); err != nil {
				return RecordingSummary{}, err
			}
			summary.Sinks++
			updateTick(&summary, e.Tick)
		case "loss":
			var e observer.Loss
			if err := remarshal(event.Event, &e); err != nil {
				return RecordingSummary{}, err
			}
			summary.PlaneLosses++
			updateTick(&summary, e.Tick)
		}
	}
	if err := scanner.Err(); err != nil {
		return RecordingSummary{}, err
	}
	return summary, nil
}

func remarshal(value any, out any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, out)
}

func updateTick(summary *RecordingSummary, tick int64) {
	if summary.FirstTick == 0 || tick < summary.FirstTick {
		summary.FirstTick = tick
	}
	if tick > summary.LastTick {
		summary.LastTick = tick
	}
}

// WriteRecordingComparison compares a debug recording to a simulation run.
func WriteRecordingComparison(paths Paths, recordingPath, runArg string) (CompareOutput, error) {
	rec, err := SummarizeRecording(recordingPath)
	if err != nil {
		return CompareOutput{}, err
	}
	run, runID, err := loadComparisonRun(paths, runArg)
	if err != nil {
		return CompareOutput{}, err
	}
	simMetrics := metricMap(run.Metrics)
	rows := []struct {
		name string
		rec  float64
		sim  float64
	}{
		{"shots", float64(rec.Shots), simMetrics["shots"].Value},
		{"hits", float64(rec.Hits), simMetrics["hits"].Value},
		{"applied_damage", rec.AppliedDamage, simMetrics["applied_damage"].Value},
		{"sinks", float64(rec.Sinks), simMetrics["own_sunk"].Value + simMetrics["enemy_sunk"].Value},
		{"friendly_hits", float64(rec.FriendlyHits), simMetrics["friendly_hits"].Value},
	}
	report := CompareReport{
		SchemaVersion: 1, GeneratedAt: time.Now(), ARun: rec.Path, BRun: runID,
		AMeta: "game recording", BMeta: run.Scenario.Name,
	}
	for _, row := range rows {
		metric := CompareMetric{
			Group: "recording", Name: row.name, A: roundCompare(row.rec), B: roundCompare(row.sim),
			Delta: roundCompare(row.sim - row.rec),
		}
		if row.rec != 0 {
			metric.DeltaPct = roundCompare((row.sim - row.rec) / row.rec * 100)
		}
		report.Metrics = append(report.Metrics, metric)
	}
	sort.Slice(report.Metrics, func(i, j int) bool { return report.Metrics[i].Name < report.Metrics[j].Name })
	dir := filepath.Join(paths.Reports, "compare")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return CompareOutput{}, err
	}
	base := strings.TrimSuffix(filepath.Base(rec.Path), filepath.Ext(rec.Path)) + "__" + runID
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
