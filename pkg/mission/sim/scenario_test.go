package sim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadScenarioAppliesDefaults 验证 JSON5 场景的默认值、slug 与稳定 hash。
// 场景：文件只写 name/mode，省略 schemaVersion/runs/maxTicks/speedMultiplier/map。
// 预期：LoadScenario 补 schemaVersion=1、runs=1、maxTicks=3600、speed=1.0、map=open_sea；
// slug 为 demo-shot，hash 非空且同内容两次加载一致。
// 覆盖：写入临时 JSON5，调用 LoadScenario 两次，断言字段、Slug 与 Hash。
func TestLoadScenarioAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scenario.json5")
	if err := os.WriteFile(path, []byte(`{
		name: "Demo Shot",
		mode: "micro",
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	sc, err := LoadScenario(path)
	if err != nil {
		t.Fatalf("LoadScenario() error = %v", err)
	}
	if sc.SchemaVersion != 1 || sc.Runs != 1 || sc.MaxTicks != 3600 {
		t.Fatalf("defaults = schema %d runs %d ticks %d", sc.SchemaVersion, sc.Runs, sc.MaxTicks)
	}
	if sc.Environment.SpeedMultiplier != 1.0 || sc.Environment.Map != "open_sea" {
		t.Fatalf("environment = %+v", sc.Environment)
	}
	if sc.Slug() != "demo-shot" {
		t.Fatalf("Slug() = %q, want demo-shot", sc.Slug())
	}
	again, err := LoadScenario(path)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Hash() == "" || sc.Hash() != again.Hash() {
		t.Fatalf("hash = %q / %q, want stable non-empty", sc.Hash(), again.Hash())
	}
}

// TestScenarioValidationRejectsBadInput 验证非法场景在加载阶段就被拒绝。
// 场景：空 name、未知 mode、runs 超上限、speedMultiplier 非 0.5/1/2、单位同时写 ship/plane、
// pos 不是 2 个数、speedRatio 超 1。
// 预期：每个 case 都返回非 nil 错误，且错误文本包含对应字段名。
// 覆盖：表驱动 7 个 Scenario，调用 Validate 断言错误关键字。
func TestScenarioValidationRejectsBadInput(t *testing.T) {
	base := func() Scenario {
		sc := DefaultDemoScenario()
		sc.Name = "ok"
		sc.Forces = nil
		return sc
	}
	cases := []struct {
		name   string
		mutate func(*Scenario)
		want   string
	}{
		{name: "no name", mutate: func(s *Scenario) { s.Name = " " }, want: "name"},
		{name: "bad mode", mutate: func(s *Scenario) { s.Mode = "nope" }, want: "mode"},
		{name: "too many runs", mutate: func(s *Scenario) { s.Runs = 10001 }, want: "runs"},
		{name: "bad speed", mutate: func(s *Scenario) { s.Environment.SpeedMultiplier = 3 }, want: "speedMultiplier"},
		{name: "unit both", mutate: func(s *Scenario) {
			s.Forces = map[string][]UnitSpec{"alpha": {{Ship: "a", Plane: "b"}}}
		}, want: "both"},
		{name: "bad pos", mutate: func(s *Scenario) {
			s.Forces = map[string][]UnitSpec{"alpha": {{Ship: "a", Pos: []float64{1}}}}
		}, want: "pos"},
		{name: "bad speed ratio", mutate: func(s *Scenario) {
			s.Forces = map[string][]UnitSpec{"alpha": {{Ship: "a", SpeedRatio: 2}}}
		}, want: "speedRatio"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc := base()
			tc.mutate(&sc)
			err := sc.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

// TestDuelTerrainValidation 验证 duel.terrain 行宽一致且至少 20x10。
// 场景：合同作战可选用 ASCII 地形；行宽不一致会让 MapData 越界，必须在 Load/Validate 阶段拒绝。
// 预期：10 行 × 20 列通过；第 2 行 19 列时返回包含 "width" 的错误。
// 覆盖：同一 DuelSpec 先合法 terrain 再破坏一行，调用 ApplyDefaults+Validate。
func TestDuelTerrainValidation(t *testing.T) {
	rows := make([]string, 10)
	for i := range rows {
		rows[i] = strings.Repeat(".", 20)
	}
	sc := DefaultDemoScenario()
	sc.Mode = ModeDuel
	sc.Duel = &DuelSpec{ShooterShip: "jiujiang", TargetShip: "jiujiang", Distance: 5, Terrain: rows}
	sc.ApplyDefaults()
	if err := sc.Validate(); err != nil {
		t.Fatalf("valid terrain rejected: %v", err)
	}
	sc.Duel.Terrain[1] = strings.Repeat(".", 19)
	if err := sc.Validate(); err == nil || !strings.Contains(err.Error(), "width") {
		t.Fatalf("bad row error = %v, want width", err)
	}
}

// TestResolveScenarioPath 验证相对路径、无扩展名和绝对路径的解析。
// 场景：lab/scenarios 下存在 dd.json5；分别传 "dd"、"dd.json5" 和绝对路径。
// 预期：前两者解析到同一绝对文件；绝对路径原样返回；空参数报错。
// 覆盖：t.TempDir 建 scenarios，写文件，调用 ResolveScenarioPath 三次并断言。
func TestResolveScenarioPath(t *testing.T) {
	home := t.TempDir()
	paths := Paths{Home: home, Scenarios: filepath.Join(home, "scenarios")}
	if err := os.MkdirAll(paths.Scenarios, 0o755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(paths.Scenarios, "dd.json5")
	if err := os.WriteFile(full, []byte(`{name:"dd",mode:"shot"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, arg := range []string{"dd", "dd.json5"} {
		got, err := ResolveScenarioPath(paths, arg)
		if err != nil {
			t.Fatalf("ResolveScenarioPath(%q) error = %v", arg, err)
		}
		if got != full {
			t.Fatalf("ResolveScenarioPath(%q) = %q, want %q", arg, got, full)
		}
	}
	got, err := ResolveScenarioPath(paths, full)
	if err != nil || got != full {
		t.Fatalf("absolute path = %q, %v", got, err)
	}
	if _, err := ResolveScenarioPath(paths, " "); err == nil {
		t.Fatal("empty scenario path should error")
	}
}

// TestLoadScenarioParsesEnablePenetration 验证场景可以按局开启生产穿甲伤害开关。
// 场景：穿甲伤害是实验性机制（默认关闭）；旧场景没有该字段时不能悄悄打开，
// 新场景显式写 environment.enablePenetration: true 时才生效。
// 预期：写 true 的场景解析为 true；省略字段的场景为 false。
// 覆盖：两个临时 JSON5，LoadScenario 后断言 Environment.EnablePenetration。
func TestLoadScenarioParsesEnablePenetration(t *testing.T) {
	dir := t.TempDir()
	write := func(name, extra string) *Scenario {
		t.Helper()
		path := filepath.Join(dir, name)
		body := `{ name: "` + name + `", mode: "duel", duel: { shooterShip: "jiujiang", targetShip: "jiujiang" }` + extra + ` }`
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		sc, err := LoadScenario(path)
		require.NoError(t, err)
		return sc
	}

	enabled := write("pen-on.json5", `, environment: { enablePenetration: true }`)
	require.True(t, enabled.Environment.EnablePenetration)
	disabled := write("pen-off.json5", "")
	require.False(t, disabled.Environment.EnablePenetration)
}
