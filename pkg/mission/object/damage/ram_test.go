package damage

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveRamScalesWithSpeedAngleAndMass 验证撞击伤害随速度平方、角度与质量比缩放。
// 场景：ram.damage=1800、目标水平减伤 0.24、自伤比 0.25；覆盖满速正撞、半速、
// 60° 斜撞、小舰撞大舰（质量比下限 0.1）、目标免疫与非法输入。
// 预期：满速正撞 base=1800、目标 1368、自伤 450；半速 base=450；60° 角 factor≈0.3536；
// 目标减伤 1 时目标伤害为 0；自伤随目标/撞击舰吨位比缩小；NaN/负速度返回零值且无 NaN。
// 覆盖：表驱动五组输入，断言 base/target/self/speedFactor/angleFactor 的具体数值；
// 失败说明撞击伤害会把速度、角度或吨位方向算反，或非法配置污染战斗数值。
func TestResolveRamScalesWithSpeedAngleAndMass(t *testing.T) {
	cases := []struct {
		name            string
		closing         float64
		bearing         float64
		attackerTonnage float64
		targetTonnage   float64
		targetReduction float64
		wantBase        float64
		wantTarget      float64
		wantSelf        float64
	}{
		{
			name: "full-speed-head-on", closing: 13.6, bearing: 0,
			attackerTonnage: 4084, targetTonnage: 4000, targetReduction: 0.24,
			wantBase: 1800, wantTarget: 1368, wantSelf: 1800 * 0.25 * (4000.0 / 4084.0),
		},
		{
			name: "half-speed", closing: 6.8, bearing: 0,
			attackerTonnage: 4084, targetTonnage: 4084, targetReduction: 0,
			wantBase: 450, wantTarget: 450, wantSelf: 450 * 0.25,
		},
		{
			name: "oblique-60", closing: 13.6, bearing: 60,
			attackerTonnage: 4084, targetTonnage: 4084, targetReduction: 0,
			wantBase:   1800 * math.Pow(math.Cos(60*math.Pi/180), 1.5),
			wantTarget: 1800 * math.Pow(math.Cos(60*math.Pi/180), 1.5),
			wantSelf:   1800 * math.Pow(math.Cos(60*math.Pi/180), 1.5) * 0.25,
		},
		{
			name: "small-rams-large", closing: 13.6, bearing: 0,
			attackerTonnage: 2000, targetTonnage: 72500, targetReduction: 0.55,
			wantBase: 1800, wantTarget: 1800 * 0.45, wantSelf: 1800 * 0.25,
		},
		{
			name: "immune-target", closing: 13.6, bearing: 0,
			attackerTonnage: 4084, targetTonnage: 4084, targetReduction: 1,
			wantBase: 1800, wantTarget: 0, wantSelf: 450,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveRam(
				1800, tc.closing, 13.6, tc.bearing, 0.25,
				tc.attackerTonnage, tc.targetTonnage, tc.targetReduction,
			)
			require.InDelta(t, tc.wantBase, got.BaseDamage, 1e-9)
			require.InDelta(t, tc.wantTarget, got.TargetDamage, 1e-9)
			require.InDelta(t, tc.wantSelf, got.SelfDamage, 1e-9)
			require.False(t, math.IsNaN(got.TargetDamage) || math.IsNaN(got.SelfDamage))
		})
	}

	zero := ResolveRam(1800, 0, 13.6, 0, 0.25, 4000, 4000, 0)
	require.Zero(t, zero.TargetDamage)
	require.Zero(t, zero.SelfDamage)
	nan := ResolveRam(1800, math.NaN(), 13.6, 0, 0.25, 4000, 4000, 0)
	require.Zero(t, nan.TargetDamage)
	require.Zero(t, nan.SelfDamage)
	// 吨位缺省时质量比按 1 处理，不能出现除零。
	noMass := ResolveRam(1800, 13.6, 13.6, 0, 0.25, 0, 0, 0.24)
	require.InDelta(t, 1800*0.25, noMass.SelfDamage, 1e-9)
}
