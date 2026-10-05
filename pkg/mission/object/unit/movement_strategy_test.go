package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestPlaneMovementSpeedNeverExceedsScaledMaxSpeed 固定“巡航 / 追击不超速”的契约：
// 战斗机追踪与轰炸机巡航都不得超过 最大速度 × 全局速度倍率。
// 调试面板上出现的“速度超过上限”只应来自起降阶段（滑跑 / 进近的世界系位移），
// 不应出现在追击敌机时。CurSpeed 是世界系速度（已乘倍率），比较基准必须同样乘倍率。
func TestPlaneMovementSpeedNeverExceedsScaledMaxSpeed(t *testing.T) {
	oldSettings := config.G
	t.Cleanup(func() { config.G = oldSettings })

	cases := []struct {
		name       string
		strategy   MovementStrategy
		multiplier float64
	}{
		{name: "fighter-slow-motion", strategy: &FighterPursuitStrategy{}, multiplier: 0.5},
		{name: "fighter-standard", strategy: &FighterPursuitStrategy{}, multiplier: 1},
		{name: "fighter-fast", strategy: &FighterPursuitStrategy{}, multiplier: 2},
		{name: "bomber-standard", strategy: &BasicMovementStrategy{}, multiplier: 1},
		{name: "bomber-fast", strategy: &BasicMovementStrategy{}, multiplier: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config.G = &config.GameSettings{SpeedMultiplier: tc.multiplier}
			plane := &Plane{
				CurHP:        100,
				MaxSpeed:     0.05,
				Acceleration: 0.01,
				RotateSpeed:  8,
				RemainRange:  10000,
				CurSpeed:     0.05,
				CurPos:       objPos.NewR(50, 50),
			}
			speedCeiling := plane.MaxSpeed * tc.multiplier
			moved := false
			for frame := range 600 {
				// 敌机绕飞机转圈，覆盖远距全速、咬尾减速、目标更快等所有分支。
				radians := float64(frame) * 3 * math.Pi / 180
				radius := 0.5 + math.Mod(float64(frame)*0.05, 6)
				enemyPos := objPos.NewR(50+math.Cos(radians)*radius, 50+math.Sin(radians)*radius)
				targetSpeed := []float64{0, 0.03, 0.04, 0.06, 10}[frame%5]
				tc.strategy.MoveTo(plane, nil, enemyPos, enemyPos, targetSpeed)
				if plane.CurSpeed > speedCeiling+1e-9 {
					t.Fatalf(
						"frame %d: CurSpeed=%v 超过上限 %v（追击/巡航不应超速）",
						frame, plane.CurSpeed, speedCeiling,
					)
				}
				moved = moved || plane.CurSpeed > 0
			}
			if !moved {
				t.Fatal("飞机始终没有速度，测试没有覆盖到移动逻辑")
			}
		})
	}
}
