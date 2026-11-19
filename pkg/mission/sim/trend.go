package sim

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TrendPoint is one run in the historical trend view.
type TrendPoint struct {
	RunID      string `json:"runId"`
	StartedAt  string `json:"startedAt"`
	Fidelity   string `json:"fidelity"`
	Commit     string `json:"commit"`
	Dirty      bool   `json:"dirty"`
	Green      int    `json:"green"`
	Yellow     int    `json:"yellow"`
	Red        int    `json:"red"`
	NewFinding int    `json:"newFindings"`
	RuntimeMS  int64  `json:"runtimeMs"`
}

// TrendSummary aggregates the visible history window.
type TrendSummary struct {
	Runs        int     `json:"runs"`
	Green       int     `json:"green"`
	Yellow      int     `json:"yellow"`
	Red         int     `json:"red"`
	NewFindings int     `json:"newFindings"`
	AvgRuntime  float64 `json:"avgRuntimeMs"`
}

// TrendData is the canonical trend output shared by HTML/Markdown/AI views.
type TrendData struct {
	SchemaVersion int          `json:"schemaVersion"`
	GeneratedAt   time.Time    `json:"generatedAt"`
	Points        []TrendPoint `json:"points"`
	Summary       TrendSummary `json:"summary"`
	// Frame 来自 frame/history.jsonl：渲染帧 p99 门禁的最近结果与均值。
	Frame FrameTrend `json:"frame"`
}

// TrendOutput lists the generated trend files.
type TrendOutput struct {
	HTML   string
	MD     string
	AIJSON string
}

// LoadHistory reads history.jsonl, skipping malformed lines.
func LoadHistory(reportDir string) ([]HistoryEntry, error) {
	path := filepath.Join(reportDir, "history.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []HistoryEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].StartedAt == entries[j].StartedAt {
			return entries[i].RunID < entries[j].RunID
		}
		return entries[i].StartedAt < entries[j].StartedAt
	})
	return entries, nil
}

// BuildTrend converts history entries (already chronological) into chart data.
func BuildTrend(entries []HistoryEntry) TrendData {
	data := TrendData{SchemaVersion: 1, GeneratedAt: time.Now()}
	for _, e := range entries {
		point := TrendPoint{
			RunID: e.RunID, StartedAt: e.StartedAt, Fidelity: e.Fidelity,
			Commit: e.Commit, Dirty: e.Dirty,
			Green: e.Scorecard.Green, Yellow: e.Scorecard.Yellow, Red: e.Scorecard.Red,
			NewFinding: e.Findings.New, RuntimeMS: e.RuntimeMS,
		}
		data.Points = append(data.Points, point)
		data.Summary.Green += point.Green
		data.Summary.Yellow += point.Yellow
		data.Summary.Red += point.Red
		data.Summary.NewFindings += point.NewFinding
	}
	data.Summary.Runs = len(data.Points)
	if data.Summary.Runs > 0 {
		var total int64
		for _, p := range data.Points {
			total += p.RuntimeMS
		}
		data.Summary.AvgRuntime = float64(total) / float64(data.Summary.Runs)
	}
	if len(data.Points) > 50 {
		data.Points = data.Points[len(data.Points)-50:]
	}
	return data
}

// WriteTrend writes trend.html, trend.md and ai/trend.json under the report dir.
func WriteTrend(paths Paths, limit int) (TrendOutput, error) {
	entries, err := LoadHistory(paths.Reports)
	if err != nil {
		return TrendOutput{}, err
	}
	if limit <= 0 || limit > 2000 {
		limit = 50
	}
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	data := BuildTrend(entries)
	frames, err := LoadFrameHistory(paths.Reports)
	if err != nil {
		return TrendOutput{}, err
	}
	if len(frames) > 50 {
		frames = frames[len(frames)-50:]
	}
	data.Frame = BuildFrameTrend(frames)
	labels := make([]string, 0, len(data.Points))
	green := make([]float64, 0, len(data.Points))
	red := make([]float64, 0, len(data.Points))
	runtime := make([]float64, 0, len(data.Points))
	for i, p := range data.Points {
		labels = append(labels, fmt.Sprintf("#%d", i+1))
		green = append(green, float64(p.Green))
		red = append(red, float64(p.Red))
		runtime = append(runtime, float64(p.RuntimeMS))
	}
	view := struct {
		TrendData
		ScoreSVG   template.HTML
		RuntimeSVG template.HTML
	}{
		TrendData: data,
		ScoreSVG: template.HTML(lineChartSVG("记分卡趋势", labels, []SVGSeries{
			{Name: "green", Color: "#2f9e6f", Values: green},
			{Name: "red", Color: "#b23a48", Values: red},
		}, 720, 280, trendMaxFloat(append(green, red...)))),
		RuntimeSVG: template.HTML(lineChartSVG("耗时趋势", labels, []SVGSeries{
			{Name: "ms", Color: "#2d6cdf", Values: runtime},
		}, 720, 280, trendMaxFloat(runtime))),
	}
	if err := os.MkdirAll(filepath.Join(paths.Reports, "ai"), 0o755); err != nil {
		return TrendOutput{}, err
	}
	htmlPath := filepath.Join(paths.Reports, "trend.html")
	if err := writeTemplate(htmlPath, trendTemplate, view); err != nil {
		return TrendOutput{}, err
	}
	mdPath := filepath.Join(paths.Reports, "trend.md")
	if err := writeFile(mdPath, []byte(trendMarkdown(data))); err != nil {
		return TrendOutput{}, err
	}
	aiPath := filepath.Join(paths.Reports, "ai", "trend.json")
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return TrendOutput{}, err
	}
	if err := writeFile(aiPath, append(payload, '\n')); err != nil {
		return TrendOutput{}, err
	}
	if err := writeIndex(paths.Reports); err != nil {
		return TrendOutput{}, err
	}
	return TrendOutput{HTML: htmlPath, MD: mdPath, AIJSON: aiPath}, nil
}

