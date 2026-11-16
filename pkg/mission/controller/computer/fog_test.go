package computer

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestSearchAdvancePrefersLiveContact 验证迷雾搜索推进优先选未过期接触。
// 场景：24x24 FactionVision 只 Stamp(2,2,2) 并 CommitExplored，Contacts 里
// "stale" 在 (20,20)、ExpireTick=0，"fresh" 在 (9,4)、ExpireTick=100；第 10 拍从
// 舰队位置 (2,2) 求推进点。
// 预期：searchAdvancePoint 返回 ok，且推进点为 (9,4) 而不是已过期的 (20,20)；
// 失败会把舰队带向早已失效的旧接触。
// 覆盖：直接调用 searchAdvancePoint，用 got.Near(objPos.New(9,4), 0.01) 断言。
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

// TestAttackOrderSearchesWhenNoEnemyVisible 验证迷雾下无可见敌舰时攻击组转入搜索推进。
// 场景：fogBattlefield 造 24x24 视野图，只点亮 (2,2) 半径 2；己方护卫舰在 (2,2)，
// 敌舰 foe 在 (20,20) 的雾里，FogOfWar=true，锚点固定 (2,2)。
// 预期：makeAttack 返回 order.searching=true；desiredRoute(routeAttack) 返回 ok，
// 且推进点距舰船 ≥1 格（不会回锚点罚站）。
// 覆盖：config.DefaultAIStrategies() 下调用 makeAttack / desiredRoute，检查 searching
// 与 route.dest 距离。
func TestAttackOrderSearchesWhenNoEnemyVisible(t *testing.T) {
	_, handler, snap := fogBattlefield(t)
	handler.anchor = objPos.New(2, 2)

	order := handler.makeAttack(nil, snap, snap.own, config.DefaultAIStrategies())
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

// TestAttackOrderHeadsForContact 验证有接触时攻击组直扑最近一次看见敌人的位置。
// 场景：fogBattlefield 中注入两个接触："stale" 在 (20,20)、ExpireTick=0；
// "fresh" 在 (18,6)、ExpireTick=SimTick+600；锚点 (2,2)。
// 预期：desiredRoute(routeAttack) 的 dest 为 (18,6)，与过期接触无关。
// 覆盖：makeAttack 后调用 desiredRoute，用 route.dest.Near((18,6), 0.01) 断言。
func TestAttackOrderHeadsForContact(t *testing.T) {
	ms, handler, snap := fogBattlefield(t)
	handler.anchor = objPos.New(2, 2)
	ms.Player.Visions[faction.ComputerAlpha].Contacts = map[string]state.Contact{
		"stale": {RX: 20, RY: 20, ExpireTick: 0},
		"fresh": {RX: 18, RY: 6, ExpireTick: ms.Core.SimTick + 600},
	}

	order := handler.makeAttack(nil, snap, snap.own, config.DefaultAIStrategies())
	route, ok := handler.desiredRoute(snap.own[0], 0, order, routeAttack, snap)
	if !ok {
		t.Fatal("有接触时攻击组应该扑向接触点")
	}
	if !route.dest.Near(objPos.New(18, 6), 0.01) {
		t.Fatalf("dest = %v, want the fresh contact at 18,6", route.dest)
	}
}

// TestSearchAdvanceFallsBackToUnexploredCentroid 验证无接触时朝未探索海面重心推进。
// 场景：fogBattlefield 的 24x24 视野图仍有未探索格，舰队在 (2,2)；先无接触求点，
// 再注入过期接触，最后把起点放在重心本身上。
// 预期：前两次都返回 vision.UnexploredCentroid()（过期接触不参与）；站在重心时
// 退到最近的前线格，避免原地不动。
// 覆盖：直接调用 searchAdvancePoint，断言与 UnexploredCentroid 的 Near 关系及
// 重心起点时返回值不同。
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

// TestSnapshotKeepsUnseenEnemyOut 验证迷雾下 AI 快照不包含看不见的敌舰。
// 场景：24x24 视野图只点亮以 (2,2) 为圆心、半径 2 的区域，敌舰 foe 在 (20,20)；
// 随后把 foe 挪进 (3,2)，正好落进刚才点亮的区域。
// 预期：雾里的 foe 既不在 snap.enemy 也不在 snap.byUID；挪进可见区后两者都出现。
// 覆盖：fogBattlefield 打开 FogOfWar、只给 ComputerAlpha 建视野，前后两次 snapshot 对比。
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

// TestAttackOrderStopsSearchingWithVisibleEnemy 验证看见敌人后退出搜索推进。
// 场景：fogBattlefield 中把敌舰 foe 从 (20,20) 挪到 (3,2)，正好落在 (2,2) 半径 2
// 的已点亮区域内；重新 snapshot 后锚点仍为 (2,2)。
// 预期：makeAttack 返回的 order.searching=false，交回正常阵位 / 追击逻辑；
// 失败意味着有可见目标仍整队向未探索海面推进。
// 覆盖：snapshot 前后对比，断言 order 非 nil 且 searching 为 false。
func TestAttackOrderStopsSearchingWithVisibleEnemy(t *testing.T) {
	ms, handler, snap := fogBattlefield(t)
	ms.Arena.Ships["foe"].CurPos = objPos.New(3, 2)
	snap = handler.snapshot(ms)
	handler.anchor = objPos.New(2, 2)

	order := handler.makeAttack(nil, snap, snap.own, config.DefaultAIStrategies())
	if order == nil || order.searching {
		t.Fatalf("order = %+v, want a normal attack order once the foe is visible", order)
	}
}
