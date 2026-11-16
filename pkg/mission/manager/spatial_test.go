package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestCombatBucketsMatchDistanceFilter 验证均匀格子网的粗筛结果与半径 10 的距离筛选一致。
// 场景：以 (1,1) 为圆心查询，near(1,1) 与 mid(6,1) 在半径内，far(40,1) 远在半径外；
// combatIndex.rebuild 按 combatCellSize=8 把三舰放进格子，再调用 eachShip 遍历候选。
// 预期：回调里用 math.Hypot 复核后 got 必须包含 near 与 mid；far 距离 39 格，即使被粗筛扫到
// 也会被复核挡掉。失败意味着 bucketRings 少算或格子键冲突，射程内的船被漏掉，开火查询会漏目标。
// 覆盖：rebuild 三艘船后调用 eachShip(1,1,10)，用 got 映射记录通过距离复核的 UID 并断言集合。
func TestCombatBucketsMatchDistanceFilter(t *testing.T) {
	near := &objUnit.BattleShip{Uid: "near", CurPos: objPos.NewR(1, 1)}
	mid := &objUnit.BattleShip{Uid: "mid", CurPos: objPos.NewR(6, 1)}
	far := &objUnit.BattleShip{Uid: "far", CurPos: objPos.NewR(40, 1)}
	idx := &combatIndex{}
	idx.rebuild(map[string]*objUnit.BattleShip{
		near.Uid: near,
		mid.Uid:  mid,
		far.Uid:  far,
	}, nil)

	const radius = 10.0
	got := map[string]bool{}
	idx.eachShip(1, 1, radius, func(ship *objUnit.BattleShip) bool {
		if math.Hypot(ship.CurPos.RX-1, ship.CurPos.RY-1) <= radius {
			got[ship.Uid] = true
		}
		return false
	})
	if !got["near"] || !got["mid"] || got["far"] {
		t.Fatalf("ships in range = %v", got)
	}
}

// TestUnloadedShipDoesNotFire 验证唯一主炮仍在装填时整舰不会产出任何弹药。
// 场景：友方 "shooter" 有 1 门 AntiShip 主炮，ReloadTime=100 但 ReloadStartAt 设为 1<<62
// （永远没到装填完成时刻），目标 "target" 在 2 格外的射程 20 内、满血 100。
// 预期：updateShipWeaponFire 后 ForwardingBullets 为 0、target.CurHP 仍为 100；
// 失败意味着装填门槛失效，战舰会无视 CD 连续开火或在装填中造成伤害。
// 覆盖：config.G 兜底初始化，ShipWeapon{HasMainGun, MaxToShipRange=20} 与舰船映射，断言弹量与血量。
func TestUnloadedShipDoesNotFire(t *testing.T) {
	if config.G == nil {
		config.G = config.NewDefaultGameSettings()
	}
	shooter := &objUnit.BattleShip{
		Uid:          "shooter",
		BelongPlayer: faction.HumanAlpha,
		CurPos:       objPos.NewR(10, 10),
		Weapon: objUnit.ShipWeapon{
			HasMainGun: true,
			MainGuns: []*objUnit.Gun{{
				ReloadTime:    100,
				ReloadStartAt: 1 << 62,
				AntiShip:      true,
				Range:         20,
			}},
			MaxToShipRange: 20,
		},
	}
	target := &objUnit.BattleShip{
		Uid:          "target",
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(12, 10),
		TotalHP:      100,
		CurHP:        100,
	}
	manager := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{
				shooter.Uid: shooter,
				target.Uid:  target,
			},
		},
	}}
	manager.updateShipWeaponFire()
	if len(manager.state.Arena.ForwardingBullets) != 0 {
		t.Fatalf("bullets = %d, want 0 while the only gun is reloading", len(manager.state.Arena.ForwardingBullets))
	}
	if target.CurHP != 100 {
		t.Fatalf("target HP = %.1f, want 100", target.CurHP)
	}
}
