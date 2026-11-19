package sim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func floatPtr(v float64) *float64 { return &v }

func shotScenario() *Scenario {
	sc := DefaultDemoScenario()
	sc.Name = "f0-test"
	sc.Mode = ModeShot
	sc.Runs = 100
	sc.Seed = 7
	sc.Shot = &ShotSpec{
		Name: "test-shot", TargetName: "target", TargetHP: 500,
		HorizontalReduction: 0.6, VerticalReduction: 0.2,
		BaseDamage: 100, CriticalRate: 0, Plunge: 0, Count: 20,
		ExpectedDamage: floatPtr(40), Tolerance: 1e-9,
	}
	return &sc
}

func metricValue(t *testing.T, result *Result, name string) float64 {
	t.Helper()
	for _, row := range result.Metrics {
		if row.Name == name {
			return row.Value
		}
	}
	t.Fatalf("metric %q not found", name)
	return 0
}

// TestLoadShotScenarioFromTestdata 验证仓库内 shot-f0.json5 可被 CLI 同款加载器解析并跑出 40 伤害。
// 场景：P4 smoke 数据随游戏仓库提交，CLI 从 lab 的 scenarios 目录复制/加载；字段名必须与 Go struct 对齐。
// 预期：LoadScenario 无错，shot.targetHP=500、expectedDamage=40；RunShot 的 mean_applied_damage=40。
// 覆盖：直接加载 testdata/shot-f0.json5，检查 Shot 字段后调用 RunShot。
func TestLoadShotScenarioFromTestdata(t *testing.T) {
	sc, err := LoadScenario(filepath.Join("testdata", "shot-f0.json5"))
	if err != nil {
		t.Fatalf("LoadScenario() error = %v", err)
	}
	if sc.Shot == nil || sc.Shot.TargetHP != 500 || sc.Shot.ExpectedDamage == nil || *sc.Shot.ExpectedDamage != 40 {
		t.Fatalf("shot spec = %+v", sc.Shot)
	}
	result, err := RunShot(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := metricValue(t, result, "mean_applied_damage"); got != 40 {
		t.Fatalf("mean_applied_damage = %v, want 40", got)
	}
}

// TestRunShotMatchesKnownAnswer 验证 F0 台对 0.6 水平减伤、平射、100 基础伤害给出精确 40。
// 场景：目标 500HP、100 runs × 20 shots、无暴击；已知答案是 100*(1-0.6)=40。
// 预期：mean_applied_damage=40，fidelity=F0、Mock=false、无 mismatch finding，公式记分卡 green。
// 覆盖：shotScenario + RunShot，按指标名取值并检查 scorecard/finding。
func TestRunShotMatchesKnownAnswer(t *testing.T) {
	result, err := RunShot(shotScenario(), time.Now())
	if err != nil {
		t.Fatalf("RunShot() error = %v", err)
	}
	if result.Mock || result.Fidelity != "F0" {
		t.Fatalf("fidelity = %q mock=%t", result.Fidelity, result.Mock)
	}
	if got := metricValue(t, result, "mean_applied_damage"); got != 40 {
		t.Fatalf("mean_applied_damage = %v, want 40", got)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("findings = %+v, want none", result.Findings)
	}
	found := false
	for _, item := range result.Scorecard {
		if item.Dimension == "伤害公式" {
			found = item.Status == "green"
		}
	}
	if !found {
		t.Fatalf("damage formula scorecard not green: %+v", result.Scorecard)
	}
}

// TestRunShotCountsEffectiveAndOverkill 验证 100 伤害打 50HP 目标时 applied=100、effective=50、overkill=50。
// 场景：10 runs × 1 shot，目标一击必沉；过杀率应为 50%，不能被算成 0 或 100。
// 预期：overkill_rate=50，mean_effective_damage=50，kill_rate=100。
// 覆盖：构造无减伤/无暴击场景，RunShot 后读取三个指标。
func TestRunShotCountsEffectiveAndOverkill(t *testing.T) {
	sc := shotScenario()
	sc.Runs = 10
	sc.Shot.TargetHP = 50
	sc.Shot.Count = 1
	sc.Shot.HorizontalReduction = 0
	sc.Shot.VerticalReduction = 0
	sc.Shot.ExpectedDamage = floatPtr(100)
	result, err := RunShot(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := metricValue(t, result, "mean_effective_damage"); got != 50 {
		t.Fatalf("mean_effective_damage = %v, want 50", got)
	}
	if got := metricValue(t, result, "overkill_rate"); got != 50 {
		t.Fatalf("overkill_rate = %v, want 50", got)
	}
	if got := metricValue(t, result, "kill_rate"); got != 100 {
		t.Fatalf("kill_rate = %v, want 100", got)
	}
}

// TestRunShotExpectedMismatchFinding 验证已知答案不一致时生成高严重度 finding。
// 场景：实际 40 伤害但 expectedDamage=100、tolerance=0；报告必须显式暴露而不是绿色通过。
// 预期：存在 ID=F0-DAMAGE-MISMATCH、Severity=high 的 finding，公式记分卡 red。
// 覆盖：修改 expected 后 RunShot，遍历 findings/scorecard 断言。
func TestRunShotExpectedMismatchFinding(t *testing.T) {
	sc := shotScenario()
	sc.Shot.ExpectedDamage = floatPtr(100)
	result, err := RunShot(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range result.Findings {
		if finding.ID == "F0-DAMAGE-MISMATCH" && finding.Severity == "high" {
			found = true
		}
	}
	if !found {
		t.Fatalf("mismatch finding missing: %+v", result.Findings)
	}
	for _, item := range result.Scorecard {
		if item.Dimension == "伤害公式" && item.Status != "red" {
			t.Fatalf("formula status = %s, want red", item.Status)
		}
	}
}

// TestWriteReportF0IsNotMock 验证 F0 结果渲染成真实报告：F0 标识、无 MOCK 横幅、只生成有数据的图。
// 场景：P4 第一版真实报告必须能从 mock 版式无缝切换；空 exchange/penetration 不应产生破图。
// 预期：report.html 含 Fidelity <b>F0</b>、无 "P1 MOCK"、含"装甲减伤曲线"、无交换/穿透图；
// summary 与 AI context 都写 F0。
// 覆盖：临时 lab + RunShot + WriteReport，读取 HTML/Markdown 并检查 charts 目录文件。
func TestWriteReportF0IsNotMock(t *testing.T) {
	home := t.TempDir()
	paths := Paths{
		Home: home, Scenarios: filepath.Join(home, "scenarios"),
		Golden: filepath.Join(home, "golden"), Baselines: filepath.Join(home, "baselines"),
		Reports: filepath.Join(home, "reports"), Notes: filepath.Join(home, "notes"),
	}
	if err := EnsureLayout(paths); err != nil {
		t.Fatal(err)
	}
	result, err := RunShot(shotScenario(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	out, err := WriteReport(paths, result, "testdata/shot-f0.json5", "jutland-sim shot testdata/shot-f0.json5")
	if err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(out.ReportHTML)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Fidelity <b>F0</b>", "装甲减伤曲线", "击沉概率 vs 射击数"} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("report missing %q", want)
		}
	}
	if strings.Contains(string(html), "P1 MOCK") {
		t.Fatal("F0 report still contains P1 MOCK banner")
	}
	if _, err := os.Stat(filepath.Join(out.Dir, "charts", "exchange_matrix.svg")); !os.IsNotExist(err) {
		t.Fatalf("exchange chart should be absent, stat err=%v", err)
	}
	summary, err := os.ReadFile(out.Summary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summary), "fidelity：F0") || strings.Contains(string(summary), "P1 MOCK") {
		t.Fatalf("summary fidelity/mock wrong:\n%s", summary)
	}
	ai, err := os.ReadFile(out.AIContext)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ai), "fidelity: F0") {
		t.Fatalf("AI context missing F0: %s", ai)
	}
	verifyChecksums(t, out.Dir)
}
