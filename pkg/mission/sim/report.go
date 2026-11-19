package sim

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

//go:embed templates/report.html.tmpl
var reportTemplateText string

var reportTemplate = template.Must(template.New("report").Parse(reportTemplateText))

const (
	echartsVersion = "6.1.0"
	echartsSource  = "cdn"
)

// ChartView is one chart in the human report.
type ChartView struct {
	ID          string
	Title       string
	Description string
	File        string
	SVG         template.HTML
}

type reportTemplateData struct {
	Result      *Result
	Manifest    Manifest
	Git         GitInfo
	StartedAt   string
	Charts      []ChartView
	EChartsJSON template.JS
	Mock        bool
}

// ReportOutput describes the generated report files.
type ReportOutput struct {
	RunID      string
	Dir        string
	ReportHTML string
	Summary    string
	AIContext  string
}

// WriteReport writes the human HTML report and the machine-readable AI view.
// It does not touch the game runtime and is safe to call after a mock or a real run.
func WriteReport(paths Paths, result *Result, scenarioPath, command string) (ReportOutput, error) {
	if result == nil {
		return ReportOutput{}, fmt.Errorf("result is nil")
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = time.Now()
	}
	result.Diagnostics = mergeDiagnostics(result.Diagnostics, RunDiagnostics(result))
	result.PreviousDiff = PreviousComparison(paths, result)
	gameGit := CollectGitInfo("")
	labGit := CollectGitInfo(paths.Home)
	runID := uniqueRunID(paths.Reports, result.StartedAt, result.Scenario.Slug(), gameGit.CommitShort)
	runDir := filepath.Join(paths.Reports, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return ReportOutput{}, fmt.Errorf("create run dir: %w", err)
	}

	charts, chartOptions := buildCharts(result)
	echartsJSON, err := marshalChartOptions(chartOptions)
	if err != nil {
		return ReportOutput{}, err
	}
	fidelity := result.Fidelity
	if fidelity == "" {
		if result.Mock {
			fidelity = "mock"
		} else {
			fidelity = "unknown"
		}
	}
	manifest := Manifest{
		SchemaVersion:  1,
		RunID:          runID,
		Mock:           result.Mock,
		Fidelity:       fidelity,
		StartedAt:      result.StartedAt.Format(time.RFC3339),
		FinishedAt:     time.Now().Format(time.RFC3339),
		DurationMS:     result.DurationMS,
		SimHome:        paths.Home,
		ReportDir:      paths.Reports,
		ScenarioPath:   scenarioPath,
		ScenarioHash:   result.Scenario.Hash(),
		ConfigHash:     result.Scenario.Hash(),
		Seed:           result.Seed,
		Runs:           result.Runs,
		MaxTicks:       result.Scenario.MaxTicks,
		Git:            gameGit,
		LabCommit:      labGit.CommitShort,
		EChartsSource:  echartsSource,
		EChartsVersion: echartsVersion,
		GoVersion:      runtime.Version(),
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		Command:        command,
	}
	data := reportTemplateData{
		Result:      result,
		Manifest:    manifest,
		Git:         gameGit,
		StartedAt:   result.StartedAt.Format("2006-01-02 15:04:05 -07:00"),
		Charts:      charts,
		EChartsJSON: template.JS(echartsJSON),
		Mock:        result.Mock,
	}

	if err := writeTemplate(filepath.Join(runDir, "report.html"), reportTemplate, data); err != nil {
		return ReportOutput{}, err
	}
	if err := writeFile(filepath.Join(runDir, "summary.md"), []byte(buildSummary(result, manifest))); err != nil {
		return ReportOutput{}, err
	}
	if err := writeJSON(filepath.Join(runDir, "result.json"), result); err != nil {
		return ReportOutput{}, err
	}
	if err := writeJSON(filepath.Join(runDir, "ai", "findings.json"), result.Findings); err != nil {
		return ReportOutput{}, err
	}
	if err := writeJSON(filepath.Join(runDir, "ai", "diagnostics.json"), result.Diagnostics); err != nil {
		return ReportOutput{}, err
	}
	if err := writeJSON(filepath.Join(runDir, "ai", "metrics.json"), map[string]any{
		"scorecard": result.Scorecard, "metrics": result.Metrics, "dataQuality": result.DataQuality,
	}); err != nil {
		return ReportOutput{}, err
	}
	if err := writeJSON(filepath.Join(runDir, "ai", "changes.json"), map[string]any{
		"mock": result.Mock, "note": changeNote(result), "fidelity": result.Fidelity,
		"scenario": result.Scenario.Name, "seed": result.Seed, "runs": result.Runs,
		"commit": gameGit.Commit, "branch": gameGit.Branch,
		"dirty": gameGit.Dirty, "worktreeHash": gameGit.WorktreeHash, "configHash": manifest.ConfigHash,
	}); err != nil {
		return ReportOutput{}, err
	}
	if err := writeFile(filepath.Join(runDir, "ai", "analysis-context.md"), []byte(buildAIContext(result, manifest))); err != nil {
		return ReportOutput{}, err
	}
	if err := writeFile(filepath.Join(runDir, "ai", "llm-prompt.md"), []byte(llmPrompt)); err != nil {
		return ReportOutput{}, err
	}
	if err := writeJSON(filepath.Join(runDir, "replay.json"), map[string]any{
		"scenario": result.Scenario, "scenarioPath": scenarioPath, "seed": result.Seed,
		"command": command, "simHome": paths.Home, "commit": gameGit.Commit,
		"dirty": gameGit.Dirty, "worktreeHash": gameGit.WorktreeHash,
	}); err != nil {
		return ReportOutput{}, err
	}
	for _, chart := range charts {
		if err := writeFile(filepath.Join(runDir, "charts", chart.File), []byte(chart.SVG)); err != nil {
			return ReportOutput{}, err
		}
	}
	files, err := collectFiles(runDir)
	if err != nil {
		return ReportOutput{}, err
	}
	manifest.Files = files
	if err := writeJSON(filepath.Join(runDir, "manifest.json"), manifest); err != nil {
		return ReportOutput{}, err
	}
	if err := writeChecksums(runDir); err != nil {
		return ReportOutput{}, err
	}
	entry := historyFromResult(paths, runID, result, gameGit, manifest.ConfigHash)
	if err := appendHistory(paths.Reports, entry); err != nil {
		return ReportOutput{}, err
	}
	if err := writeIndex(paths.Reports); err != nil {
		return ReportOutput{}, err
	}
	return ReportOutput{
		RunID:      runID,
		Dir:        runDir,
		ReportHTML: filepath.Join(runDir, "report.html"),
		Summary:    filepath.Join(runDir, "summary.md"),
		AIContext:  filepath.Join(runDir, "ai", "analysis-context.md"),
	}, nil
}

