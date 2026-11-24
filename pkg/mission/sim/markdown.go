package sim

import (
	"fmt"
	"strings"
)

func resultFidelity(result *Result) string {
	if result.Fidelity != "" {
		return result.Fidelity
	}
	if result.Mock {
		return "mock"
	}
	return "unknown"
}

func buildSummary(result *Result, manifest Manifest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", result.Scenario.Name)
	if result.Mock {
		b.WriteString("> **P1 MOCK**：合成数据，只用于确认版式/字段，不代表真实战斗结果。\n\n")
	}
	fmt.Fprintf(&b, "- 时间：%s\n", manifest.StartedAt)
	fmt.Fprintf(&b, "- 耗时：%d ms · Seed：%d · Runs：%d\n", manifest.DurationMS, result.Seed, result.Runs)
	if manifest.Git.Available {
		fmt.Fprintf(&b, "- Commit：`%s`", manifest.Git.CommitShort)
		if manifest.Git.Branch != "" {
			fmt.Fprintf(&b, " · %s", manifest.Git.Branch)
		}
		if manifest.Git.Subject != "" {
			fmt.Fprintf(&b, " · %s", manifest.Git.Subject)
		}
		b.WriteString("\n")
		if manifest.Git.Dirty {
			fmt.Fprintf(
				&b,
				"- *工作区有未提交改动（%d 个文件，worktree `%s`）：结果可能无法从 commit 精确复现。*\n",
				manifest.Git.DirtyCount,
				manifest.Git.WorktreeHash,
			)
		} else {
			b.WriteString("- 工作区：clean\n")
		}
	} else {
		b.WriteString("- Commit：unknown（git 不可用）\n")
	}
	fmt.Fprintf(&b, "- 场景：`%s` · fidelity：%s\n", manifest.ScenarioPath, resultFidelity(result))
	if shot := result.Scenario.Shot; shot != nil {
		fmt.Fprintf(
			&b,
			"- 弹药：%s · 目标：%s (%s) · 基础伤害 %.1f\n",
			shot.AmmoName,
			shot.TargetName,
			shot.TargetType,
			shot.BaseDamage,
		)
	}
	fmt.Fprintf(&b, "- 完整报告：[report.html](report.html) · AI 视图：[ai/analysis-context.md](ai/analysis-context.md)\n\n")

	b.WriteString("## 记分卡\n\n| 维度 | 指标 | 结果 | 目标带 | 状态 |\n|---|---|---|---|---|\n")
	for _, item := range result.Scorecard {
		status := map[string]string{"green": "🟢", "yellow": "🟡", "red": "🔴"}[item.Status]
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", item.Dimension, item.Metric, item.Value, item.Target, status)
	}
	if result.BattleReport != nil {
		report := result.BattleReport
		b.WriteString("\n## 战报\n\n")
		if report.Note != "" {
			fmt.Fprintf(&b, "> %s\n\n", report.Note)
		}
		fmt.Fprintf(
			&b,
			"- 结果：%s · 时长 %d ticks · 场景侧 %s\n",
			report.Status, report.Ticks, report.Side,
		)
		fmt.Fprintf(
			&b,
			"- 我方：舰艇 %s；飞机 %s；损失吨位 %s；造成伤害 %s\n",
			report.Own.ShipsLine, report.Own.PlanesLine,
			report.Own.SunkTonnage, report.Own.DamageDealt,
		)
		fmt.Fprintf(
			&b,
			"- 敌方：舰艇 %s；飞机 %s；损失吨位 %s；造成伤害 %s\n",
			report.Enemy.ShipsLine, report.Enemy.PlanesLine,
			report.Enemy.SunkTonnage, report.Enemy.DamageDealt,
		)
		fmt.Fprintf(
			&b,
			"- 随舰损失飞机：我方 %s 架 / 敌方 %s 架\n",
			report.Own.PlanesWithShip, report.Enemy.PlanesWithShip,
		)
		fmt.Fprintf(
			&b,
			"- 交换比：%s · 我方保存率：%s · 资金：%s\n",
			report.Exchange, report.Survival, report.Own.Funds,
		)
		fmt.Fprintf(
			&b,
			"- 关键战果：%d 条；完整战况：%d 条（[battle-log.jsonl](battle-log.jsonl)）\n",
			report.KillsTotal, report.EventsTotal,
		)
		if len(report.Kills) > 0 {
			b.WriteString("\n### 关键战果（最新 10 条）\n\n")
			for idx, event := range report.Kills {
				if idx >= 10 {
					break
				}
				fmt.Fprintf(&b, "- `%s` %s %s\n", event.Time, event.Side, event.Summary)
			}
		}
	}
	b.WriteString("\n## 关键指标\n\n| 分组 | 指标 | 值 | CI |\n|---|---|---|---|\n")
	for _, row := range result.Metrics {
		fmt.Fprintf(&b, "| %s | %s | %v %s | %s |\n", row.Group, row.Name, row.Value, row.Unit, row.CI)
	}
	if result.PreviousDiff != nil {
		b.WriteString("\n")
		b.WriteString(previousDiffMarkdown(result.PreviousDiff))
	}
	if len(result.Diagnostics) > 0 {
		b.WriteString("\n## 诊断\n\n")
		b.WriteString(diagnosticsMarkdown(result.Diagnostics))
	}
	b.WriteString("\n## Findings\n\n")
	for _, finding := range result.Findings {
		fmt.Fprintf(
			&b,
			"- **[%s/%s/%s] %s** — %s\n",
			finding.Kind,
			finding.Severity,
			finding.Status,
			finding.Title,
			finding.Summary,
		)
		if finding.Repro != "" {
			fmt.Fprintf(&b, "  - 复现：`%s`\n", finding.Repro)
		}
	}
	battleTrend := result.BattleReport != nil && len(result.BattleReport.Trend) > 0
	if len(result.Charts.HitRate) > 0 || len(result.Charts.Survival) > 0 ||
		len(result.Charts.Exchange) > 0 || len(result.Charts.Penetration) > 0 || battleTrend {
		b.WriteString("\n## 图表\n\n")
		if len(result.Charts.HitRate) > 0 {
			b.WriteString("![命中/减伤](charts/hit_rate_vs_range.svg)\n\n")
		}
		if len(result.Charts.Survival) > 0 {
			if result.Fidelity == "F1" || result.Fidelity == "F2" || result.Fidelity == "F3" {
				b.WriteString("![舰队生存率](charts/survival_curve.svg)\n\n")
			} else {
				b.WriteString("![击沉概率](charts/survival_curve.svg)\n\n")
			}
		}
		if battleTrend {
			b.WriteString("![在役舰艇](charts/battle_ships.svg)\n\n")
			b.WriteString("![在役飞机](charts/battle_planes.svg)\n\n")
			b.WriteString("![累计损失](charts/battle_losses.svg)\n\n")
		}
		if len(result.Charts.Exchange) > 0 {
			b.WriteString("![交换矩阵](charts/exchange_matrix.svg)\n\n")
		}
		if len(result.Charts.Penetration) > 0 {
			b.WriteString("![穿透包线](charts/penetration_envelope.svg)\n\n")
		}
	}
	b.WriteString("## 数据质量\n\n")
	for _, warning := range result.DataQuality {
		fmt.Fprintf(&b, "- %s\n", warning)
	}
	b.WriteString("\n## 复现\n\n```bash\n")
	b.WriteString(manifest.Command)
	b.WriteString("\n```\n")
	return b.String()
}

