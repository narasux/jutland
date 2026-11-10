package computer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

func TestSnapshotSeparatesSubmarines(t *testing.T) {
	ownSub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	ownDD := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 6, 5)
	enemySub := makeShip("esub", faction.HumanAlpha, objUnit.ShipTypeSubmarine, 10, 5)
	enemySub.DiveState = objUnit.DiveStateSubmerged
	enemySub.MarkDetected(faction.ComputerAlpha, 100, enemySub.CurPos)
	enemyDD := makeShip("edd", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 11, 5)

	ms := newMission(50, []*objUnit.BattleShip{ownSub, ownDD, enemySub, enemyDD}, nil)
	snap := NewHandler(faction.ComputerAlpha).snapshot(ms)

	require.Len(t, snap.ownSubs, 1)
	require.Len(t, snap.own, 1, "水面编队不包含潜艇")
	require.Len(t, snap.enemySubs, 1)
	require.Len(t, snap.enemy, 1, "水面择敌不包含潜艇")
	require.Len(t, snap.subContacts, 1)
	require.True(t, snap.subContacts[0].live)
}

func TestSubmarineAIDivesOnEnemyAndFires(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Weapon.MaxToSubmergedRange = 6
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 15, 5)

	ms := newMission(0, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.Contains(t, out, instr.GenInstrUid(instr.NameShipDive, sub.Uid),
		"10 格内的敌舰应触发下潜接敌")
	require.Equal(t, enemy.Uid, sub.AttackTarget, "应把最近敌舰设为攻击目标")
}

func TestSubmarineAISurfacesWhenNoEnemy(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.DiveState = objUnit.DiveStateSubmerged

	ms := newMission(0, []*objUnit.BattleShip{sub}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.Contains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid),
		"无敌情时应回到水面高速巡航")
}

func TestSubmarineAIRetreatsWhenDetected(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.DiveState = objUnit.DiveStateSubmerged
	sub.MarkDetected(faction.HumanAlpha, 100, sub.CurPos)
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 9, 5)

	ms := newMission(10, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 10)

	require.NotEmpty(t, out, "被点亮时应下发撤离航路")
	require.Empty(t, sub.AttackTarget, "撤离时不再咬住目标")
}

func TestASWHunterLeavesFleetAndChasesContact(t *testing.T) {
	dd := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	dd.Weapon.HasAntiSubWeapon = true
	enemySub := makeShip("esub", faction.HumanAlpha, objUnit.ShipTypeSubmarine, 8, 0)
	enemySub.DiveState = objUnit.DiveStateSubmerged
	enemySub.MarkDetected(faction.ComputerAlpha, 1000, enemySub.CurPos)

	ms := newMission(10, []*objUnit.BattleShip{dd, enemySub}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)

	hunters := handler.selectASWHunters(snap, map[string]bool{})
	require.True(t, hunters[dd.Uid], "有深弹的驱逐舰应被派去猎潜")

	handler.asw = hunters
	out := handler.aswInstructions(snap)
	require.NotEmpty(t, out, "接触存在时应下发追击航路")
	require.Equal(t, enemySub.Uid, dd.AttackTarget)
}

func TestASWEngageDistanceFollowsWeaponRange(t *testing.T) {
	rackShip := &objUnit.BattleShip{}
	rackShip.Weapon.MaxToSubmergedRange = 2
	require.InDelta(t, 1.2, aswEngageDistance(rackShip), 1e-9)

	longRange := &objUnit.BattleShip{}
	longRange.Weapon.MaxToSubmergedRange = 8
	require.InDelta(t, 4.8, aswEngageDistance(longRange), 1e-9)

	noWeapon := &objUnit.BattleShip{}
	require.InDelta(t, aswFallbackEngageRange, aswEngageDistance(noWeapon), 1e-9)
}

// 深弹架只有 2 格：接触在 4 格时反潜舰必须继续压上，不能停在旧的距离上。
func TestASWHunterKeepsClosingWhenContactOutsideWeaponRange(t *testing.T) {
	sub := makeShip("esub", faction.HumanAlpha, objUnit.ShipTypeSubmarine, 4, 0)
	sub.DiveState = objUnit.DiveStateSubmerged
	sub.MarkDetected(faction.ComputerAlpha, 1000, sub.CurPos)

	dd := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	dd.Weapon.HasAntiSubWeapon = true
	dd.Weapon.MaxToSubmergedRange = 2

	ms := newMission(10, []*objUnit.BattleShip{dd, sub}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	handler.asw = map[string]bool{dd.Uid: true}

	out := handler.aswInstructions(snap)
	if len(out) == 0 {
		t.Fatal("contact at 4 tiles is outside the 2-tile depth charge range: hunter must keep closing")
	}
}