func uniqueRunID(reportDir string, startedAt time.Time, slug, commit string) string {
	if slug == "" {
		slug = "scenario"
	}
	if commit == "" {
		commit = "nogit"
	}
	base := startedAt.Format("20060102-150405") + "_" + slug + "_" + commit
	id := base
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(reportDir, "runs", id)); os.IsNotExist(err) {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
}

func buildCharts(result *Result) ([]ChartView, map[string]any) {
	charts := result.Charts
	f0 := result.Fidelity == "F0"
	var views []ChartView
	options := map[string]any{}

	if len(charts.HitRate) > 0 {
		hitAP := make([]float64, 0, len(charts.HitRate))
		hitHE := make([]float64, 0, len(charts.HitRate))
		for _, p := range charts.HitRate {
			hitAP = append(hitAP, p.AP)
			hitHE = append(hitHE, p.HE)
		}
		yMax := 1.0
		title := "命中率 vs 距离"
		desc := "AP/HE 在不同射程比例下的单发命中率。"
		names := []string{"AP", "HE"}
		if f0 {
			title = "装甲减伤曲线"
			desc = "F0 纯伤害台：AP=水平装甲带，HE=垂直装甲带；横轴为减伤比例。"
			names = []string{"水平装甲", "垂直装甲"}
			yMax = math.Ceil(math.Max(maxFloat(hitAP), maxFloat(hitHE)) * 1.1)
			if yMax <= 0 {
				yMax = 1
			}
		}
		svg := lineChartSVG(title, charts.HitRateLabels, []SVGSeries{
			{Name: names[0], Color: svgAPColor, Values: hitAP},
			{Name: names[1], Color: svgHEColor, Values: hitHE},
		}, 720, 300, yMax)
		views = append(
			views,
			ChartView{
				ID:          "hit-rate",
				Title:       title,
				Description: desc,
				File:        "hit_rate_vs_range.svg",
				SVG:         template.HTML(svg),
			},
		)
		options["hit-rate"] = map[string]any{
			"animation": false,
			"tooltip":   map[string]any{"trigger": "axis"},
			"legend":    map[string]any{"data": names, "top": 0, "left": "center"},
			"grid":      map[string]any{"left": 48, "right": 18, "top": 48, "bottom": 30},
			"xAxis":     map[string]any{"type": "category", "data": charts.HitRateLabels},
			"yAxis":     map[string]any{"type": "value", "min": 0, "max": yMax},
			"series": []any{
				map[string]any{
					"name":      names[0],
					"type":      "line",
					"smooth":    true,
					"data":      hitAP,
					"itemStyle": map[string]any{"color": svgAPColor},
				},
				map[string]any{
					"name":      names[1],
					"type":      "line",
					"smooth":    true,
					"data":      hitHE,
					"itemStyle": map[string]any{"color": svgHEColor},
				},
			},
		}
	}

	if len(charts.Survival) > 0 {
		survival := make([]float64, 0, len(charts.Survival))
		labels := make([]string, 0, len(charts.Survival))
		for _, p := range charts.Survival {
			survival = append(survival, p.Probability)
			if f0 {
				labels = append(labels, fmt.Sprintf("%d", p.Minute))
			} else {
				labels = append(labels, fmt.Sprintf("%dm", p.Minute))
			}
		}
		title := "击沉概率曲线"
		desc := "从交火开始，t 分钟时目标已击沉的比例。"
		switch {
		case f0:
			title = "击沉概率 vs 射击数"
			desc = "F0 纯伤害台：累计射击 n 发时目标已击沉的比例。"
		case result.Fidelity == "F1" || result.Fidelity == "F2":
			title = "舰队生存率"
			desc = "F1/F2 对射：shooter 方剩余舰船比例随战斗分钟变化。"
		case result.Fidelity == "F3":
			title = "舰队生存率"
			desc = "F3 完整战斗：shooter 方剩余舰船比例随战斗分钟变化。"
		}
		svg := lineChartSVG(title, labels, []SVGSeries{
			{Name: "survival", Color: svgAPColor, Values: survival},
		}, 720, 300, 1.0)
		views = append(
			views,
			ChartView{
				ID:          "survival",
				Title:       title,
				Description: desc,
				File:        "survival_curve.svg",
				SVG:         template.HTML(svg),
			},
		)
		options["survival"] = map[string]any{
			"animation": false,
			"tooltip":   map[string]any{"trigger": "axis"},
			"grid":      map[string]any{"left": 48, "right": 18, "top": 24, "bottom": 32},
			"xAxis":     map[string]any{"type": "category", "data": labels},
			"yAxis":     map[string]any{"type": "value", "min": 0, "max": 1},
			"series": []any{
				map[string]any{
					"name":      "survival",
					"type":      "line",
					"step":      "end",
					"data":      survival,
					"itemStyle": map[string]any{"color": svgAPColor},
				},
			},
		}
	}

	if len(charts.Exchange) > 0 && len(charts.ExchangeRows) > 0 && len(charts.ExchangeCols) > 0 {
		exchangeValues := map[string]float64{}
		rowIndex := map[string]int{}
		colIndex := map[string]int{}
		for i, row := range charts.ExchangeRows {
			rowIndex[row] = i
		}
		for i, col := range charts.ExchangeCols {
			colIndex[col] = i
		}
		heatData := make([][3]any, 0, len(charts.Exchange))
		for _, c := range charts.Exchange {
			exchangeValues[c.Attacker+"|"+c.Defender] = c.Ratio
			heatData = append(heatData, [3]any{colIndex[c.Defender], rowIndex[c.Attacker], c.Ratio})
		}
		svg := heatmapSVG("交换矩阵", charts.ExchangeRows, charts.ExchangeCols, exchangeValues, 720, 340)
		views = append(
			views,
			ChartView{
				ID:          "exchange",
				Title:       "交换矩阵",
				Description: "行=攻方，列=守方，值=成本交换比；1.0 为均势。",
				File:        "exchange_matrix.svg",
				SVG:         template.HTML(svg),
			},
		)
		options["exchange"] = map[string]any{
			"animation": false,
			"tooltip":   map[string]any{"position": "top"},
			"grid":      map[string]any{"left": 48, "right": 72, "top": 26, "bottom": 36},
			"xAxis":     map[string]any{"type": "category", "data": charts.ExchangeCols},
			"yAxis":     map[string]any{"type": "category", "data": charts.ExchangeRows, "inverse": true},
			"visualMap": map[string]any{
				"min":        0.6,
				"max":        1.4,
				"calculable": true,
				"orient":     "vertical",
				"right":      0,
				"top":        "middle",
				"itemHeight": 100,
				"inRange":    map[string]any{"color": []string{"#b23a48", "#f2d16b", "#2f9e6f"}},
			},
			"series": []any{
				map[string]any{
					"name":  "exchange",
					"type":  "heatmap",
					"data":  heatData,
					"label": map[string]any{"show": true},
				},
			},
		}
	}

	if len(charts.Penetration) > 0 {
		penAP := make([]float64, 0, len(charts.Penetration))
		penHE := make([]float64, 0, len(charts.Penetration))
		penArmor := make([]float64, 0, len(charts.Penetration))
		for _, p := range charts.Penetration {
			penAP = append(penAP, p.AP)
			penHE = append(penHE, p.HE)
			penArmor = append(penArmor, p.Armor)
		}
		yMax := charts.PenetrationMax
		if yMax <= 0 {
			yMax = math.Max(maxFloat(penAP), math.Max(maxFloat(penHE), maxFloat(penArmor))) * 1.1
		}
		svg := lineChartSVG("穿透包线", charts.HitRateLabels, []SVGSeries{
			{Name: "AP pen", Color: svgAPColor, Values: penAP},
			{Name: "HE pen", Color: svgHEColor, Values: penHE},
			{Name: "belt armor", Color: svgArmorColor, Values: penArmor},
		}, 720, 300, yMax)
		views = append(
			views,
			ChartView{
				ID:          "penetration",
				Title:       "穿透包线",
				Description: "AP/HE 穿深（mm）vs 装甲；红色为装甲带。",
				File:        "penetration_envelope.svg",
				SVG:         template.HTML(svg),
			},
		)
		options["penetration"] = map[string]any{
			"animation": false,
			"tooltip":   map[string]any{"trigger": "axis"},
			"legend": map[string]any{
				"data": []string{"AP pen", "HE pen", "belt armor"},
				"top":  0,
				"left": "center",
			},
			"grid":  map[string]any{"left": 48, "right": 18, "top": 48, "bottom": 30},
			"xAxis": map[string]any{"type": "category", "data": charts.HitRateLabels},
			"yAxis": map[string]any{"type": "value", "min": 0, "max": yMax},
			"series": []any{
				map[string]any{
					"name":      "AP pen",
					"type":      "line",
					"smooth":    true,
					"data":      penAP,
					"itemStyle": map[string]any{"color": svgAPColor},
				},
				map[string]any{
					"name":      "HE pen",
					"type":      "line",
					"smooth":    true,
					"data":      penHE,
					"itemStyle": map[string]any{"color": svgHEColor},
				},
				map[string]any{
					"name":      "belt armor",
					"type":      "line",
					"data":      penArmor,
					"lineStyle": map[string]any{"type": "dashed"},
					"itemStyle": map[string]any{"color": svgArmorColor},
				},
			},
		}
	}
	return views, options
}