func buildAIContext(result *Result, manifest Manifest) string {
	var b strings.Builder
	b.WriteString("# Simulation analysis context\n\n")
	if result.Mock {
		b.WriteString("> MOCK: synthetic P1 data. Do not treat these numbers as real balance evidence.\n\n")
	}
	b.WriteString("## 1. Run\n\n")
	fmt.Fprintf(&b, "- scenario: %s (%s)\n", result.Scenario.Name, result.Scenario.Mode)
	fmt.Fprintf(&b, "- started_at: %s\n", manifest.StartedAt)
	fmt.Fprintf(&b, "- duration_ms: %d\n", manifest.DurationMS)
	fmt.Fprintf(&b, "- seed: %d · runs: %d · max_ticks: %d\n", result.Seed, result.Runs, result.Scenario.MaxTicks)
	if shot := result.Scenario.Shot; shot != nil {
		fmt.Fprintf(
			&b,
			"- shot: ammo=%s target=%s hp=%.0f base_damage=%.1f count=%d\n",
			shot.AmmoName,
			shot.TargetName,
			shot.TargetHP,
			shot.BaseDamage,
			shot.Count,
		)
	}
	fmt.Fprintf(
		&b,
		"- commit: %s · branch: %s · dirty: %t\n",
		manifest.Git.CommitShort,
		manifest.Git.Branch,
		manifest.Git.Dirty,
	)
	if manifest.Git.Dirty {
		fmt.Fprintf(&b, "- worktree_hash: %s · dirty_files: %d\n", manifest.Git.WorktreeHash, manifest.Git.DirtyCount)
		for _, file := range manifest.Git.DirtyFiles {
			fmt.Fprintf(&b, "  - %s %s\n", file.Status, file.Path)
		}
		b.WriteString("- warning: uncommitted changes; results may not map to a commit.\n")
	}
	fmt.Fprintf(&b, "- config_hash: %s · scenario_path: %s\n", manifest.ConfigHash, manifest.ScenarioPath)
	fmt.Fprintf(&b, "- sim_home: %s · lab_commit: %s\n", manifest.SimHome, manifest.LabCommit)
	fmt.Fprintf(&b, "- fidelity: %s · mock=%t\n\n", resultFidelity(result), result.Mock)

	b.WriteString("## 2. Scorecard\n\n| dimension | metric | value | target | status |\n|---|---|---|---|---|\n")
	for _, item := range result.Scorecard {
		fmt.Fprintf(
			&b,
			"| %s | %s | %s | %s | %s |\n",
			item.Dimension,
			item.Metric,
			item.Value,
			item.Target,
			item.Status,
		)
	}
	if result.PreviousDiff != nil {
		b.WriteString("\n## 2.1 Previous diff\n\n")
		fmt.Fprintf(&b, "- previous: `%s`\n", result.PreviousDiff.ARun)
		b.WriteString("| metric | prev | this | delta | delta% |\n|---|---:|---:|---:|---:|\n")
		for _, m := range result.PreviousDiff.Metrics {
			fmt.Fprintf(&b, "| %s | %v | %v | %v | %v%% |\n", m.Name, m.A, m.B, m.Delta, m.DeltaPct)
		}
		for _, f := range result.PreviousDiff.FindingsAdded {
			fmt.Fprintf(&b, "- finding added: `%s`\n", f.ID)
		}
		for _, f := range result.PreviousDiff.FindingsGone {
			fmt.Fprintf(&b, "- finding gone: `%s`\n", f.ID)
		}
	}
	b.WriteString("\n## 3. Findings\n\n")
	for _, finding := range result.Findings {
		fmt.Fprintf(&b, "### %s [%s/%s] %s\n\n", finding.ID, finding.Kind, finding.Severity, finding.Title)
		fmt.Fprintf(&b, "%s\n\n", finding.Summary)
		for _, evidence := range finding.Evidence {
			fmt.Fprintf(&b, "- `%s` = %v", evidence.Metric, evidence.Value)
			if evidence.Baseline != 0 {
				fmt.Fprintf(&b, " (baseline %v, delta %v)", evidence.Baseline, evidence.Delta)
			}
			if evidence.CI != "" {
				fmt.Fprintf(&b, ", CI %s", evidence.CI)
			}
			if evidence.File != "" {
				fmt.Fprintf(&b, " — %s", evidence.File)
			}
			b.WriteString("\n")
		}
		if finding.SuggestedExperiment != "" {
			fmt.Fprintf(&b, "- suggested_experiment: %s\n", finding.SuggestedExperiment)
		}
		if finding.Repro != "" {
			fmt.Fprintf(&b, "- repro: `%s`\n", finding.Repro)
		}
		b.WriteString("\n")
	}

	b.WriteString("## 4. Metrics\n\n| group | metric | value | unit | CI |\n|---|---|---|---|---|\n")
	for _, row := range result.Metrics {
		fmt.Fprintf(&b, "| %s | %s | %v | %s | %s |\n", row.Group, row.Name, row.Value, row.Unit, row.CI)
	}
	b.WriteString("\n## 5. Data quality\n\n")
	for _, warning := range result.DataQuality {
		fmt.Fprintf(&b, "- %s\n", warning)
	}
	b.WriteString("\n## 6. Files\n\n")
	b.WriteString("- human report: ../../report.html\n")
	b.WriteString("- summary: ../../summary.md\n")
	b.WriteString("- findings: findings.json\n")
	b.WriteString("- metrics: metrics.json\n")
	b.WriteString("- diagnostics: diagnostics.json\n")
	b.WriteString("- changes: changes.json\n")
	b.WriteString("- manifest: ../../manifest.json\n")
	b.WriteString("- result: ../../result.json\n\n")
	b.WriteString("## 7. Data dictionary\n\n")
	b.WriteString("- `status`: green = inside target band, yellow = watch, red = needs explanation.\n")
	b.WriteString("- `exchange_ratio`: enemy value lost / own value lost; 1.0 is even.\n")
	b.WriteString("- `ttk_median`: median time-to-kill for runs that actually sank the target.\n")
	b.WriteString("- `censored`: share of runs that hit the tick cap without a sink.\n")
	b.WriteString("- `raw/applied/effective`: damage before/after armor, and capped at remaining HP.\n")
	b.WriteString("- `worktree_hash`: stable fingerprint of uncommitted changes; `clean` when clean.\n")
	fmt.Fprintf(&b, "- This run uses fidelity %s; interpret it at that level.\n\n", resultFidelity(result))
	b.WriteString("## 8. Repro\n\n```bash\n")
	b.WriteString(manifest.Command)
	b.WriteString("\n```\n")
	return b.String()
}

