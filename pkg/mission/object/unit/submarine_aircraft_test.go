package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// TestCanOperateAircraftRequiresSurface 验证只有浮航状态的潜艇能起降飞机。
// 场景：伊-400 编入晴岚后弹射器与水面回收点都在艇体上；下潜、上浮、下潜过渡中
// 都必须禁止起降，而驱逐舰等非潜艇基地的水面状态与其深度字段无关。
// 预期：潜艇 DiveStateSurface 返回 true，Diving/Submerged/Surfacing 返回 false；
// 非潜艇舰船即使深度字段是默认零值也返回 true。
// 覆盖：表驱动潜艇的四个深度状态 + 一艘驱逐舰，逐一交给 CanOperateAircraft 断言返回值。
func TestCanOperateAircraftRequiresSurface(t *testing.T) {
	sub := func(state DiveState) *BattleShip {
		return &BattleShip{
			Type:      ShipTypeSubmarine,
			Submarine: SubmarineState{DiveState: state},
		}
	}
	testCases := []struct {
		name string
		base AircraftBase
		want bool
	}{
		{name: "submarine-surface", base: sub(DiveStateSurface), want: true},
		{name: "submarine-diving", base: sub(DiveStateDiving), want: false},
		{name: "submarine-submerged", base: sub(DiveStateSubmerged), want: false},
		{name: "submarine-surfacing", base: sub(DiveStateSurfacing), want: false},
		{name: "destroyer", base: &BattleShip{Type: ShipTypeDestroyer}, want: true},
		{name: "missing-base", base: (*BattleShip)(nil), want: false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanOperateAircraft(tc.base); got != tc.want {
				t.Fatalf("CanOperateAircraft(%s) = %t, want %t", tc.name, got, tc.want)
			}
		})
	}
}

// TestSubmarineTakeOffWaitsForSurface 验证下潜中的潜艇不能放飞舰载机。
// 场景：伊-400 平时保持下潜，玩家让它上浮后才弹射晴岚；弹射器在水下时放机
// 会让飞机凭空出现在水面，所以同一编组在两种深度状态下各起飞一次。
// 预期：DiveStateSurface 时 TakeOff 返回 M6A1 且库存从 3 减到 2；切到
// DiveStateSubmerged 后 TakeOff 返回 nil，库存保持 2。
// 覆盖：registerTestPlane 注册临时鱼雷机，ResolveDeck 绑定单弹射点甲板，
// 断言返回机名、nil 与 CurCount。
func TestSubmarineTakeOffWaitsForSurface(t *testing.T) {
	useDefaultSettings(t)
	registerTestPlane(t, "test-submarine-torpedo", PlaneTypeTorpedoBomber)

	ship := &BattleShip{
		Uid:         "i-400-test",
		Type:        ShipTypeSubmarine,
		Length:      122,
		Width:       12,
		CurPos:      objPos.NewR(50, 50),
		CurRotation: 0,
		Aircraft: ShipAircraft{
			Groups: []PlaneGroup{{
				Name:       "test-submarine-torpedo",
				MaxCount:   3,
				CurCount:   3,
				TargetType: object.TypeShip,
			}},
		},
		Submarine: SubmarineState{DiveState: DiveStateSurface},
	}
	ship.Aircraft.ResolveDeck(&CarrierDeck{
		Name:          "submarine-catapult",
		TakeoffPoints: []TakeoffPoint{{Forward: 0.2, Lateral: 0, RunLength: 0.22}},
		Landing:       LandingConfig{Mode: LandingModeSea, Forward: 0.5, Lateral: 1.1},
	})

	plane := ship.Aircraft.TakeOff(ship, object.TypeShip)
	if plane == nil {
		t.Fatalf("surface submarine refused to launch aircraft")
	}
	if plane.Name != "test-submarine-torpedo" {
		t.Fatalf("launched plane = %q, want test-submarine-torpedo", plane.Name)
	}
	if got := ship.Aircraft.Groups[0].CurCount; got != 2 {
		t.Fatalf("stock after launch = %d, want 2", got)
	}

	ship.Submarine.DiveState = DiveStateSubmerged
	if plane := ship.Aircraft.TakeOff(ship, object.TypeShip); plane != nil {
		t.Fatalf("submerged submarine launched plane %q", plane.Name)
	}
	if got := ship.Aircraft.Groups[0].CurCount; got != 2 {
		t.Fatalf("stock changed while submerged = %d, want 2", got)
	}
}