func maxFloat(values []float64) float64 {
	m := 0.0
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

func marshalChartOptions(options map[string]any) (string, error) {
	data, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("marshal chart options: %w", err)
	}
	return strings.ReplaceAll(string(data), "</", `<\/`), nil
}

func writeTemplate(path string, tmpl *template.Template, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer file.Close()
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("render %s: %w", path, err)
	}
	return nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	return writeFile(path, data)
}

func collectFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() == "checksums.sha256" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func writeChecksums(root string) error {
	files, err := collectFiles(root)
	if err != nil {
		return err
	}
	var b strings.Builder
	for _, rel := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("read %s: %w", full, err)
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&b, "%s  %s\n", hex.EncodeToString(sum[:]), rel)
	}
	return writeFile(filepath.Join(root, "checksums.sha256"), []byte(b.String()))
}

func historyFromResult(paths Paths, runID string, result *Result, git GitInfo, configHash string) HistoryEntry {
	scorecard := ScorecardCounts{}
	for _, item := range result.Scorecard {
		switch item.Status {
		case "green":
			scorecard.Green++
		case "yellow":
			scorecard.Yellow++
		case "red":
			scorecard.Red++
		}
	}
	findings := FindingCounts{}
	for _, finding := range result.Findings {
		switch finding.Status {
		case "new":
			findings.New++
		case "persisting":
			findings.Persisting++
		case "fixed":
			findings.Fixed++
		case "noise":
			findings.Noise++
		}
	}
	return HistoryEntry{
		RunID:        runID,
		Date:         result.StartedAt.Format("2006-01-02"),
		StartedAt:    result.StartedAt.Format("2006-01-02 15:04:05"),
		Suite:        string(result.Scenario.Mode),
		Mock:         result.Mock,
		Fidelity:     result.Fidelity,
		Commit:       git.CommitShort,
		Branch:       git.Branch,
		Dirty:        git.Dirty,
		WorktreeHash: git.WorktreeHash,
		Config:       configHash,
		Scorecard:    scorecard,
		Findings:     findings,
		RuntimeMS:    result.DurationMS,
		Report:       filepath.ToSlash(filepath.Join("runs", runID, "report.html")),
	}
}

