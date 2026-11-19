package sim

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EvaluationRow is one run in the cross-fidelity evaluation.
type EvaluationRow struct {
	RunID       string             `json:"runId"`
	Scenario    string             `json:"scenario"`
	Fidelity    string             `json:"fidelity"`
	Mode        Mode               `json:"mode"`
	Seed        int64              `json:"seed"`
	Green       int                `json:"green"`
	Yellow      int                `json:"yellow"`
	Red         int                `json:"red"`
	Findings    int                `json:"findings"`
	Diagnostics int                `json:"diagnostics"`
	Metrics     map[string]float64 `json:"metrics"`
	ReportHTML  string             `json:"reportHtml"`
}

// EvaluationReport aggregates a suite into a human/AI overview.
type EvaluationReport struct {
	SchemaVersion int             `json:"schemaVersion"`
	Suite         string          `json:"suite"`
	GeneratedAt   time.Time       `json:"generatedAt"`
	Runs          int             `json:"runs"`
	Fidelity      map[string]int  `json:"fidelity"`
	Status        map[string]int  `json:"status"`
	Rows          []EvaluationRow `json:"rows"`
	Missing       []string        `json:"missing,omitempty"`
}

var evalTemplate = template.Must(template.New("eval").Parse(`<!doctype html>
<html lang="zh-Hans"><head><meta charset="utf-8"><title>Jutland evaluation: {{.Report.Suite}}</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:2rem;color:#17202a;background:#f7f9fb}
h1{margin-bottom:.2rem} .small{color:#5f6b7a;font-size:.9rem}
table{border-collapse:collapse;width:100%;background:#fff;margin:1rem 0}
th,td{border:1px solid #dbe2ea;padding:.45rem .55rem;text-align:left;font-size:.9rem}
th{background:#eef3f8} tr:nth-child(even){background:#fafcfe}
.green{color:#1e7b45;font-weight:600}.yellow{color:#9a6b00;font-weight:600}.red{color:#b42318;font-weight:600}
a{color:#155eef;text-decoration:none}
</style></head><body>
<h1>Jutland 仿真评估</h1>
<p class="small">suite <code>{{.Report.Suite}}</code> · {{.Report.Runs}} runs · generated {{.GeneratedAt}}</p>
<section><h2>总览</h2><table><tr><th>绿</th><th>黄</th><th>红</th><th>Findings</th><th>Diagnostics</th></tr>
<tr><td class="green">{{index .Report.Status "green"}}</td><td class="yellow">{{index .Report.Status "yellow"}}</td><td class="red">{{index .Report.Status "red"}}</td><td>{{.Findings}}</td><td>{{.Diagnostics}}</td></tr></table>
<p class="small">Fidelity: {{range $k,$v := .Report.Fidelity}}{{$k}}={{$v}} {{end}}</p></section>
<section><h2>Runs</h2><table><thead><tr><th>run</th><th>scenario</th><th>fidelity</th><th>mode</th><th>green</th><th>yellow</th><th>red</th><th>findings</th><th>diag</th><th>shots</th><th>hit%</th><th>success</th><th>enemy sunk</th><th>p99ms</th></tr></thead><tbody>
{{range .Report.Rows}}<tr><td><a href="../runs/{{.RunID}}/report.html">{{.RunID}}</a></td><td>{{.Scenario}}</td><td>{{.Fidelity}}</td><td>{{.Mode}}</td><td class="green">{{.Green}}</td><td class="yellow">{{.Yellow}}</td><td class="red">{{.Red}}</td><td>{{.Findings}}</td><td>{{.Diagnostics}}</td><td>{{index .Metrics "shots"}}</td><td>{{index .Metrics "hit_rate"}}</td><td>{{index .Metrics "kill_rate"}}</td><td>{{index .Metrics "tick_p99_ms"}}</td></tr>{{end}}
</tbody></table></section>
{{if .Report.Missing}}<section><h2>Missing</h2><ul>{{range .Report.Missing}}<li>{{.}}</li>{{end}}</ul></section>{{end}}
</body></html>`))

