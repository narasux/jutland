package sim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestDemoResultIsDeterministic 验证同一 seed 的合成数据完全一致，不同 seed 至少有一处变化。
// 场景：P1 mock 用于版式评审；若同 seed 结果抖动，报告 diff 和截图都不可复现。
// 预期：同 seed/同 startedAt 的 Result DeepEqual；seed+1 的命中率曲线不再相等。
// 覆盖：构造两个同参 scenario 调用 GenerateDemoResult，再用 reflect.DeepEqual 与 charts 对比。
func TestDemoResultIsDeterministic(t *testing.T) {
	sc := DefaultDemoScenario()
	at := time.Date(2026, 11, 19, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	a := GenerateDemoResult(&sc, at)
	b := GenerateDemoResult(&sc, at)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed produced different demo results")
	}
	sc2 := sc
	sc2.Seed++
	c := GenerateDemoResult(&sc2, at)
	if reflect.DeepEqual(a.Charts, c.Charts) {
		t.Fatal("different seed produced identical charts")
	}
}

// TestDemoResultShape 验证 mock 报告所需的 section 都有数据。
// 场景：P1 报告模板直接 range scorecard/findings/charts；空 section 会让版式评审失去意义。
// 预期：Mock=true；scorecard≥5、findings≥2、四种图表数据非空；finding 带证据和复现命令。
// 覆盖：DefaultDemoScenario + GenerateDemoResult，逐项断言切片长度和字段。
func TestDemoResultShape(t *testing.T) {
	sc := DefaultDemoScenario()
	r := GenerateDemoResult(&sc, time.Now())
	if !r.Mock {
		t.Fatal("Mock = false, want true")
	}
	if len(r.Scorecard) < 5 || len(r.Findings) < 2 {
		t.Fatalf("scorecard=%d findings=%d, want >=5/2", len(r.Scorecard), len(r.Findings))
	}
	if len(r.Charts.HitRate) == 0 || len(r.Charts.Survival) == 0 ||
		len(r.Charts.Exchange) == 0 || len(r.Charts.Penetration) == 0 {
		t.Fatalf("charts empty: %+v", r.Charts)
	}
	for _, f := range r.Findings {
		if len(f.Evidence) == 0 || f.Repro == "" {
			t.Fatalf("finding %s missing evidence/repro", f.ID)
		}
	}
}