func appendHistory(reportDir string, entry HistoryEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(reportDir, "history.jsonl")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open history: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	return nil
}

func writeIndex(reportDir string) error {
	historyPath := filepath.Join(reportDir, "history.jsonl")
	data, err := os.ReadFile(historyPath)
	if err != nil && !os.IsNotExist(err) {
		return err
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
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].RunID > entries[j].RunID })
	if len(entries) > 200 {
		entries = entries[:200]
	}
	return writeTemplate(filepath.Join(reportDir, "index.html"), indexTemplate, entries)
}

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="zh-Hans"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Jutland simulation runs</title>
<style>
body{margin:0;background:#f5f7fa;color:#172033;font:14px/1.6 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif}
main{max-width:1100px;margin:0 auto;padding:28px 22px 56px}
h1{font-size:26px;margin:0 0 6px}.muted{color:#5b6675}.card{background:#fff;border:1px solid #dce3ec;border-radius:12px;padding:18px 20px;box-shadow:0 1px 2px rgba(16,24,40,.06)}
table{width:100%;border-collapse:collapse;font-size:13px}th,td{text-align:left;padding:8px 10px;border-bottom:1px solid #e6ebf2}th{color:#5b6675;background:#fafbfc}
tr:last-child td{border-bottom:0}a{color:#2d6cdf;text-decoration:none}.tag{display:inline-block;font-size:11px;padding:0 7px;border-radius:999px;border:1px solid #dce3ec;color:#5b6675;margin-right:6px}.mock{background:#fff7e6;border-color:#f2d39a;color:#7a4b00}.small{font-size:12px}
</style></head><body><main>
<h1>Jutland simulation runs</h1><p class="muted">本地报告索引；最近 200 次。打开 <code>runs/&lt;id&gt;/report.html</code> 查看详情；<a href="trend.html">查看趋势</a> · <a href="ai/analysis-brief.md">分析简报</a>。</p>
<div class="card"><table><thead><tr><th>时间</th><th>场景</th><th>提交</th><th>记分卡</th><th>Findings</th><th>耗时</th></tr></thead><tbody>
{{range .}}<tr>
<td><a href="{{.Report}}">{{.StartedAt}}</a><div class="muted small"><code>{{.RunID}}</code></div></td>
<td>{{.Suite}}{{if .Mock}} <span class="tag mock">MOCK</span>{{end}}{{if .Dirty}} <span class="tag">dirty</span>{{end}}</td>
<td><code>{{.Commit}}</code>{{if .Branch}} {{.Branch}}{{end}}</td>
<td>🟢 {{.Scorecard.Green}} · 🟡 {{.Scorecard.Yellow}} · 🔴 {{.Scorecard.Red}}</td>
<td>new {{.Findings.New}} · persist {{.Findings.Persisting}} · fixed {{.Findings.Fixed}}</td>
<td>{{.RuntimeMS}} ms</td>
</tr>{{else}}<tr><td colspan="6" class="muted">还没有 run；先执行 <code>jutland-sim demo</code>。</td></tr>{{end}}
</tbody></table></div>
</main></body></html>`))

func changeNote(result *Result) string {
	if result == nil {
		return "无结果"
	}
	if result.Mock {
		return "P1 合成数据"
	}
	switch result.Fidelity {
	case "F0":
		return "F0 纯伤害/穿深台"
	case "F1":
		return "F1 真实配置对射"
	case "F2":
		return "F2 合同作战"
	case "F3":
		return "F3 AI vs AI 完整任务"
	default:
		return "未知保真度"
	}
}
