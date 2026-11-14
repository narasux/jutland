package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func depthChargeState(ships ...*objUnit.BattleShip) *state.MissionState {
	byUID := map[string]*objUnit.BattleShip{}
	for _, ship := range ships {
		byUID[ship.Uid] = ship
	}
	return &state.MissionState{
		Core:  state.MissionCoreState{SimTick: 100},
		Arena: state.MissionArenaState{Ships: byUID},
	}
}

func newDepthCharge(pos objPos.MapPos, player faction.Player) *objBullet.Bullet {
	return &objBullet.Bullet{
		Name: "US/DC/300/1940/Mk6", Type: objBullet.TypeDepthCharge,
		Damage: 1000, CurPos: pos, TargetPos: pos,
		BelongPlayer: player, BlastRadius: 1.3,
		HitObjType: 99, // 任意非 None，避免与"未命中"混淆
	}
}

func TestDepthChargeOnlyDamagesSubmarines(t *testing.T) {
	submerged := &objUnit.BattleShip{
		Uid: "sub1", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		DiveState: objUnit.DiveStateSubmerged,
	}
	surfaced := &objUnit.BattleShip{
		Uid: "sub2", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.5, 10),
	}
	destroyer := &objUnit.BattleShip{
		Uid: "dd", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10.8),
	}
	ms := depthChargeState(submerged, surfaced, destroyer)
	submerged.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, submerged.CurPos)

	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()
	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	m.resolveDepthChargeDamage(bt)

	require.Less(t, submerged.CurHP, 100.0, "被点亮的下潜潜艇应受伤")
	require.Less(t, surfaced.CurHP, 100.0, "浮航潜艇按水面规则受伤")
	require.Equal(t, 100.0, destroyer.CurHP, "深弹不伤水面舰")
}

func TestDepthChargeBlindFireDoesNotDamage(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		DiveState: objUnit.DiveStateSubmerged,
	}
	ms := depthChargeState(sub)
	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()

	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	m.resolveDepthChargeDamage(bt)

	require.Equal(t, 100.0, sub.CurHP, "未被探测时盲炸不结算")
}

func TestDepthChargeShouldExplodeTriggers(t *testing.T) {
	ms := depthChargeState()
	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()

	// 到瞄准点附近
	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	bt.ProximityRadius = 1.0
	bt.TargetPos = objPos.NewR(10, 10)
	require.True(t, m.depthChargeShouldExplode(bt))

	// 还没到：距瞄准点 3 格，朝目标方向飞行
	bt.CurPos = objPos.NewR(7, 10)
	bt.Rotation = 90
	bt.TargetPos = objPos.NewR(10, 10)
	bt.Life = 100
	require.False(t, m.depthChargeShouldExplode(bt))

	// 越过瞄准点
	bt.CurPos = objPos.NewR(11, 10)
	require.True(t, m.depthChargeShouldExplode(bt))

	// 寿命耗尽
	bt.CurPos = objPos.NewR(7, 10)
	bt.Life = 0
	require.True(t, m.depthChargeShouldExplode(bt))
}

func TestDepthChargePipelineThroughShotBullets(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 3000, TotalHP: 3000, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		DiveState: objUnit.DiveStateSubmerged,
	}
	sub.MarkDetected(faction.HumanAlpha, 500, sub.CurPos)
	ms := depthChargeState(sub)
	ms.Core.SimTick = 100

	bt := &objBullet.Bullet{
		Name: "US/DC/300/1940/Mk6", Type: objBullet.TypeDepthCharge,
		Damage: 1000, CurPos: objPos.NewR(10, 10), TargetPos: objPos.NewR(10, 10),
		ProximityRadius: 1.0, BlastRadius: 1.3, Speed: 0.03, Life: 40,
		Shooter: "dd", ShooterObjType: object.TypeShip,
		BelongPlayer: faction.HumanAlpha, TargetObjType: object.TypeShip,
	}
	ms.Arena.ForwardingBullets = []*objBullet.Bullet{bt}

	m := &MissionManager{state: ms}
	m.updateShotBullets()

	require.Less(t, sub.CurHP, 3000.0, "深弹应通过通用弹药循环结算到潜艇")
	require.Empty(t, ms.Arena.ForwardingBullets, "深弹引爆后不应继续飞行")
}

func TestDepthChargeRackCanFireAhead(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	shooter := &objUnit.BattleShip{
		Uid: "dd", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.HumanAlpha, CurPos: objPos.NewR(10, 10),
		CurRotation: 90, MaxSpeed: 1,
	}
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(11.5, 10),
		DiveState: objUnit.DiveStateSubmerged,
	}
	launcher := objUnit.NewRocketLauncher(
		"UK/DC/5/MkVII", -0.86,
		objUnit.FiringArc{Start: 0, End: 180},
		objUnit.FiringArc{Start: 180, End: 360},
	)
	bullets := launcher.Fire(shooter, sub)
	require.Len(t, bullets, 1, "舰艏正对目标时深弹架也应能投掷")
	require.Equal(t, objBullet.TypeDepthCharge, bullets[0].Type)
}

// TestDepthChargeSinkRecordsShooter 深弹击沉潜艇时必须记录投弹方与武器，
// 否则战报只剩「沉没 潜艇 X」，看不出是谁击沉的。
func TestDepthChargeSinkRecordsShooter(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", Name: "nautilus", CurHP: 50, TotalHP: 100,
		Type:         objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
	}
	ms := depthChargeState(sub)
	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()

	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	bt.Shooter = "dd"
	bt.ShooterName = "gurkha"
	bt.ShooterType = string(objUnit.ShipTypeDestroyer)
	bt.ShooterObjType = object.TypeShip
	m.resolveDepthChargeDamage(bt)

	require.Zero(t, sub.CurHP, "深弹应击沉潜艇")
	require.True(t, sub.SunkByHit)
	require.Equal(t, "gurkha", sub.LastAttackerName)
	require.Equal(t, string(objUnit.ShipTypeDestroyer), sub.LastAttackerType)
	require.Equal(t, faction.HumanAlpha, sub.LastAttackerPlayer)
	require.Equal(t, string(objBullet.TypeDepthCharge), sub.LastHit.WeaponType)
	require.Positive(t, sub.DamageTaken)

	// 击沉记录必须带上投弹方和武器，详情页才能渲染「驱逐舰 X 使用 深水炸弹 击沉 潜艇 Y」。
	ms.Stats.RecordShipLost(sub)
	require.Len(t, ms.Stats.BattleLog, 1)
	sink := ms.Stats.BattleLog[0]
	require.Equal(t, state.BattleLogShipSunk, sink.Action)
	require.Equal(t, "gurkha", sink.ActorName)
	require.Equal(t, string(objBullet.TypeDepthCharge), sink.Hit.WeaponType)
}
