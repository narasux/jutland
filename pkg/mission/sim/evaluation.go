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
func (r EvaluationRow) Metric(name string) string {
	if value, ok := r.Metrics[name]; ok {
		return fmt.Sprintf("%v", value)
	}
	return "—"
}

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
	Smoke       bool               `json:"smoke"`
	Metrics     map[string]float64 `json:"metrics"`
	// FindingsByKind 按 bug/data/balance 拆分，避免数据类 finding 淹没真正的 bug。
	FindingsByKind map[string]int `json:"findingsByKind,omitempty"`
	RedItems       []string       `json:"redItems,omitempty"`
	FindingIDs     []string       `json:"findingIds,omitempty"`
	ReportHTML     string         `json:"reportHtml"`
}

// PenetrationSummary is one penetration-scenario row in the suite evaluation.
type PenetrationSummary struct {
	Scenario           string  `json:"scenario"`
	RunID              string  `json:"runId"`
	PenetrationMM      float64 `json:"penetrationMm"`
	ReferenceArmor     float64 `json:"referenceArmor"`
	ReferenceRatio     float64 `json:"referenceRatio"`
	MaxPenetratedArmor float64 `json:"maxPenetratedArmor"`
	APPenetrated       int     `json:"apPenetrated"`
}

// EvaluationReport aggregates a suite into a human/AI overview.
type EvaluationReport struct {
	SchemaVersion int            `json:"schemaVersion"`
	Suite         string         `json:"suite"`
	GeneratedAt   time.Time      `json:"generatedAt"`
	Runs          int            `json:"runs"`
	Fidelity      map[string]int `json:"fidelity"`
	Status        map[string]int `json:"status"`
	SmokeRed      int            `json:"smokeRed"`
	BalanceRed    int            `json:"balanceRed"`
	// BugFindings 只统计 Kind=bug 的 finding；门禁用它判断是否真的坏了。
	BugFindings     int                  `json:"bugFindings"`
	DataFindings    int                  `json:"dataFindings"`
	BalanceFindings int                  `json:"balanceFindings"`
	Rows            []EvaluationRow      `json:"rows"`
	Penetration     []PenetrationSummary `json:"penetration,omitempty"`
	Missing         []string             `json:"missing,omitempty"`
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
<section><h2>总览</h2><table><tr><th>绿</th><th>黄</th><th>红</th><th>红(smoke/balance)</th><th>Findings</th><th>Diagnostics</th></tr>
<tr><td class="green">{{index .Report.Status "green"}}</td><td class="yellow">{{index .Report.Status "yellow"}}</td><td class="red">{{index .Report.Status "red"}}</td><td>{{.Report.SmokeRed}} / {{.Report.BalanceRed}}</td><td>{{.Findings}} <span class="small">(bug {{.Report.BugFindings}} / data {{.Report.DataFindings}} / balance {{.Report.BalanceFindings}})</span></td><td>{{.Diagnostics}}</td></tr></table>
<p class="small">Fidelity: {{range $k,$v := .Report.Fidelity}}{{$k}}={{$v}} {{end}}</p></section>
<section><h2>Runs</h2><table><thead><tr><th>run</th><th>scenario</th><th>fidelity</th><th>mode</th><th>green</th><th>yellow</th><th>red</th><th>findings</th><th>diag</th><th>shots</th><th>hit%</th><th>success</th><th>enemy sunk</th><th>p99ms</th></tr></thead><tbody>
{{range .Report.Rows}}<tr><td><a href="../runs/{{.RunID}}/report.html">{{.RunID}}</a></td><td>{{.Scenario}}</td><td>{{.Fidelity}}</td><td>{{.Mode}}</td><td class="green">{{.Green}}</td><td class="yellow">{{.Yellow}}</td><td class="red">{{.Red}}</td><td>{{.Findings}}</td><td>{{.Diagnostics}}</td><td>{{index .Metrics "shots"}}</td><td>{{index .Metrics "hit_rate"}}</td><td>{{index .Metrics "kill_rate"}}</td><td>{{index .Metrics "tick_p99_ms"}}</td></tr>{{end}}
</tbody></table></section>
{{if .Report.Penetration}}<section><h2>穿甲矩阵</h2><table><thead><tr><th>scenario</th><th>穿深 mm</th><th>参考装甲 mm</th><th>参考比值</th><th>最大可穿装甲 mm</th><th>AP 穿透点</th></tr></thead><tbody>
{{range .Report.Penetration}}<tr><td><a href="../runs/{{.RunID}}/report.html">{{.Scenario}}</a></td><td>{{printf "%.1f" .PenetrationMM}}</td><td>{{printf "%.1f" .ReferenceArmor}}</td><td>{{printf "%.2f" .ReferenceRatio}}</td><td>{{printf "%.1f" .MaxPenetratedArmor}}</td><td>{{.APPenetrated}}</td></tr>{{end}}
</tbody></table></section>{{end}}
<section><h2>红项与 Finding</h2><ul>
{{range .Report.Rows}}{{if .RedItems}}<li><a href="../runs/{{.RunID}}/report.html">{{.RunID}}</a><ul>{{range .RedItems}}<li class="red">{{.}}</li>{{end}}</ul></li>{{end}}{{end}}
{{range .Report.Rows}}{{range .FindingIDs}}<li>{{.}}</li>{{end}}{{end}}
</ul></section>
{{if .Report.Missing}}<section><h2>Missing</h2><ul>{{range .Report.Missing}}<li>{{.}}</li>{{end}}</ul></section>{{end}}
</body></html>`))

// WriteEvaluation reads a suite summary and writes an aggregate evaluation.
func WriteEvaluation(paths Paths, summaryPath string) (EvaluationReport, string, error) {
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return EvaluationReport{}, "", err
	}
	var suite SuiteSummary
	if err := json.Unmarshal(data, &suite); err != nil {
		return EvaluationReport{}, "", fmt.Errorf("parse suite summary: %w", err)
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
		smoke := result.Fidelity == "F2" ||
			strings.Contains(strings.ToLower(strings.Join(result.DataQuality, " ")), "smoke")
		row := EvaluationRow{
			RunID: run.RunID, Scenario: result.Scenario.Name, Fidelity: result.Fidelity,
			Mode: result.Scenario.Mode, Seed: result.Seed, Findings: len(result.Findings),
			Diagnostics: len(result.Diagnostics), Smoke: smoke, ReportHTML: run.ReportHTML,
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
				label := "balance"
				if row.Smoke {
					report.SmokeRed++
					label = "smoke"
				} else {
					report.BalanceRed++
				}
				row.RedItems = append(
					row.RedItems,
					fmt.Sprintf(
						"[%s] %s/%s=%s (target %s)",
						label,
						item.Dimension,
						item.Metric,
						item.Value,
						item.Target,
					),
				)
			}
			report.Status[item.Status]++
		}
		metricIndex := metricMap(result.Metrics)
		for _, name := range []string{
			"shots", "hit_rate", "mission_success", "enemy_sunk", "own_sunk", "tick_p99_ms",
			"penetration_mm", "reference_armor", "reference_ratio", "max_penetrated_armor", "ap_penetrated",
		} {
			if m, ok := metricIndex[name]; ok {
				row.Metrics[name] = m.Value
			}
		}
		// 穿甲场景（F0 纯几何）单独汇总成矩阵，避免在 shots/hit% 列全是横杠。
		if result.Scenario.Mode == ModePenetration {
			report.Penetration = append(report.Penetration, PenetrationSummary{
				Scenario:           result.Scenario.Name,
				RunID:              run.RunID,
				PenetrationMM:      metricIndex["penetration_mm"].Value,
				ReferenceArmor:     metricIndex["reference_armor"].Value,
				ReferenceRatio:     metricIndex["reference_ratio"].Value,
				MaxPenetratedArmor: metricIndex["max_penetrated_armor"].Value,
				APPenetrated:       int(metricIndex["ap_penetrated"].Value),
			})
		}
		for _, finding := range result.Findings {
			row.FindingIDs = append(row.FindingIDs, finding.ID+": "+finding.Title)
			if row.FindingsByKind == nil {
				row.FindingsByKind = map[string]int{}
			}
			row.FindingsByKind[finding.Kind]++
			switch finding.Kind {
			case "bug":
				report.BugFindings++
			case "balance":
				report.BalanceFindings++
			default:
				report.DataFindings++
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
		return report, "", err
	}
	var html strings.Builder
	if err := evalTemplate.Execute(&html, map[string]any{
		"Report": report, "GeneratedAt": report.GeneratedAt.Format("2006-01-02 15:04:05"),
		"Findings": findings, "Diagnostics": diagnostics,
	}); err != nil {
		return report, "", err
	}
	if err := os.WriteFile(htmlPath, []byte(html.String()), 0o644); err != nil {
		return report, "", err
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, "", err
	}
	if err := os.WriteFile(jsonPath, append(payload, '\n'), 0o644); err != nil {
		return report, "", err
	}
	return report, htmlPath, nil
}

// EvaluationGateError turns an evaluation into a CI gate result.
// 缺 run 或 smoke 红项说明引擎/场景真的坏了，直接失败；balance 红项是待评审的平衡信号，
// 只保留在报告里，不阻断 CI。这样夜间/提交前门禁不会被“需要重新平衡”的提示卡住。
func EvaluationGateError(report EvaluationReport) error {
	if len(report.Missing) > 0 {
		return fmt.Errorf(
			"evaluation missing %d run(s): %s",
			len(report.Missing), strings.Join(report.Missing, ", "),
		)
	}
	if report.SmokeRed > 0 {
		return fmt.Errorf("evaluation has %d smoke red item(s)", report.SmokeRed)
	}
	if report.BugFindings > 0 {
		return fmt.Errorf("evaluation has %d bug finding(s)", report.BugFindings)
	}
	return nil
}

func writeEvaluationMarkdown(path string, report EvaluationReport, findings, diagnostics int) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Jutland 仿真评估：%s\n\n", report.Suite)
	fmt.Fprintf(
		&b,
		"- runs: %d\n- green: %d\n- yellow: %d\n- red: %d (smoke %d / balance %d)\n"+
			"- findings: %d (bug %d / data %d / balance %d)\n- diagnostics: %d\n\n",
		report.Runs,
		report.Status["green"],
		report.Status["yellow"],
		report.Status["red"],
		report.SmokeRed,
		report.BalanceRed,
		findings,
		report.BugFindings,
		report.DataFindings,
		report.BalanceFindings,
		diagnostics,
	)
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
			row.Metric("shots"),
			row.Metric("hit_rate"),
			row.Metric("mission_success"),
			row.Metric("enemy_sunk"),
			row.Metric("tick_p99_ms"),
		)
	}
	if len(report.Penetration) > 0 {
		b.WriteString("\n## 穿甲矩阵\n\n")
		b.WriteString(
			"| scenario | 穿深 mm | 参考装甲 mm | 参考比值 | 最大可穿装甲 mm | AP 穿透点 |\n" +
				"|---|---:|---:|---:|---:|---:|\n",
		)
		for _, pen := range report.Penetration {
			fmt.Fprintf(
				&b,
				"| %s | %.1f | %.1f | %.2f | %.1f | %d |\n",
				pen.Scenario,
				pen.PenetrationMM,
				pen.ReferenceArmor,
				pen.ReferenceRatio,
				pen.MaxPenetratedArmor,
				pen.APPenetrated,
			)
		}
	}
	b.WriteString("\n## 红项与 Finding\n\n")
	for _, row := range report.Rows {
		if len(row.RedItems) == 0 && len(row.FindingIDs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", row.RunID)
		for _, item := range row.RedItems {
			fmt.Fprintf(&b, "- RED %s\n", item)
		}
		for _, item := range row.FindingIDs {
			fmt.Fprintf(&b, "- finding %s\n", item)
		}
	}
	if len(report.Missing) > 0 {
		fmt.Fprintf(&b, "\n## Missing\n\n- %s\n", strings.Join(report.Missing, "\n- "))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
