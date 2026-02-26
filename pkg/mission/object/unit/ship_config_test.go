package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/stretchr/testify/assert"
)

// TestBattleShipConfigSightRangeLoading 测试战舰配置中的sightRange字段加载
func TestBattleShipConfigSightRangeLoading(t *testing.T) {
	tests := []struct {
		name          string
		ship          *BattleShip
		expectedRange float64
		description   string
	}{
		{
			name: "特殊战舰使用自定义视野范围",
			ship: &BattleShip{
				Uid:          "duck",
				BelongPlayer: faction.HumanAlpha,
				CurHP:        100,
				CurPos:       objPos.NewR(0, 0),
				SightRange:   30, // 从配置加载的自定义值
				Weapon: ShipWeapon{
					MaxToShipRange: 15,
				},
			},
			expectedRange: 30,
			description:   "特殊战舰应使用配置中的自定义视野范围",
		},
		{
			name: "航空母舰使用配置的视野范围",
			ship: &BattleShip{
				Uid:          "yorktown",
				BelongPlayer: faction.HumanAlpha,
				CurHP:        100,
				CurPos:       objPos.NewR(0, 0),
				SightRange:   25, // 从配置加载的值
				Weapon: ShipWeapon{
					MaxToShipRange: 10,
				},
			},
			expectedRange: 25,
			description:   "航空母舰应使用配置中的视野范围",
		},
		{
			name: "货轮使用较小的视野范围",
			ship: &BattleShip{
				Uid:          "liberty",
				BelongPlayer: faction.HumanAlpha,
				CurHP:        100,
				CurPos:       objPos.NewR(0, 0),
				SightRange:   8, // 从配置加载的值
				Weapon: ShipWeapon{
					MaxToShipRange: 5,
				},
			},
			expectedRange: 8,
			description:   "货轮应使用较小的配置视野范围",
		},
		{
			name: "战列舰使用默认值计算",
			ship: &BattleShip{
				Uid:          "lowa",
				BelongPlayer: faction.HumanAlpha,
				CurHP:        100,
				CurPos:       objPos.NewR(0, 0),
				SightRange:   0, // 使用默认值
				Weapon: ShipWeapon{
					MaxToShipRange: 20,
				},
			},
			expectedRange: 25, // 20 + 5
			description:   "战列舰应使用默认值计算（武器射程 + 5）",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.ship.GetSightRange()
			assert.Equal(t, test.expectedRange, actual, test.description)
		})
	}
}

// TestBattleShipConfigValidation 测试配置验证逻辑
func TestBattleShipConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		sightRange    float64
		expectedRange float64
		description   string
	}{
		{
			name:          "负值配置应使用默认值",
			sightRange:    -5,
			expectedRange: 15, // 10 + 5
			description:   "负值配置应被忽略并使用默认值计算",
		},
		{
			name:          "零值配置应使用默认值",
			sightRange:    0,
			expectedRange: 15, // 10 + 5
			description:   "零值配置应使用默认值计算",
		},
		{
			name:          "正值配置应使用自定义值",
			sightRange:    12,
			expectedRange: 12,
			description:   "正值配置应使用自定义值",
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
					MaxToShipRange: 10,
				},
			}
			actual := ship.GetSightRange()
			assert.Equal(t, test.expectedRange, actual, test.description)
		})
	}
}