func trendMarkdown(data TrendData) string {
	var b strings.Builder
	b.WriteString("# Jutland simulation trend\n\n")
	fmt.Fprintf(&b, "- generated: %s\n", data.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- runs: %d\n", data.Summary.Runs)
	fmt.Fprintf(
		&b,
		"- scorecard: green %d / yellow %d / red %d\n",
		data.Summary.Green,
		data.Summary.Yellow,
		data.Summary.Red,
	)
	fmt.Fprintf(&b, "- new findings: %d\n", data.Summary.NewFindings)
	fmt.Fprintf(&b, "- avg runtime: %.0f ms\n", data.Summary.AvgRuntime)
	if data.Frame.Samples > 0 {
		status := "pass"
		if !data.Frame.LatestPass {
			status = "fail"
		}
		fmt.Fprintf(
			&b,
			"- frame gate: latest p99 %.3f ms (%s，预算 %.1f ms)，avg %.3f ms，worst %.3f ms，samples %d\n",
			data.Frame.LatestP99MS, status, data.Frame.BudgetMS,
			data.Frame.AvgP99MS, data.Frame.WorstP99MS, data.Frame.Samples,
		)
	}
	b.WriteString("\n")
	b.WriteString("| started | fidelity | commit | green | red | findings | ms |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, p := range data.Points {
		fmt.Fprintf(&b, "| %s | %s | `%s` | %d | %d | %d | %d |\n",
			p.StartedAt, p.Fidelity, p.Commit, p.Green, p.Red, p.NewFinding, p.RuntimeMS)
	}
	return b.String()
}

var trendTemplate = template.Must(template.New("trend").Parse(`<!doctype html>
<html lang="zh-Hans"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Jutland simulation trend</title>
<style>
body{margin:0;background:#f5f7fa;color:#172033;font:14px/1.6 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif}
main{max-width:1100px;margin:0 auto;padding:28px 22px 56px}h1{font-size:26px;margin:0 0 6px}.muted{color:#5b6675}
.card{background:#fff;border:1px solid #dce3ec;border-radius:12px;padding:18px 20px;margin:16px 0;box-shadow:0 1px 2px rgba(16,24,40,.06)}
table{width:100%;border-collapse:collapse;font-size:13px}th,td{text-align:left;padding:8px 10px;border-bottom:1px solid #e6ebf2}th{color:#5b6675;background:#fafbfc}
a{color:#2d6cdf;text-decoration:none}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:16px}
</style></head><body><main>
<h1>Jutland simulation trend</h1><p class="muted">最近 {{.Summary.Runs}} 次 run；生成于 {{.GeneratedAt.Format "2006-01-02 15:04:05"}}。索引：<a href="index.html">index.html</a> · 分析：<a href="ai/analysis-brief.md">analysis-brief.md</a></p>
<div class="grid"><div class="card"><h2>记分卡趋势</h2>{{.ScoreSVG}}</div><div class="card"><h2>耗时趋势</h2>{{.RuntimeSVG}}</div></div>
{{if .Frame.Samples}}<div class="card"><h2>渲染帧门禁</h2><p>latest p99 {{printf "%.3f" .Frame.LatestP99MS}} ms · avg {{printf "%.3f" .Frame.AvgP99MS}} ms · worst {{printf "%.3f" .Frame.WorstP99MS}} ms · budget {{printf "%.1f" .Frame.BudgetMS}} ms · samples {{.Frame.Samples}} · {{if .Frame.LatestPass}}<span style="color:#2f9e6f">pass</span>{{else}}<span style="color:#b23a48">fail</span>{{end}}</p></div>{{end}}
<div class="card"><h2>汇总</h2><p>green {{.Summary.Green}} · yellow {{.Summary.Yellow}} · red {{.Summary.Red}} · new findings {{.Summary.NewFindings}} · avg {{printf "%.0f" .Summary.AvgRuntime}} ms</p></div>
<div class="card"><h2>Runs</h2><table><thead><tr><th>时间</th><th>fidelity</th><th>commit</th><th>green</th><th>red</th><th>findings</th><th>ms</th></tr></thead><tbody>
{{range .Points}}<tr><td>{{.StartedAt}}</td><td>{{.Fidelity}}</td><td><code>{{.Commit}}</code></td><td>{{.Green}}</td><td>{{.Red}}</td><td>{{.NewFinding}}</td><td>{{.RuntimeMS}}</td></tr>{{else}}<tr><td colspan="7" class="muted">还没有 run。</td></tr>{{end}}
</tbody></table></div>
</main></body></html>`))

func trendMaxFloat(values []float64) float64 {
	maxValue := 0.0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
