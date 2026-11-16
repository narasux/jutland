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

// TestPlanePatrolCirclesBase 验证无目标的战斗机绕基地盘旋而不返航。
// 场景：fighter-patrol-test 型号（MaxSpeed 0.08、RotateSpeed 20、RemainRange 200）
// 位于航母 carrier (10,10) 东侧 6 格处，NewPlanePatrol 后推进 600 拍。
// 预期：指令不结束、ForceReturn=false；飞机至少一次进入距基地 3 格内、全程不
// 超过 8 格，绕基地方位角累计 ≥180°，RemainRange 相比初始值下降（空中盘旋耗油）。
// 失败意味着 CAP 会在无目标时直接返航或原地悬停。
// 覆盖：newPatrolTestState(t, 6) + NewPlanePatrol + 600 次 Exec，逐拍累计
// base.CurPos.Angle(plane.CurPos) 并检查距离与燃油。
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

// TestPlanePatrolEndsOnReturnConditions 验证 ForceReturn 与基地沉没结束巡逻。
// 场景：两段子场景——(1) 飞机 ForceReturn=true 后执行巡逻；(2) 从 Arena.Ships
// 删除 carrier 后执行巡逻。
// 预期：第一段立即 Executed；第二段同样 Executed 且 plane.ForceReturn=true，
// 把飞机交给 updatePlaneAttackOrReturn / PlaneReturn 按坠毁处理。
// 覆盖：newPatrolTestState(t, 6) + NewPlanePatrol 两次，分别断言 Executed 与
// ForceReturn。
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

// TestPlanePatrolCenterAdvancesTowardEnemyContact 验证有接触时巡逻圆心向敌前出。
// 场景：withPatrolIntel 把 SimTick 设为 100、地图 256x256，HumanAlpha 视野里
// 敌接触在基地 (10,10) 正东 80 格 (90,10)、ExpireTick=1000；战斗机在基地上，
// 推进 900 拍。
// 预期：飞机在基地朝东 patrolForwardOffset=10 格、半径 patrolRadius=2.5 的
// 圆上盘旋，距基地落在 10±2.5±1 即 6.5~13.5 格，且 RX>基地 RX（不会停在西侧）。
// 覆盖：newPatrolTestState(t,0) + withPatrolIntel + 900 次 Exec，断言最终距离
// 区间与东西方向。
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

// TestPlanePatrolCenterFallsBackToMapCenter 验证无接触时朝地图中心前出。
// 场景：withPatrolIntel 给 256x256 地图但不给任何接触，基地在 (10,10)、地图中心
// (128,128) 在其东南；推进 900 拍。
// 预期：巡逻圆心沿基地→地图中心方向前出 10 格，飞机距基地仍在 6.5~13.5 格，
// 且 RX>基地 RX；这样 CAP 面向双方相向展开的方向，而不是死守基地上空。
// 覆盖：newPatrolTestState(t,0) + withPatrolIntel(nil) + 900 次 Exec，断言距离
// 区间与 RX 方向。
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

// TestPlanePatrolCenterIgnoresExpiredContact 验证过期接触不参与巡逻前出方向。
// 场景：withPatrolIntel 把 SimTick 推到 100，接触点在 (250,10)、ExpireTick 在
// 调用前按初始 SimTick=10 记为 9；基地在 (10,10)，地图 256x256。
// 预期：patrolCenter 忽略该接触，改用地图中心 (128,128)，因基地在西北角，
// 返回圆心 RX>10 且 RY>10（东南方向）；若采用过期接触，圆心会朝正东。
// 覆盖：直接调用 NewPlanePatrol(...).patrolCenter(ms, plane, base)，断言坐标方向。
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
