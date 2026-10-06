package instruction

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// newPatrolTestState 造一艘停着的基地和一架在空巡航的战斗机，用于巡逻相关测试。
func newPatrolTestState(t *testing.T, offsetX float64) (*state.MissionState, *objUnit.Plane) {
	t.Helper()
	previous := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = previous })

	const name = "fighter-patrol-test"
	old, had := objUnit.PlaneMap[name]
	objUnit.PlaneMap[name] = &objUnit.Plane{
		Name: name, Type: objUnit.PlaneTypeFighter, TotalHP: 20, CurHP: 20,
		MaxSpeed: 0.08, RotateSpeed: 20, RemainRange: 200,
		Weapon: objUnit.PlaneWeapon{Guns: []*objUnit.Gun{{}}},
	}
	t.Cleanup(func() {
		if had {
			objUnit.PlaneMap[name] = old
			return
		}
		delete(objUnit.PlaneMap, name)
	})

	base := &objUnit.BattleShip{
		Uid: "carrier", CurHP: 100, CurPos: objPos.NewR(10, 10),
		BelongPlayer: faction.HumanAlpha,
	}
	plane := objUnit.NewPlane(name, objPos.NewR(10+offsetX, 10), 0, base.Uid, faction.HumanAlpha)
	ms := &state.MissionState{
		Core: state.MissionCoreState{SimTick: 10},
		Arena: state.MissionArenaState{
			Ships:  map[string]*objUnit.BattleShip{base.Uid: base},
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
		},
	}
	return ms, plane
}

// 没有目标的战斗机绕着基地盘旋等下一个目标，不返航。
func TestPlanePatrolCirclesBase(t *testing.T) {
	ms, plane := newPatrolTestState(t, 6)
	base := ms.Arena.Ships["carrier"]
	order := NewPlanePatrol(plane.Uid)

	startRange := plane.RemainRange
	turned, approached := 0.0, false
	previous := base.CurPos.Angle(plane.CurPos)
	for range 600 {
		ms.Core.SimTick++
		if err := order.Exec(ms); err != nil {
			t.Fatal(err)
		}
		bearing := base.CurPos.Angle(plane.CurPos)
		turned += math.Mod(bearing-previous+540, 360) - 180
		previous = bearing

		if distance := plane.CurPos.Distance(base.CurPos); distance < 3 {
			approached = true
		}
		if distance := plane.CurPos.Distance(base.CurPos); distance > 8 {
			t.Fatalf("巡逻时飞离基地过远: distance = %v", distance)
		}
	}
	if order.Executed() {
		t.Fatal("巡逻不该在没有返航条件时结束")
	}
	if plane.ForceReturn {
		t.Fatal("巡逻不等于返航")
	}
	if !approached {
		t.Fatal("战斗机应该飞回基地附近盘旋，而不是停在原地")
	}
	if turned < 180 {
		t.Fatalf("盘旋应该扫过一整圈方位角，实际累计 %v 度", turned)
	}
	if plane.RemainRange >= startRange {
		t.Fatal("空中盘旋应该消耗燃油")
	}
}

// 燃油耗尽或基地沉没后巡逻结束，交给 updatePlaneAttackOrReturn 换返航指令。
func TestPlanePatrolEndsOnReturnConditions(t *testing.T) {
	ms, plane := newPatrolTestState(t, 6)
	order := NewPlanePatrol(plane.Uid)

	plane.ForceReturn = true
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !order.Executed() {
		t.Fatal("必须返航的飞机不该继续巡逻")
	}

	// 基地沉没：巡逻结束并把飞机交给返航流程（与 PlaneReturn 一致按坠毁处理）。
	ms, plane = newPatrolTestState(t, 6)
	order = NewPlanePatrol(plane.Uid)
	delete(ms.Arena.Ships, "carrier")
	if err := order.Exec(ms); err != nil {
		t.Fatal(err)
	}
	if !order.Executed() || !plane.ForceReturn {
		t.Fatal("基地消失后巡逻应该结束并转为返航")
	}
}

