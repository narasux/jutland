package damage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveShipArmorAndPlunge 验证水平/垂直装甲按落角线性混合，平射吃水平带、吊射吃垂直带。
// 场景：基础 100 伤害、水平减伤 0.6、垂直减伤 0.2；这是舰船 HurtBy 的核心公式，F0 台会复用。
// 预期：plunge=0 -> 40，plunge=1 -> 80，plunge=0.5 -> 60；失败说明落角插值或减伤方向错了。
// 覆盖：表驱动三个落角，断言 realDamage 与无暴击倍率 1。
func TestResolveShipArmorAndPlunge(t *testing.T) {
	cases := []struct {
		name   string
		plunge float64
		want   float64
	}{
		{name: "flat", plunge: 0, want: 40},
		{name: "vertical", plunge: 1, want: 80},
		{name: "half", plunge: 0.5, want: 60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, mult := ResolveShip(100, 0.6, 0.2, tc.plunge, 0, 0.5)
			require.InDelta(t, tc.want, got, 1e-9)
			require.Equal(t, 1, mult)
		})
	}
}

// TestResolveShipCriticalBands 验证暴击判定：<rate/10 十倍，<rate 三倍，否则普通。
// 场景：criticalRate=1.0 覆盖十倍/三倍，0.8 覆盖普通；observer/F0 需要正确统计暴击倍率。
// 预期：randVal=0.05 -> 10x，0.5 -> 3x，0.95 -> 1x；失败说明暴击阈值或乘法错了。
// 覆盖：表驱动三个 randVal，断言 realDamage 与 criticalMultiplier。
func TestResolveShipCriticalBands(t *testing.T) {
	cases := []struct {
		randVal float64
		rate    float64
		want    float64
		mult    int
	}{
		{randVal: 0.05, rate: 1.0, want: 1000, mult: 10},
		{randVal: 0.5, rate: 1.0, want: 300, mult: 3},
		{randVal: 0.95, rate: 0.8, want: 100, mult: 1},
	}
	for _, tc := range cases {
		got, mult := ResolveShip(100, 0, 0, 0, tc.rate, tc.randVal)
		require.InDelta(t, tc.want, got, 1e-9)
		require.Equal(t, tc.mult, mult)
	}
}
