package duel

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/sim"
)

// TestRunDuelAAF2 验证防空链路：Bismarck 的防空火力至少击落一架舰载机。
// 场景：testdata/duel-aa.json5，Enterprise 100000HP vs Bismarck 50000HP，距离 5，12000 tick。
// 预期：Fidelity=F2、aircraft_losses>0；不要求 plane_shots>0（飞机可能在投弹前被击落）。
// 覆盖：LoadScenario -> RunDuel，按 metric 名断言 aircraft_losses。
func TestRunDuelAAF2(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-aa.json5"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunDuel(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F2", result.Fidelity)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Greater(t, metrics["aircraft_losses"], 0.0)
	for _, finding := range result.Findings {
		require.NotEqual(t, "F2-AIR-NO-SORTIE", finding.ID)
	}
}

// TestRunDuelFogF2 验证迷雾 + 舰载机合同作战 smoke：开启 FogOfWar 后航母仍能发现/攻击近距目标。
// 场景：testdata/duel-carrier-fog.json5，航母 vs 5000HP 驱逐舰，距离 5，迷雾开启。
// 预期：Fidelity=F2、plane_shots>0、enemy_sunk=1；失败说明 Visions 初始化或迷雾下的可见性链路断了。
// 覆盖：LoadScenario -> RunDuel，检查空中射击与击沉。
func TestRunDuelFogF2(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-carrier-fog.json5"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunDuel(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F2", result.Fidelity)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Greater(t, metrics["plane_shots"], 0.0)
	require.Equal(t, 1.0, metrics["enemy_sunk"])
}

// TestRunDuelSubmarineF2 验证潜艇合同作战：下潜潜艇用鱼雷击沉水面舰，且自身未被无反潜目标点亮。
// 场景：testdata/duel-sub.json5，su_k_class vs jiujiang，距离 5，目标 1000HP，3600 tick。
// 预期：Fidelity=F2、sub_shots>0、enemy_sunk=1、own_sunk=0；失败说明水下可视性/鱼雷路径或 headless 指令执行坏了。
// 覆盖：LoadScenario -> RunDuel，按 metric 名断言水下射击与结果。
func TestRunDuelSubmarineF2(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-sub.json5"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunDuel(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F2", result.Fidelity)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Greater(t, metrics["sub_shots"], 0.0)
	require.Equal(t, 1.0, metrics["enemy_sunk"])
	require.Equal(t, 0.0, metrics["own_sunk"])
}

// TestRunDuelASWF2 验证反潜链路：Porter 声呐点亮下潜潜艇并投放深弹。
// 场景：testdata/duel-asw.json5，距离 1，潜艇初始 800HP；潜艇会先手鱼雷，重点不是胜负而是 asw_shots。
// 预期：Fidelity=F2、asw_shots>0、sub_shots>0；失败说明声呐/深弹/水下目标选择断了。
// 覆盖：LoadScenario -> RunDuel，按 metric 名断言 ASW 与潜艇射击都发生。
func TestRunDuelASWF2(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-asw.json5"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunDuel(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F2", result.Fidelity)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Greater(t, metrics["asw_shots"], 0.0)
	require.Greater(t, metrics["sub_shots"], 0.0)
}

// TestRunDuelCarrierF2 验证合同作战 smoke：Enterprise 舰载机起飞、投弹并参与击沉。
// 场景：testdata/duel-carrier.json5，航母 vs 5000HP 驱逐舰，maxTicks=6000；舰炮射程内但目标血量足以等到飞机。
// 预期：Fidelity=F2、plane_shots/plane_hits > 0、至少 1 个 plane kill、无 F2-AIR-NO-SORTIE。
// 覆盖：LoadScenario -> RunDuel，按 metric 名断言空中事件；失败说明 headless 没有执行 sortie 指令。
func TestRunDuelCarrierF2(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-carrier.json5"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunDuel(sc, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F2", result.Fidelity)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Greater(t, metrics["plane_shots"], 0.0)
	require.Greater(t, metrics["plane_hits"], 0.0)
	require.Greater(t, metrics["plane_kills"], 0.0)
	for _, finding := range result.Findings {
		require.NotEqual(t, "F2-AIR-NO-SORTIE", finding.ID)
	}
}

