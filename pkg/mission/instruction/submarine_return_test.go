package instruction

import (
	"testing"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestPlaneReturnHoldsWhileSubmergedBase 验证潜艇下潜时返航机在艇位附近盘旋等待。
// 场景：晴岚投完鱼雷返航，伊-400 在它回家途中下潜（DiveStateSubmerged），
// 艇侧水面不能降落；若直接开始待场/进近，飞机会停在一艘完全不露头的艇旁。
// 预期：40 拍内飞行阶段始终是巡航、飞机仍在 Arena.Planes、CurCount 保持 0，
// 与潜艇的距离比返航起点更近，剩余航程有消耗；潜艇改回水面后的下一拍照常进入
// LandingStaging。
// 覆盖：newLandingTestState 造一艘临时潜艇母舰，把返航机放在艇位 12 格外，
// 逐拍执行 PlaneReturn 断言阶段/距离/航程，再切换 DiveStateSurface 断言阶段推进。
func TestPlaneReturnHoldsWhileSubmergedBase(t *testing.T) {
	ms, ship, planes := newLandingTestState(t, "test-submerged-return", 1)
	plane := planes[0]
	ship.Type = objUnit.ShipTypeSubmarine
	ship.Length = 122
	ship.Width = 12
	ship.Submarine.DiveState = objUnit.DiveStateSubmerged
	plane.CurPos = objPos.NewR(50, 62)

	returnInstr := NewPlaneReturn(plane.Uid)
	startDistance := plane.CurPos.Distance(ship.CurPos)
	startRange := plane.RemainRange

	for frame := 0; frame < 40; frame++ {
		execPlaneReturn(t, returnInstr, ms)
		if plane.Flight.Phase != objUnit.PlaneFlightPhaseCruising {
			t.Fatalf("frame %d: phase = %s, want %s while submarine submerged",
				frame, plane.Flight.Phase, objUnit.PlaneFlightPhaseCruising)
		}
		if returnInstr.Executed() {
			t.Fatalf("frame %d: return finished while submarine submerged", frame)
		}
	}

	if _, ok := ms.Arena.Planes[plane.Uid]; !ok {
		t.Fatalf("holding plane left the arena")
	}
	if got := ship.Aircraft.Groups[0].CurCount; got != 0 {
		t.Fatalf("stock = %d, want 0 before recovery", got)
	}
	if distance := plane.CurPos.Distance(ship.CurPos); distance >= startDistance {
		t.Fatalf("holding plane distance = %.2f, want < start %.2f", distance, startDistance)
	}
	if plane.RemainRange >= startRange {
		t.Fatalf("holding plane range = %.2f, want < start %.2f", plane.RemainRange, startRange)
	}

	// 已经开始最后进近时艇又下潜：同样退回盘旋，不把飞机丢进无法降落的水面。
	plane.Flight.Phase = objUnit.PlaneFlightPhaseLandingApproach
	execPlaneReturn(t, returnInstr, ms)
	if plane.Flight.Phase != objUnit.PlaneFlightPhaseCruising {
		t.Fatalf("phase = %s, want %s after aborting final approach",
			plane.Flight.Phase, objUnit.PlaneFlightPhaseCruising)
	}

	ship.Submarine.DiveState = objUnit.DiveStateSurface
	execPlaneReturn(t, returnInstr, ms)
	if plane.Flight.Phase != objUnit.PlaneFlightPhaseLandingStaging {
		t.Fatalf("phase = %s, want %s after surfacing",
			plane.Flight.Phase, objUnit.PlaneFlightPhaseLandingStaging)
	}
}

// TestPlaneReturnDitchesWhenFuelRunsOutUnderWater 验证等不到上浮的返航机油尽即弃机。
// 场景：潜艇一直保持下潜，返航机在艇位附近盘旋到 RemainRange 归零，
// 艇侧水面始终不可用；此时应判弃机而不是凭空入库。
// 预期：Exec 后 CurHP=0、指令 Executed、库存 CurCount 仍为 0；飞机留在
// Arena.Planes 由管理端按坠毁流程清理。
// 覆盖：把返航机 RemainRange 置 0 并在下潜潜艇上执行一次 PlaneReturn.Exec，
// 断言血量、指令状态、库存与实体是否仍在空中集合。
func TestPlaneReturnDitchesWhenFuelRunsOutUnderWater(t *testing.T) {
	ms, ship, planes := newLandingTestState(t, "test-submerged-ditch", 1)
	plane := planes[0]
	ship.Type = objUnit.ShipTypeSubmarine
	ship.Submarine.DiveState = objUnit.DiveStateSubmerged
	plane.RemainRange = 0

	returnInstr := NewPlaneReturn(plane.Uid)
	execPlaneReturn(t, returnInstr, ms)

	if plane.CurHP > 0 {
		t.Fatalf("plane HP = %.1f, want 0 for ditched aircraft", plane.CurHP)
	}
	if !returnInstr.Executed() {
		t.Fatalf("return instruction not finished after ditching")
	}
	if got := ship.Aircraft.Groups[0].CurCount; got != 0 {
		t.Fatalf("stock = %d, want 0 (ditched plane must not be recovered)", got)
	}
	if _, ok := ms.Arena.Planes[plane.Uid]; !ok {
		t.Fatalf("ditched plane removed before manager crash handling")
	}
}

// TestPlaneReturnRecoversOnSurfacedSubmarine 验证浮航潜艇能完整走完着水回收。
// 场景：伊-400 一直浮在水面，晴岚投完鱼雷返航，使用水上飞机母舰的舷侧着水甲板；
// 若小尺寸艇体的进近圆弧或着水点算不出来，飞机会一直卡在待场/进近，永远回不了库。
// 预期：2500 拍内指令执行完成，飞机从 Arena.Planes 移除、库存 +1，且全程
// 经过 LandingDeck 阶段（不是在待场或进近阶段凭空消失）。
// 覆盖：ResolveDeck 绑定 sea 着陆配置后逐拍执行 PlaneReturn，断言 Executed、
// CurCount、实体移除与是否到过着水段。
func TestPlaneReturnRecoversOnSurfacedSubmarine(t *testing.T) {
	ms, ship, planes := newLandingTestState(t, "test-submarine-recover", 1)
	ship.Type = objUnit.ShipTypeSubmarine
	ship.Length = 122
	ship.Width = 12
	ship.Submarine.DiveState = objUnit.DiveStateSurface
	ship.Aircraft.ResolveDeck(&objUnit.CarrierDeck{
		Name: "submarine-sea",
		Landing: objUnit.LandingConfig{
			Mode:           objUnit.LandingModeSea,
			Forward:        0.5,
			Lateral:        1.1,
			ApproachLength: 3,
		},
	})

	plane := planes[0]
	plane.CurPos = objPos.NewR(53, 50)
	returnInstr := NewPlaneReturn(plane.Uid)

	reachedDeck := false
	for frame := 0; frame < 2500 && !returnInstr.Executed(); frame++ {
		execPlaneReturn(t, returnInstr, ms)
		if plane.Flight.Phase == objUnit.PlaneFlightPhaseLandingDeck {
			reachedDeck = true
		}
	}
	if !returnInstr.Executed() {
		t.Fatalf("return not finished in 2500 frames, phase = %s", plane.Flight.Phase)
	}
	if !reachedDeck {
		t.Fatalf("plane recovered without reaching the water landing run")
	}
	if got := ship.Aircraft.Groups[0].CurCount; got != 1 {
		t.Fatalf("recovered stock = %d, want 1", got)
	}
	if _, ok := ms.Arena.Planes[plane.Uid]; ok {
		t.Fatalf("plane still in arena after recovery")
	}
}
