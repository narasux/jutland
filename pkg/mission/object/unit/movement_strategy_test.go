package unit

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestPlaneMovementSpeedNeverExceedsScaledMaxSpeed 验证飞机巡航/追击速度不超过
// 最大速度 × 全局速度倍率。
// 场景：5 组策略/倍率——战斗机追击 0.5x/1x/2x、基础巡航 1x/2x；飞机 MaxSpeed=0.05、
// Acceleration=0.01、RotateSpeed=8、初始 CurSpeed=0.05；敌机每帧绕圈（半径 0.5–6.5 格），
// targetSpeed 在 0/0.03/0.04/0.06/10 间轮换，覆盖远距全速、咬尾减速、目标更快等分支。
// 预期：600 帧内每帧 CurSpeed ≤ MaxSpeed×multiplier（0.025 / 0.05 / 0.1），且飞机确实移动过；
// 调试面板的“超速”只应来自起降阶段（滑跑 / 进近的世界系位移），不应出现在追击敌机时。
// 覆盖：表驱动 5 组，循环 600 帧调用 strategy.MoveTo 并逐帧断言速度上限，最后检查 moved。
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