// withPatrolIntel 给巡逻测试状态补上视野接触与地图尺寸，用于前出圆心相关断言。
func withPatrolIntel(
	ms *state.MissionState, contacts map[string]state.Contact, width, height int,
) {
	ms.Core.SimTick = 100
	ms.Core.MissionMD.MapCfg = &mapcfg.MapCfg{Width: width, Height: height}
	ms.Player.Visions = map[faction.Player]*state.FactionVision{
		faction.HumanAlpha: {Width: width, Height: height, Contacts: contacts},
	}
}

// 有敌舰接触时，巡逻圆心前出到基地朝敌一侧，而不是死守基地上空。
func TestPlanePatrolCenterAdvancesTowardEnemyContact(t *testing.T) {
	ms, plane := newPatrolTestState(t, 0)
	base := ms.Arena.Ships["carrier"]
	withPatrolIntel(ms, map[string]state.Contact{
		// 接触点在正东 80 格处：圆心应沿 +x 前出 patrolForwardOffset 格。
		"enemy": {RX: base.CurPos.RX + 80, RY: base.CurPos.RY, ExpireTick: 1000},
	}, 256, 256)
	order := NewPlanePatrol(plane.Uid)

	for range 900 {
		ms.Core.SimTick++
		if err := order.Exec(ms); err != nil {
			t.Fatal(err)
		}
	}

	distance := plane.CurPos.Distance(base.CurPos)
	if distance < patrolForwardOffset-patrolRadius-1 || distance > patrolForwardOffset+patrolRadius+1 {
		t.Fatalf("巡逻应停在基地朝敌方向 %v 格附近盘旋，实际距离 %v", patrolForwardOffset, distance)
	}
	if plane.CurPos.RX <= base.CurPos.RX {
		t.Fatalf("敌情在正东，巡逻不该停在基地西侧: pos = %v", plane.CurPos)
	}
}

// 没有敌舰接触时朝地图中心前出，让 CAP 面向双方相向展开的方向。
func TestPlanePatrolCenterFallsBackToMapCenter(t *testing.T) {
	ms, plane := newPatrolTestState(t, 0)
	base := ms.Arena.Ships["carrier"]
	withPatrolIntel(ms, nil, 256, 256)
	order := NewPlanePatrol(plane.Uid)

	for range 900 {
		ms.Core.SimTick++
		if err := order.Exec(ms); err != nil {
			t.Fatal(err)
		}
	}

	distance := plane.CurPos.Distance(base.CurPos)
	if distance < patrolForwardOffset-patrolRadius-1 || distance > patrolForwardOffset+patrolRadius+1 {
		t.Fatalf("无接触时应朝地图中心前出 %v 格盘旋，实际距离 %v", patrolForwardOffset, distance)
	}
	if plane.CurPos.RX <= base.CurPos.RX {
		t.Fatalf("地图中心在基地东侧，巡逻不该停在西侧: pos = %v", plane.CurPos)
	}
}

// 过期的敌舰接触不能作为前出方向。
func TestPlanePatrolCenterIgnoresExpiredContact(t *testing.T) {
	ms, plane := newPatrolTestState(t, 0)
	withPatrolIntel(ms, map[string]state.Contact{
		"enemy": {RX: 250, RY: 10, ExpireTick: ms.Core.SimTick - 1},
	}, 256, 256)

	center := NewPlanePatrol(plane.Uid).patrolCenter(ms, plane, ms.Arena.Ships["carrier"])
	// 过期接触被忽略：参考点退回地图中心 (128,128)，基地在 (10,10)，方向应指向东南。
	if center.RX <= 10 || center.RY <= 10 {
		t.Fatalf("过期接触不该参与前出方向: center = %v", center)
	}
}
