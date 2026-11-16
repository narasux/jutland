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

// TestPlaneScoutLoitersThenReturns 验证手动侦察到达点位后盘旋 30 秒再返航。
// 场景：侦察机已停在侦察点 (3,3)，第 10 拍下达 NewPlaneScout(..., manual=true)，
// 到达即开始计时，loiterUntil = SimTick + ReloadTicks(30)。
// 预期：首拍 loiterUntil 大于当前 SimTick；把 SimTick 推到 loiterUntil 再执行，
// plane.ForceReturn=true 且指令 Executed。盘旋时间没生效会让飞机到点立刻掉头。
// 覆盖：newLoiterTestState 注册 scout-loiter-test（视距 SightRangeScout），断言两个状态。
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

// TestPlaneScoutCirclesInsteadOfHovering 验证侦察机到达点位后绕点盘旋而非悬停。
// 场景：newLoiterTestState 把 scout-loiter-test 放在侦察点 (3,3)，第 10 拍执行
// NewPlaneScout(..., manual=true)；到达判定 scoutArriveRadius=1.5 格、
// 盘旋半径 scoutLoiterRadius=1.0 格，连续推进 240 拍。
// 预期：飞机离起点 >0.05 格，绕点方位角累计 ≥180°，且距侦察点始终不超过
// 2*scoutLoiterRadius=2 格；失败意味着到点后直飞圆心、原地悬停。
// 覆盖：newLoiterTestState + orbitAt 的切向/径向合成，逐拍累计
// point.Angle(plane.CurPos) 并记录位移。
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

// TestTrackFleetHoldsOutsideAntiAir 验证侦察机在敌舰防空圈外 2 格安全边处保持距离。
// 场景：24x24 视野图上敌舰在 (10,10)、对空射程 6 格；侦察机先在 (10,16.5)
// 即 6.5 格处，再退到 (10,22) 即 12 格处。
// 预期：6.5 格落在 6 加 2 格边内，trackFleet 返回 hold=true；12 格时返回
// move=true、hold=false。失败意味着侦察机会贴进防空火力或被挡在过远处。
// 覆盖：直接调用 trackFleet，敌舰只填 CurHP/BelongPlayer/CurPos/MaxToPlaneRange。
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
	_, hold := trackFleet(ms, plane, vision, 0)
	if !hold {
		t.Fatal("scout inside the anti-air margin should hold")
	}
	plane.CurPos = objPos.NewR(10, 22)
	move, hold := trackFleet(ms, plane, vision, 0)
	if !move || hold {
		t.Fatal("scout outside anti-air should close on the fleet")
	}
}

// TestPlaneScoutCirclesWhileHoldingNearFleet 验证贴防空圈保持距离时仍绕圈观察。
// 场景：64x64 视野图点亮 (12,30) 半径 20；敌舰 fleet 在 (12,30)、对空射程 6 格；
// scout-hold-test 侦察机在 (12,38) 即 8 格处，手动侦察目标 (12,50)，连续推进 240 拍。
// 预期：首拍 order.holding=true；其后飞机绕 holdPoint 移动 >0.5 格，离保持点
// 不超过 2*scoutLoiterRadius=2 格，且与敌舰距离始终 >MaxToPlaneRange=6 格。
// 失败意味着 hold 分支直接 return 不发移动，敌我侦察机会停在舰队旁不动。
// 覆盖：注册 scout-hold-test 模板（SightRangeScout）+ NewPlaneScout + 逐拍 Exec，
// 检查 holding、位移、盘旋半径与防空距离。
func TestPlaneScoutCirclesWhileHoldingNearFleet(t *testing.T) {
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = previous })

	const name = "scout-hold-test"
	old, had := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name: name, Type: objUnit.PlaneTypeScout, TotalHP: 20, CurHP: 20,
		MaxSpeed: 0.1, RotateSpeed: 30, RemainRange: 400,
		SightRange: objUnit.SightRangeScout,
	}
	t.Cleanup(func() {
		if had {
			objUnit.PlaneMap[name] = old
			return
		}
		delete(objUnit.PlaneMap, name)
	})

	ship := &objUnit.BattleShip{
		Uid: "fleet", CurHP: 100, BelongPlayer: faction.ComputerAlpha,
		CurPos: objPos.New(12, 30),
		Weapon: objUnit.ShipWeapon{MaxToPlaneRange: 6},
	}
	vision := state.NewFactionVision(64, 64)
	vision.Stamp(ship.CurPos.RX, ship.CurPos.RY, 20)
	// 侦察机已经在防空圈（6 格）加 2 格安全边之外，不能再往里飞。
	plane := objUnit.NewPlane(name, objPos.NewR(12, 38), 0, "carrier", faction.HumanAlpha)
	ms := &state.MissionState{
		Core: state.MissionCoreState{SimTick: 10},
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
			Ships:  map[string]*objUnit.BattleShip{ship.Uid: ship},
		},
		Player: state.MissionPlayerState{
			Visions: map[faction.Player]*state.FactionVision{faction.HumanAlpha: vision},
		},
	}
	order := NewPlaneScout(plane.Uid, objPos.New(12, 50), false)
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !order.holding {
		t.Fatal("贴到防空圈外沿时应该转入保持距离")
	}

	anchor := plane.CurPos.Copy()
	moved := false
	for range 240 {
		ms.Core.SimTick++
		if err := order.Exec(ms); err != nil {
			t.Fatal(err)
		}
		anchorDistance := plane.CurPos.Distance(anchor)
		if anchorDistance > 0.5 {
			moved = true
		}
		if anchorDistance > 2*scoutLoiterRadius {
			t.Fatalf("保持距离盘旋时不该飞离接触点过远: distance = %v", anchorDistance)
		}
		if distance := plane.CurPos.Distance(ship.CurPos); distance <= ship.Weapon.MaxToPlaneRange {
			t.Fatalf("保持距离时不能钻进防空圈: distance = %v", distance)
		}
	}
	if !moved {
		t.Fatal("贴到防空圈外沿后飞机应该绕圈保持观察，而不是原地悬停")
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

// TestPlaneScoutEvadesThreatInsteadOfReturning 验证遇敌机时规避并继续侦察而非返航。
// 场景：evadeTestState 中侦察机在 (3,3) 朝北，敌战斗机 enemy-fighter 在正东
// (11,3) 相距 8 格（scoutThreatRange=36/2=18 格内且正在接近）；侦察目标 (3,9)。
// 预期：执行后 plane.ForceReturn=false、order.Executed()=false，机头转向背离敌机
// 一侧（CurRotation>180，向西/南）；提前返航会把慢速侦察机送进进近航线被击落。
// 覆盖：newPlane + NewPlaneScout + 单次 Exec，断言返航/完成状态与 CurRotation。
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

// TestPlaneScoutResumesMissionAfterThreatClears 验证威胁解除后恢复侦察航线。
// 场景：evadeTestState 首次 Exec 后 order.evading=true；把敌机拉到
// (3+SightRangeScout,3) 即 36 格外（scoutSafeRange=36*3/4=27 格之外），侦察机
// 复位到 (3,3)、航向 180°，目标仍是南侧 (3,9)。
// 预期：再次 Exec 后 evading=false、不返航、指令未完成，CurRotation 保持 180°
// 继续朝侦察点飞。
// 覆盖：evadeTestState + 两次 Exec，断言 evading/ForceReturn/Executed 与航向。
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