// TestWriteReportCreatesHumanAndAIViews 验证一次 run 会同时生成人看 HTML、AI 上下文、manifest、history 和 checksums。
// 场景：P1 交付需要同一个 result.json 渲染两份视图；报告目录必须可离线打开并带 dirty 备注。
// 预期：report.html 含场景名/ECharts 回退/MOCK；ai/analysis-context.md 含 Run/Findings/dirty；
// manifest 记录 simHome/runId；history.jsonl 恰好一行；checksums 能校验文件。
// 覆盖：临时 lab + GenerateDemoResult + WriteReport，逐文件读取并解析 JSON/校验和。
func TestWriteReportCreatesHumanAndAIViews(t *testing.T) {
	home := t.TempDir()
	paths := Paths{
		Home:      home,
		Scenarios: filepath.Join(home, "scenarios"),
		Golden:    filepath.Join(home, "golden"),
		Baselines: filepath.Join(home, "baselines"),
		Reports:   filepath.Join(home, "reports"),
		Notes:     filepath.Join(home, "notes"),
	}
	if err := EnsureLayout(paths); err != nil {
		t.Fatal(err)
	}
	sc := DefaultDemoScenario()
	at := time.Date(2026, 11, 19, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	result := GenerateDemoResult(&sc, at)
	out, err := WriteReport(paths, result, "built-in:demo-report", "jutland-sim demo")
	if err != nil {
		t.Fatalf("WriteReport() error = %v", err)
	}
	for _, path := range []string{
		out.ReportHTML, out.Summary, out.AIContext,
		filepath.Join(out.Dir, "result.json"),
		filepath.Join(out.Dir, "manifest.json"),
		filepath.Join(out.Dir, "checksums.sha256"),
		filepath.Join(out.Dir, "ai", "findings.json"),
		filepath.Join(out.Dir, "ai", "changes.json"),
		filepath.Join(out.Dir, "charts", "hit_rate_vs_range.svg"),
		filepath.Join(paths.Reports, "history.jsonl"),
		filepath.Join(paths.Reports, "index.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}

	html, err := os.ReadFile(out.ReportHTML)
	if err != nil {
		t.Fatal(err)
	}
	required := []string{"demo-report", "echarts", "chart-fallback", "<svg", "P1 MOCK", "commit"}
	if CollectGitInfo("").Available {
		required = append(required, "工作区")
	}
	for _, want := range required {
		if !strings.Contains(string(html), want) {
			t.Fatalf("report.html missing %q", want)
		}
	}
	ai, err := os.ReadFile(out.AIContext)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## 1. Run", "## 3. Findings", "dirty:", "## 8. Repro"} {
		if !strings.Contains(string(ai), want) {
			t.Fatalf("analysis-context missing %q", want)
		}
	}

	var manifest Manifest
	data, err := os.ReadFile(filepath.Join(out.Dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.RunID != out.RunID || manifest.SimHome != home || !manifest.Mock {
		t.Fatalf("manifest = %+v", manifest)
	}
	var changes map[string]any
	changesData, err := os.ReadFile(filepath.Join(out.Dir, "ai", "changes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(changesData, &changes); err != nil {
		t.Fatal(err)
	}
	if changes["note"] != "P1 合成数据" {
		t.Fatalf("changes note = %v", changes["note"])
	}
	if manifest.ScenarioPath != "built-in:demo-report" {
		t.Fatalf("ScenarioPath = %q", manifest.ScenarioPath)
	}

	history, err := os.ReadFile(filepath.Join(paths.Reports, "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(history)), "\n"); lines != 0 {
		t.Fatalf("history should have one line, got %d newlines", lines)
	}
	if !strings.Contains(string(history), out.RunID) {
		t.Fatalf("history missing run id %s", out.RunID)
	}
	index, err := os.ReadFile(filepath.Join(paths.Reports, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), out.RunID) {
		t.Fatalf("index missing run id")
	}

	verifyChecksums(t, out.Dir)
}

// TestWriteReportRendersBattleReportAndFullLog 验证 F3 战报会进入 HTML/summary，完整战况另写 JSONL。
// 场景：游戏内战报包含总览、舰艇明细、关键战果和滚动战况；仿真报告只给一张生存率曲线时
// 无法复盘具体击沉过程，因此 F3 需要把这些区块一起渲染，并把全量日志留在 run 目录。
// 预期：HTML 含战报总览/舰艇明细/关键战果/战况记录/在役舰艇图；summary 含「## 战报」；
// battle-log.jsonl 一行一个事件且被 checksums 收录；result.json 有 battleReport 但没有完整 battleLog。
// 覆盖：临时 lab + 带 BattleReport/BattleLog 的 F3 result 调 WriteReport，逐文件断言并校验 checksums。
func TestWriteReportRendersBattleReportAndFullLog(t *testing.T) {
	home := t.TempDir()
	paths := Paths{
		Home:      home,
		Scenarios: filepath.Join(home, "scenarios"),
		Golden:    filepath.Join(home, "golden"),
		Baselines: filepath.Join(home, "baselines"),
		Reports:   filepath.Join(home, "reports"),
		Notes:     filepath.Join(home, "notes"),
	}
	if err := EnsureLayout(paths); err != nil {
		t.Fatal(err)
	}
	sc := DefaultDemoScenario()
	at := time.Date(2026, 11, 23, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
	result := GenerateDemoResult(&sc, at)
	result.Fidelity = "F3"
	result.BattleReport = &BattleReport{
		Side: "P2", Status: "success", EndReason: "victory", Ticks: 600,
		Cards: []BattleReportCard{
			{Label: "舰艇损失", Value: "1", Sub: "500 吨"},
			{Label: "敌方被击沉", Value: "1", Sub: "30,000 吨"},
		},
		Own: BattleReportSide{
			Label:          "我方",
			ShipsLine:      "2 / 0 / 1 / 1",
			PlanesLine:     "10 / 0 / 2 / 8",
			PlanesWithShip: "1",
			InitialTonnage: "1,000",
			SunkTonnage:    "500",
			ActiveTonnage:  "500",
			DamageDealt:    "1,234",
			Funds:          "100 → 85 (+5 / -20)",
		},
		Enemy: BattleReportSide{
			Label:          "敌方",
			ShipsLine:      "1 / 0 / 1 / 0",
			PlanesLine:     "0 / 0 / 0 / 0",
			PlanesWithShip: "0",
			InitialTonnage: "30,000",
			SunkTonnage:    "30,000",
			ActiveTonnage:  "0",
			DamageDealt:    "500",
			Funds:          "无数据",
		},
		OwnShipRows: []BattleReportDetailRow{{Name: "加贺", Initial: 1, Lost: 1}},
		Kills: []BattleReportEvent{{
			Tick: 10, Time: "00:10", Side: "我方", SideClass: "side-own",
			Summary: "航空母舰 加贺 610mm 鱼雷 击沉 战列舰 亚利桑那，吨位 30,000 吨", Kill: true,
		}},
		KillsTotal: 1,
		Events: []BattleReportEvent{{
			Tick: 20, Time: "00:20", Side: "我方", SideClass: "side-own",
			Summary: "航空母舰 加贺 127mm 防空炮 击落 战斗机 零式，造成 0.52 伤害",
		}},
		EventsTotal: 12, EventsOmitted: 11,
		Exchange: "2.00x", Survival: "50.0%",
		Trend: []BattleReportTrendPoint{
			{Minute: 0, OwnShips: 2, EnemyShips: 1, OwnPlanes: 10},
			{Minute: 10, OwnShips: 1, EnemyShips: 0, OwnPlanes: 8},
		},
	}
	result.BattleLog = []BattleReportEvent{{
		Tick: 10, Time: "00:10", Summary: "航空母舰 加贺 610mm 鱼雷 击沉 战列舰 亚利桑那", Kill: true,
	}}
	out, err := WriteReport(paths, result, "built-in:battle-report", "jutland-macro macro battle-report")
	if err != nil {
		t.Fatalf("WriteReport() error = %v", err)
	}

	html, err := os.ReadFile(out.ReportHTML)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"战报总览", "舰艇明细", "关键战果", "战况记录", "battle-log.jsonl",
		"在役舰艇", "航空母舰 加贺 610mm 鱼雷 击沉",
	} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("report.html missing %q", want)
		}
	}
	assertUniqueHTMLIDs(t, string(html))
	summary, err := os.ReadFile(out.Summary)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## 战报", "关键战果", "完整战况：12 条"} {
		if !strings.Contains(string(summary), want) {
			t.Fatalf("summary.md missing %q", want)
		}
	}
	logPath := filepath.Join(out.Dir, "battle-log.jsonl")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("battle-log.jsonl: %v", err)
	}
	if lines := strings.Count(strings.TrimSpace(string(logData)), "\n") + 1; lines != 1 {
		t.Fatalf("battle-log.jsonl lines = %d, want 1", lines)
	}
	if !strings.Contains(string(logData), "航空母舰 加贺 610mm 鱼雷 击沉") {
		t.Fatalf("battle-log.jsonl missing kill event")
	}
	resultData, err := os.ReadFile(filepath.Join(out.Dir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var resultJSON map[string]any
	if err := json.Unmarshal(resultData, &resultJSON); err != nil {
		t.Fatal(err)
	}
	if _, ok := resultJSON["battleReport"]; !ok {
		t.Fatalf("result.json missing battleReport")
	}
	if _, ok := resultJSON["battleLog"]; ok {
		t.Fatalf("result.json should not embed full battleLog")
	}
	verifyChecksums(t, out.Dir)
}

// assertUniqueHTMLIDs 保证报告里的 id 不重复。图表容器与章节曾共用 battle-ships，
// 导致 ECharts 把图初始化到 section 上而显示空白，必须由渲染层回归测试兜住。
func assertUniqueHTMLIDs(t *testing.T, html string) {
	t.Helper()
	seen := map[string]int{}
	for _, match := range regexp.MustCompile(`id="([^"]+)"`).FindAllStringSubmatch(html, -1) {
		seen[match[1]]++
	}
	for id, count := range seen {
		if count > 1 {
			t.Fatalf("duplicate HTML id %q (%d)", id, count)
		}
	}
}

func verifyChecksums(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "checksums.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatal("checksums is empty")
	}
	for _, line := range lines {
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			t.Fatalf("bad checksum line %q", line)
		}
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(parts[1])))
		if err != nil {
			t.Fatalf("read %s: %v", parts[1], err)
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != parts[0] {
			t.Fatalf("checksum mismatch for %s", parts[1])
		}
	}
}
