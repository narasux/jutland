package instruction

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// newLoiterTestState 造一架带移动策略、停在侦察点上的侦察机，用于盘旋相关测试。
func newLoiterTestState(t *testing.T, point objPos.MapPos) (*state.MissionState, *objUnit.Plane) {
	t.Helper()
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = previous })

	const name = "scout-loiter-test"
	old, had := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name: name, Type: objUnit.PlaneTypeScout, TotalHP: 20, CurHP: 20,
		MaxSpeed: 0.1, RotateSpeed: 30, RemainRange: 80,
		SightRange: objUnit.SightRangeScout,
	}
	t.Cleanup(func() {
		if had {
			objUnit.PlaneMap[name] = old
			return
		}
		delete(objUnit.PlaneMap, name)
	})

	plane := objUnit.NewPlane(name, point, 0, "carrier", faction.HumanAlpha)
	ms := &state.MissionState{
		Core: state.MissionCoreState{SimTick: 10},
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}
	return ms, plane
}

func TestPlaneScoutLoitersThenReturns(t *testing.T) {
	point := objPos.New(3, 3)
	ms, plane := newLoiterTestState(t, point)
	order := NewPlaneScout(plane.Uid, point, true)
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if order.loiterUntil <= ms.Core.SimTick {
		t.Fatal("arrival should start a 30 second loiter")
	}
	ms.Core.SimTick = order.loiterUntil
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !plane.ForceReturn || !order.Executed() {
		t.Fatal("loiter should end in a return")
	}
}

// 到达侦察点后飞机应该绕着点位盘旋，而不是原地悬停。
func TestPlaneScoutCirclesInsteadOfHovering(t *testing.T) {
	point := objPos.New(3, 3)
	ms, plane := newLoiterTestState(t, point)
	order := NewPlaneScout(plane.Uid, point, true)
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}

	start := plane.CurPos.Copy()
	turned, moved := 0.0, false
	previous := point.Angle(plane.CurPos)
	for range 240 {
		ms.Core.SimTick++
		if err := order.Exec(ms); err != nil {
			t.Fatal(err)
		}
		bearing := point.Angle(plane.CurPos)
		turned += math.Mod(bearing-previous+540, 360) - 180
		previous = bearing
		if plane.CurPos.Distance(start) > 0.05 {
			moved = true
		}
		if distance := plane.CurPos.Distance(point); distance > 2*scoutLoiterRadius {
			t.Fatalf("盘旋时飞离侦察点过远: distance = %v", distance)
		}
	}
	if !moved {
		t.Fatal("到达侦察点后飞机应该在盘旋，而不是停在原地")
	}
	if turned < 180 {
		t.Fatalf("盘旋应该扫过一整圈方位角，实际累计 %v 度", turned)
	}
}

func TestTrackFleetHoldsOutsideAntiAir(t *testing.T) {
	vision := state.NewFactionVision(24, 24)
	vision.Stamp(10, 10, 12)
	ship := &objUnit.BattleShip{
		Uid: "enemy", CurHP: 10, BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.New(10, 10),
		Weapon: objUnit.ShipWeapon{MaxToPlaneRange: 6},
	}
	ms := &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
	}
	plane := &objUnit.Plane{
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(10, 16.5),
		SightRange:   36,
	}
	_, hold := trackFleet(ms, plane, vision)
	if !hold {
		t.Fatal("scout inside the anti-air margin should hold")
	}
	plane.CurPos = objPos.NewR(10, 22)
	move, hold := trackFleet(ms, plane, vision)
	if !move || hold {
		t.Fatal("scout outside anti-air should close on the fleet")
	}
}

// evadeTestState 造一架在 (3,3) 朝北巡航的侦察机，以及点在正东的敌机。
func evadeTestState(t *testing.T) (*state.MissionState, *objUnit.Plane, *objUnit.Plane) {
	t.Helper()
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = previous })

	const name = "scout-evade-test"
	old, had := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name: name, Type: objUnit.PlaneTypeScout, TotalHP: 20, CurHP: 20,
		MaxSpeed: 1, RotateSpeed: 45, RemainRange: 80,
		SightRange: objUnit.SightRangeScout,
	}
	t.Cleanup(func() {
		if had {
			objUnit.PlaneMap[name] = old
			return
		}
		delete(objUnit.PlaneMap, name)
	})

	plane := objUnit.NewPlane(name, objPos.NewR(3, 3), 0, "carrier", faction.HumanAlpha)
	// 敌机只作为威胁源，不需要移动策略。
	enemy := &objUnit.Plane{
		Uid: "enemy-fighter", Type: objUnit.PlaneTypeFighter, CurHP: 10, MaxSpeed: 1,
		CurPos: objPos.NewR(11, 3), BelongPlayer: faction.ComputerAlpha,
	}
	ms := &state.MissionState{
		Core: state.MissionCoreState{SimTick: 10},
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{plane.Uid: plane, enemy.Uid: enemy},
		},
	}
	return ms, plane, enemy
}

// 遇到敌机应该规避继续侦察，而不是当场放弃任务返航。
func TestPlaneScoutEvadesThreatInsteadOfReturning(t *testing.T) {
	ms, plane, _ := evadeTestState(t)
	order := NewPlaneScout(plane.Uid, objPos.New(3, 9), true)
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if plane.ForceReturn || order.Executed() {
		t.Fatal("遇敌应转入规避继续侦察，而不是返航")
	}
	// 敌机在正东，规避应把机头转向西半边。
	if plane.CurRotation <= 180 {
		t.Fatalf("应该朝背离敌机的方向转弯，rotation = %v", plane.CurRotation)
	}
}

// 威胁解除后应回到侦察航线继续飞向侦察点。
func TestPlaneScoutResumesMissionAfterThreatClears(t *testing.T) {
	ms, plane, enemy := evadeTestState(t)
	order := NewPlaneScout(plane.Uid, objPos.New(3, 9), true)
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !order.evading {
		t.Fatal("敌机在威胁距离内时应该处于规避状态")
	}

	// 敌机拉开到安全距离之外，侦察点在南边，机头应保持朝南继续任务。
	enemy.CurPos = objPos.NewR(3+objUnit.SightRangeScout, 3)
	plane.CurPos = objPos.NewR(3, 3)
	plane.CurRotation = 180
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if order.evading || plane.ForceReturn || order.Executed() {
		t.Fatal("威胁解除后应结束规避并继续侦察")
	}
	if plane.CurRotation != 180 {
		t.Fatalf("应该继续朝侦察点飞，rotation = %v, want 180", plane.CurRotation)
	}
}
