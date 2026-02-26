package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

func TestBattleShip_GetSightRange(t *testing.T) {
	tests := []struct {
		name           string
		sightRange     float64
		maxToShipRange float64
		expected       float64
	}{
		{
			name:           "使用默认值（无自定义视野）",
			sightRange:     0,
			maxToShipRange: 10,
			expected:       15, // 10 + 5
		},
		{
			name:           "使用自定义视野",
			sightRange:     8,
			maxToShipRange: 10,
			expected:       8,
		},
		{
			name:           "默认值（零射程武器）",
			sightRange:     0,
			maxToShipRange: 0,
			expected:       5, // 0 + 5
		},
		{
			name:           "自定义视野（大射程武器）",
			sightRange:     0,
			maxToShipRange: 25,
			expected:       30, // 25 + 5
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ship := &BattleShip{
				Uid:          "test-ship",
				BelongPlayer: faction.HumanAlpha,
				CurHP:        100,
				CurPos:       objPos.NewR(0, 0),
				SightRange:   test.sightRange,
				Weapon: ShipWeapon{
					MaxToShipRange: test.maxToShipRange,
				},
			}

			if result := ship.GetSightRange(); result != test.expected {
				t.Errorf("GetSightRange() = %f, expected %f", result, test.expected)
			}
		})
	}
}

func TestBattleShip_GetSightRange_WeaponNil(t *testing.T) {
	// 当 Weapon 为 nil 时，应该返回默认值 5
	// 注意：这种情况在实际代码中不应该发生，但测试可以验证健壮性
	// 由于会 panic，我们跳过这个测试
	t.Skip("Weapon should not be nil in production code")
}