const llmPrompt = `你是 Jutland 的平衡/质量分析助手。只使用提供的 analysis-context.md、findings.json、metrics.json、diagnostics.json、changes.json。

任务：
1. 把每个 finding 分类为 bug / perf / balance / design / noise；
2. 指出证据是否充分；不充分标 inconclusive；
3. 给出最多 3 个下一步实验（场景/参数/seed），不要直接改数值；
4. 不要编造数字；引用 finding id / metric / file；
5. 输出 JSON：{findings:[{id,kind,severity,confidence,reason,experiment}], summary}

禁止：
- 仅凭图表外观下结论；
- 把噪声当回归；
- 建议大规模重做；
- 把 P1 MOCK 数据当成真实平衡结论。

如果 dirty=true，归因到 commit 时降低置信度，优先看 worktree_hash/config_hash。
`

func previousDiffMarkdown(diff *CompareReport) string {
	if diff == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 与上次对比\n\n- 基准：`%s`\n- 本次：`%s`\n\n", diff.ARun, diff.BRun)
	if len(diff.Metrics) > 0 {
		b.WriteString("| 分组 | 指标 | A | B | Delta | Delta% |\n|---|---|---:|---:|---:|---:|\n")
		for _, m := range diff.Metrics {
			fmt.Fprintf(&b, "| %s | %s | %v | %v | %v | %v%% |\n", m.Group, m.Name, m.A, m.B, m.Delta, m.DeltaPct)
		}
		b.WriteString("\n")
	}
	if len(diff.ScoreChanges) > 0 {
		fmt.Fprintf(&b, "记分卡变化：%s\n\n", strings.Join(diff.ScoreChanges, "; "))
	}
	if len(diff.FindingsAdded) > 0 {
		ids := make([]string, 0, len(diff.FindingsAdded))
		for _, f := range diff.FindingsAdded {
			ids = append(ids, f.ID)
		}
		fmt.Fprintf(&b, "新增 finding：%s\n\n", strings.Join(ids, ", "))
	}
	if len(diff.FindingsGone) > 0 {
		ids := make([]string, 0, len(diff.FindingsGone))
		for _, f := range diff.FindingsGone {
			ids = append(ids, f.ID)
		}
		fmt.Fprintf(&b, "消失 finding：%s\n\n", strings.Join(ids, ", "))
	}
	return b.String()
}
