package computer

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// 迷雾里舰队推进时，最近一次看见敌人的位置优先，别被未探索海面带走。
func TestSearchAdvancePrefersLiveContact(t *testing.T) {
	vision := state.NewFactionVision(24, 24)
	vision.Stamp(2, 2, 2)
	vision.CommitExplored()
	vision.Contacts = map[string]state.Contact{
		"stale": {RX: 20, RY: 20, ExpireTick: 0},
		"fresh": {RX: 9, RY: 4, ExpireTick: 100},
	}

	got, ok := searchAdvancePoint(vision, objPos.New(2, 2), 10)
	if !ok {
		t.Fatal("有未过期接触时应该给出推进点")
	}
	if !got.Near(objPos.New(9, 4), 0.01) {
		t.Fatalf("推进点 = %v, want 未过期的接触点 9,4", got)
	}
}

// fogBattlefield 造一张开了迷雾的小战场：自己舰在已探索海面，敌舰远在雾里。
func fogBattlefield(t *testing.T) (*state.MissionState, *ComputerDecisionHandler, *battleSnapshot) {
	t.Helper()
	vision := state.NewFactionVision(24, 24)
	// 半径 2 盖住 (2,2) 和 (3,2)，敌舰摆在 (3,2) 时刚好可见。
	vision.Stamp(2, 2, 2)
	vision.CommitExplored()
	own := makeShip("own", faction.ComputerAlpha, objUnit.ShipTypeFrigate, 2, 2)
	enemy := makeShip("foe", faction.HumanAlpha, objUnit.ShipTypeFrigate, 20, 20)
	ms := newMission(0, []*objUnit.BattleShip{own, enemy}, nil)
	ms.Core.FogOfWar = true
	ms.Player.Visions = map[faction.Player]*state.FactionVision{
		faction.ComputerAlpha: vision,
	}
	handler := NewHandler(faction.ComputerAlpha)
	return ms, handler, handler.snapshot(ms)
}

// 迷雾下没有可见目标时，攻击组必须转入搜索推进，而不是回锚点罚站。
func TestAttackOrderSearchesWhenNoEnemyVisible(t *testing.T) {
	_, handler, snap := fogBattlefield(t)
	handler.anchor = objPos.New(2, 2)

	order := handler.makeAttack(nil, snap, snap.own)
	if order == nil || !order.searching {
		t.Fatalf("order = %+v, want a searching attack order", order)
	}

	ship := snap.own[0]
	route, ok := handler.desiredRoute(ship, 0, order, routeAttack, snap)
	if !ok {
		t.Fatal("看不见敌人时攻击组应该主动推进")
	}
	if route.dest.Distance(ship.CurPos) < 1 {
		t.Fatalf("推进点 %v 就在脚下，等于没动", route.dest)
	}
}

// 有接触时直接扑最近一次看见敌人的位置。
func TestAttackOrderHeadsForContact(t *testing.T) {
	ms, handler, snap := fogBattlefield(t)
	handler.anchor = objPos.New(2, 2)
	ms.Player.Visions[faction.ComputerAlpha].Contacts = map[string]state.Contact{
		"stale": {RX: 20, RY: 20, ExpireTick: 0},
		"fresh": {RX: 18, RY: 6, ExpireTick: ms.Core.SimTick + 600},
	}

	order := handler.makeAttack(nil, snap, snap.own)
	route, ok := handler.desiredRoute(snap.own[0], 0, order, routeAttack, snap)
	if !ok {
		t.Fatal("有接触时攻击组应该扑向接触点")
	}
	if !route.dest.Near(objPos.New(18, 6), 0.01) {
		t.Fatalf("dest = %v, want the fresh contact at 18,6", route.dest)
	}
}

// 连接触都没有时，朝未探索海面的重心推，而不是各自找最近的前线散开。
func TestSearchAdvanceFallsBackToUnexploredCentroid(t *testing.T) {
	ms, _, _ := fogBattlefield(t)
	vision := ms.Player.Visions[faction.ComputerAlpha]

	want, ok := vision.UnexploredCentroid()
	if !ok {
		t.Fatal("测试地图应该还有未探索海面")
	}
	got, ok := searchAdvancePoint(vision, objPos.New(2, 2), ms.Core.SimTick)
	if !ok {
		t.Fatal("应该算得出推进点")
	}
	if !got.Near(want, 0.01) {
		t.Fatalf("推进点 = %v, want 未探索重心 %v", got, want)
	}

	// 过期的接触不算数。
	vision.Contacts = map[string]state.Contact{"stale": {RX: 20, RY: 20, ExpireTick: 0}}
	if got, _ := searchAdvancePoint(vision, objPos.New(2, 2), ms.Core.SimTick); !got.Near(want, 0.01) {
		t.Fatalf("推进点 = %v, 过期接触不该把舰队带偏", got)
	}

	// 推进点就在脚下时退回最近的前线格，避免舰队站住不动。
	if got, ok := searchAdvancePoint(vision, want, ms.Core.SimTick); !ok || got.Near(want, 0.01) {
		t.Fatalf("推进点 = %v, 站在重心上时应该换一个前线格", got)
	}
}

func TestSnapshotKeepsUnseenEnemyOut(t *testing.T) {
	ms, handler, snap := fogBattlefield(t)
	if len(snap.enemy) != 0 {
		t.Fatalf("snapshot enemy = %v, want none while the foe is unseen", snap.enemy)
	}
	if _, ok := snap.byUID["foe"]; ok {
		t.Fatal("an unseen enemy leaked into byUID")
	}

	ms.Arena.Ships["foe"].CurPos = objPos.New(3, 2)
	snap = handler.snapshot(ms)
	if len(snap.enemy) != 1 || snap.byUID["foe"] == nil {
		t.Fatalf("snapshot = %+v, want the foe once it is in sight", snap.enemy)
	}
}

// 看见了敌人就不再搜索推进，交回正常的阵位 / 追击逻辑。
func TestAttackOrderStopsSearchingWithVisibleEnemy(t *testing.T) {
	ms, handler, snap := fogBattlefield(t)
	ms.Arena.Ships["foe"].CurPos = objPos.New(3, 2)
	snap = handler.snapshot(ms)
	handler.anchor = objPos.New(2, 2)

	order := handler.makeAttack(nil, snap, snap.own)
	if order == nil || order.searching {
		t.Fatalf("order = %+v, want a normal attack order once the foe is visible", order)
	}
}