// TestRunDuelFleetAggregatesRuns 验证多舰/多 seed 聚合：2v1 focus 打 5 个 seed 都击沉且零自损。
// 场景：testdata/duel-fleet.json5，shooter 2 艘 vs target 1 艘（800HP），runs=5，doctrine=focus。
// 预期：Fidelity=F1、Runs=5、wins=5、enemy_sunk=5、own_sunk=0、exchange_ratio=999、Survival 非空。
// 覆盖：LoadScenario -> RunDuel，按 metric 名断言聚合值，检查没有 F1-CENSORED/FRIENDLY finding。
func TestRunDuelFleetAggregatesRuns(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-fleet.json5"))
	if err != nil {
		t.Fatalf("LoadScenario() error = %v", err)
	}
	result, err := RunDuel(sc, time.Now())
	if err != nil {
		t.Fatalf("RunDuel() error = %v", err)
	}
	require.Equal(t, 5, result.Runs)
	require.NotEmpty(t, result.Charts.Survival)
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Equal(t, 5.0, metrics["wins"])
	require.Equal(t, 5.0, metrics["enemy_sunk"])
	require.Equal(t, 0.0, metrics["own_sunk"])
	require.Equal(t, 999.0, metrics["exchange_ratio"])
	for _, finding := range result.Findings {
		require.NotEqual(t, "F1-CENSORED", finding.ID)
		require.NotEqual(t, "F1-FRIENDLY-FIRE", finding.ID)
	}
}

// TestRunDuelSameSeedProducesSameMetrics 验证同 seed 的两次 F1 对射指标一致。
// 场景：真实配置、固定 RNG/UID/时钟；若还有 map 序或全局随机泄漏，shots/hits/伤害会漂移。
// 预期：两次 Metrics 与 Scorecard 完全一致（DurationMS 不比）；Fidelity 都是 F1。
// 覆盖：同一 scenario 调 RunDuel 两次，比较 Metrics/Scorecard/Findings。
func TestRunDuelSameSeedProducesSameMetrics(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-jiujiang.json5"))
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 19, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	a, err := RunDuel(sc, at)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RunDuel(sc, at)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "F1", a.Fidelity)
	require.Equal(t, a.Metrics, b.Metrics)
	require.Equal(t, a.Scorecard, b.Scorecard)
	require.Equal(t, a.Findings, b.Findings)
}

// TestRunDuelSinksTarget 验证 F1 用真实 ShipMap/GunMap/BulletMap 跑出射击、命中与击沉。
// 场景：testdata/duel-jiujiang.json5，两艘同型 100mm 护卫舰相距 4 格，目标初始 500HP，seed=7。
// 预期：Fidelity=F1、Shots/Hits 非空、Sinks=1、无 F1-NO-SHOTS/友伤 finding；结果不是 mock。
// 覆盖：LoadScenario -> RunDuel 1200 tick，断言结果字段、finding 与 charts 为空（F1 暂不画图）。
func TestRunDuelSinksTarget(t *testing.T) {
	sc, err := sim.LoadScenario(filepath.Join("..", "testdata", "duel-jiujiang.json5"))
	if err != nil {
		t.Fatalf("LoadScenario() error = %v", err)
	}
	result, err := RunDuel(sc, time.Date(2026, 11, 19, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)))
	if err != nil {
		t.Fatalf("RunDuel() error = %v", err)
	}
	require.False(t, result.Mock)
	require.Equal(t, "F1", result.Fidelity)
	require.NotEmpty(t, result.Metrics)
	require.NotEmpty(t, result.Charts.Survival)
	require.Zero(t, len(result.Charts.HitRate)+len(result.Charts.Exchange)+len(result.Charts.Penetration))
	for _, finding := range result.Findings {
		require.NotEqual(t, "F1-NO-SHOTS", finding.ID)
		require.NotEqual(t, "F1-FRIENDLY-FIRE", finding.ID)
	}
	metrics := map[string]float64{}
	for _, row := range result.Metrics {
		metrics[row.Name] = row.Value
	}
	require.Greater(t, metrics["shots"], 0.0)
	require.Greater(t, metrics["hits"], 0.0)
	require.Equal(t, 1.0, metrics["enemy_sunk"])
	for _, item := range result.Scorecard {
		if item.Dimension == "结果" {
			require.Equal(t, "green", item.Status)
		}
	}
}
