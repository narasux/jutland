package unit

import (
	"testing"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// staticBase 测试用静态基地（模拟陆地机场的几何契约：速度恒 0、朝向恒定）。
type staticBase struct {
	uid      string
	pos      objPos.MapPos
	rotation float64
	// length / width 以地图块为单位
	length   float64
	width    float64
	aircraft ShipAircraft
}

func (b *staticBase) BaseUid() string                  { return b.uid }
func (b *staticBase) BasePos() objPos.MapPos           { return b.pos }
func (b *staticBase) BaseRotation() float64            { return b.rotation }
func (b *staticBase) BaseLength() float64              { return b.length * constants.MapBlockSize }
func (b *staticBase) BaseWidth() float64               { return b.width * constants.MapBlockSize }
func (b *staticBase) BaseSpeed() float64               { return 0 }
func (b *staticBase) BaseBelongPlayer() faction.Player { return faction.HumanAlpha }
func (b *staticBase) BaseAircraft() *ShipAircraft      { return &b.aircraft }

func TestIsBaseMissing(t *testing.T) {
	if !isBaseMissing(nil) {
		t.Fatal("nil interface should be missing")
	}
	if !isBaseMissing((*BattleShip)(nil)) {
		t.Fatal("nil carrier pointer should be missing")
	}
	ship := &BattleShip{Uid: "carrier"}
	if isBaseMissing(ship) {
		t.Fatal("existing carrier should not be missing")
	}
}

func TestTakeoffFromStaticBase(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	oldTemplate, hadTemplate := PlaneMap["F"]
	PlaneMap["F"] = &Plane{
		Name: "F", TotalHP: 100, CurHP: 100,
		MaxSpeed: 0.12, Acceleration: 0.01, RotateSpeed: 12, RemainRange: 100,
	}
	t.Cleanup(func() {
		if hadTemplate {
			PlaneMap["F"] = oldTemplate
		} else {
			delete(PlaneMap, "F")
		}
	})

	// 静态基地：跑道沿正北，长 6 格；起飞点 forward=1 即跑道起点（后端）
	base := &staticBase{
		uid: "af-1", pos: objPos.NewR(50, 50), rotation: 0, length: 6, width: 0.8,
		aircraft: ShipAircraft{Groups: []PlaneGroup{{Name: "F", MaxCount: 2, CurCount: 2}}},
	}
	plane := NewPlane("F", base.BasePos(), 0, base.BaseUid(), faction.HumanAlpha)
	plane.StartTakeoff(base, TakeoffPoint{Forward: 1, Lateral: 0, RunLength: 1})

	// 起飞点位于跑道后端：前后偏移 = 6*(0.5-1) = -3 格，rotation 0（朝北）时
	// 正 forward 指向 RY 减小方向，故后端位于中心以南 y = 50 + 3 = 53
	if plane.FlightPhase != PlaneFlightPhaseTakingOff {
		t.Fatalf("phase = %s, want %s", plane.FlightPhase, PlaneFlightPhaseTakingOff)
	}
	if plane.CurPos.RX < 49.99 || plane.CurPos.RX > 50.01 ||
		plane.CurPos.RY < 52.99 || plane.CurPos.RY > 53.01 {
		t.Fatalf("takeoff start pos = %s, want (50, 53)", plane.CurPos.String())
	}
	if plane.CurRotation != 0 {
		t.Fatalf("takeoff rotation = %.2f, want 0", plane.CurRotation)
	}

	// 推进至巡航：沿跑道滑跑后向北爬升，飞机应越过跑道前端（y < 47）
	for frame := 0; frame < 2000 && !plane.IsCruising(); frame++ {
		plane.UpdateTakeoff(nil, base)
	}
	if !plane.IsCruising() {
		t.Fatalf("phase = %s, want %s (pos=%s)",
			plane.FlightPhase, PlaneFlightPhaseCruising, plane.CurPos.String())
	}
	if plane.CurPos.RY >= 47 {
		t.Fatalf("plane should climb north of runway start, pos = %s", plane.CurPos.String())
	}
	if plane.CurSpeed <= 0 {
		t.Fatal("cruising plane should have positive speed")
	}
}

func TestStaticBaseLandingGeometry(t *testing.T) {
	// 静态基地的着陆几何应与航母一致：回收点在跑道中点，进近段起点在其后方
	base := &staticBase{
		uid: "af-1", pos: objPos.NewR(50, 50), rotation: 0, length: 6, width: 0.8,
		aircraft: ShipAircraft{
			Groups: []PlaneGroup{{Name: "F", MaxCount: 2, CurCount: 2}},
		},
	}
	base.aircraft.ResolveDeck(&CarrierDeck{
		Name:          "TestAirfieldDeck",
		TakeoffPoints: []TakeoffPoint{{Forward: 1, Lateral: 0, RunLength: 1}},
		Landing: LandingConfig{
			Mode: LandingModeDeck, Forward: 0.5, Lateral: 0,
			ApproachAngle: 0, ApproachLength: 3,
		},
	})

	landing := base.aircraft.landingConfig()
	end := carrierLandingDeckEndPos(base, landing)
	// 回收点 = 跑道中点 (50, 50)：forward 偏移 = 基准长*(0.5-0.5) = 0
	if end.RX < 49.99 || end.RX > 50.01 || end.RY < 49.99 || end.RY > 50.01 {
		t.Fatalf("landing end pos = %s, want (50, 50)", end.String())
	}
	start := landingFinalStartPos(base, landing)
	// 最终进近段起点在回收点后方 3 个基准长（基准长封顶 4 格 → 12 格）：y = 50 + 12 = 62
	if start.RY < 61.99 || start.RY > 62.01 {
		t.Fatalf("landing final start pos = %s, want y = 62", start.String())
	}
}

// TestLongRunwayPhaseGeometryStaysNearBase 回归：超长跑道（陆地机场）的起飞滑跑
// 与进近待场结构必须按封顶基准缩放，待场/进近点不得被放大到基地远端之外。
func TestLongRunwayPhaseGeometryStaysNearBase(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	oldTemplate, hadTemplate := PlaneMap["F"]
	PlaneMap["F"] = &Plane{
		Name: "F", TotalHP: 100, CurHP: 100,
		MaxSpeed: 0.12, Acceleration: 0.01, RotateSpeed: 12, RemainRange: 200,
	}
	t.Cleanup(func() {
		if hadTemplate {
			PlaneMap["F"] = oldTemplate
		} else {
			delete(PlaneMap, "F")
		}
	})

	// 与珍珠港关卡一致的 17 格跑道
	base := &staticBase{
		uid: "af-long", pos: objPos.NewR(55, 63), rotation: 45, length: 17, width: 0.8,
		aircraft: ShipAircraft{
			Groups: []PlaneGroup{{Name: "F", MaxCount: 2, CurCount: 2}},
		},
	}
	base.aircraft.ResolveDeck(&CarrierDeck{
		Name:          "LongRunwayDeck",
		TakeoffPoints: []TakeoffPoint{{Forward: 1, Lateral: 0, RunLength: 1}},
		Landing: LandingConfig{
			Mode: LandingModeDeck, Forward: 0.5, Lateral: 0,
			ApproachAngle: 0, ApproachLength: 3,
		},
	})

	// 起飞点仍在真实跑道端头（中心后方 8.5 格 → (48.99, 69.01)）
	plane := NewPlane("F", base.BasePos(), 45, base.BaseUid(), faction.HumanAlpha)
	plane.StartTakeoff(base, base.aircraft.deck.TakeoffPoints[0])
	if plane.CurPos.RX < 48.9 || plane.CurPos.RX > 49.1 ||
		plane.CurPos.RY < 68.9 || plane.CurPos.RY > 69.1 {
		t.Fatalf("takeoff start should sit on the real runway threshold, pos = %s", plane.CurPos.String())
	}
	// 滑跑距离按封顶基准（4 格）而非跑道全长
	if plane.takeoffRunLength != maxPhaseUnitLength {
		t.Fatalf("takeoff run length = %f, want %f", plane.takeoffRunLength, maxPhaseUnitLength)
	}
	// 推进至巡航：起飞全程应在 500 帧内完成（约 8 秒）
	for frame := 0; frame < 500 && !plane.IsCruising(); frame++ {
		plane.UpdateTakeoff(nil, base)
	}
	if !plane.IsCruising() {
		t.Fatalf("takeoff did not finish within 500 frames, phase = %s", plane.FlightPhase)
	}

	// 待场入口与最终进近段起点距基地中心不超过 30 格（5.5/3 个封顶基准长）
	length := phaseUnitInMapBlocks(base)
	gate := landingGateLocalOffset(0)
	gatePos := carrierRelativePos2D(base, length*gate.forward, length*gate.lateral)
	if gatePos.Distance(base.BasePos()) > 30 {
		t.Fatalf("staging gate too far from base: %f", gatePos.Distance(base.BasePos()))
	}
	start := landingFinalStartPos(base, base.aircraft.landingConfig())
	if start.Distance(base.BasePos()) > 30 {
		t.Fatalf("final approach start too far from base: %f", start.Distance(base.BasePos()))
	}
}