// WriteEvaluation reads a suite summary and writes an aggregate evaluation.
func WriteEvaluation(paths Paths, summaryPath string) (string, error) {
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return "", err
	}
	var suite SuiteSummary
	if err := json.Unmarshal(data, &suite); err != nil {
		return "", fmt.Errorf("parse suite summary: %w", err)
	}
	report := EvaluationReport{
		SchemaVersion: 1, Suite: suite.Name, GeneratedAt: time.Now(), Runs: len(suite.Runs),
		Fidelity: map[string]int{}, Status: map[string]int{},
	}
	findings, diagnostics := 0, 0
	for _, run := range suite.Runs {
		result, _, err := loadComparisonRun(paths, run.RunID)
		if err != nil {
			report.Missing = append(report.Missing, run.RunID)
			continue
		}
		row := EvaluationRow{
			RunID: run.RunID, Scenario: result.Scenario.Name, Fidelity: result.Fidelity,
			Mode: result.Scenario.Mode, Seed: result.Seed, Findings: len(result.Findings),
			Diagnostics: len(result.Diagnostics), ReportHTML: run.ReportHTML,
			Metrics: map[string]float64{},
		}
		for _, item := range result.Scorecard {
			switch item.Status {
			case "green":
				row.Green++
			case "yellow":
				row.Yellow++
			case "red":
				row.Red++
			}
			report.Status[item.Status]++
		}
		for _, name := range []string{"shots", "hit_rate", "mission_success", "enemy_sunk", "own_sunk", "tick_p99_ms"} {
			if m, ok := metricMap(result.Metrics)[name]; ok {
				row.Metrics[name] = m.Value
			}
		}
		report.Fidelity[result.Fidelity]++
		findings += len(result.Findings)
		diagnostics += len(result.Diagnostics)
		report.Rows = append(report.Rows, row)
	}
	dir := filepath.Dir(summaryPath)
	base := strings.TrimSuffix(filepath.Base(summaryPath), filepath.Ext(summaryPath))
	mdPath := filepath.Join(dir, base+"-evaluation.md")
	htmlPath := filepath.Join(dir, base+"-evaluation.html")
	jsonPath := filepath.Join(dir, base+"-evaluation.json")
	if err := writeEvaluationMarkdown(mdPath, report, findings, diagnostics); err != nil {
		return "", err
	}
	var html strings.Builder
	if err := evalTemplate.Execute(&html, map[string]any{
		"Report": report, "GeneratedAt": report.GeneratedAt.Format("2006-01-02 15:04:05"),
		"Findings": findings, "Diagnostics": diagnostics,
	}); err != nil {
		return "", err
	}
	if err := os.WriteFile(htmlPath, []byte(html.String()), 0o644); err != nil {
		return "", err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(jsonPath, append(payload, '\n'), 0o644); err != nil {
		return "", err
	}
	return htmlPath, nil
}

func writeEvaluationMarkdown(path string, report EvaluationReport, findings, diagnostics int) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Jutland 仿真评估：%s\n\n", report.Suite)
	fmt.Fprintf(&b, "- runs: %d\n- green: %d\n- yellow: %d\n- red: %d\n- findings: %d\n- diagnostics: %d\n\n",
		report.Runs, report.Status["green"], report.Status["yellow"], report.Status["red"], findings, diagnostics)
	b.WriteString(
		"| run | scenario | fidelity | mode | green | yellow | red | findings | diag | shots | hit% | success | enemy sunk | p99ms |\n",
	)
	b.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, row := range report.Rows {
		fmt.Fprintf(
			&b,
			"| [%s](../runs/%s/report.html) | %s | %s | %s | %d | %d | %d | %d | %d | %v | %v | %v | %v | %v |\n",
			row.RunID,
			row.RunID,
			row.Scenario,
			row.Fidelity,
			row.Mode,
			row.Green,
			row.Yellow,
			row.Red,
			row.Findings,
			row.Diagnostics,
			row.Metrics["shots"],
			row.Metrics["hit_rate"],
			row.Metrics["mission_success"],
			row.Metrics["enemy_sunk"],
			row.Metrics["tick_p99_ms"],
		)
	}
	if len(report.Missing) > 0 {
		fmt.Fprintf(&b, "\n## Missing\n\n- %s\n", strings.Join(report.Missing, "\n- "))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
