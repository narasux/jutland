package computer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestSnapshotSeparatesSubmarines 验证潜艇不进水面编队与水面择敌，走独立潜艇通道。
// 场景：己方 1 艘潜艇在 (5,5)、1 艘驱逐舰在 (6,5)，敌方 1 艘已下潜并被点亮的
// 潜艇在 (10,5)、1 艘驱逐舰在 (11,5)，第 50 拍做快照。
// 预期：own/ownSubs、enemy/enemySubs 各 1 条，subContacts 1 条且 live=true。
// 覆盖：enemySub 调 MarkDetected(ComputerAlpha, 100)，断言四组长度与接触状态。
func TestSnapshotSeparatesSubmarines(t *testing.T) {
	ownSub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	ownDD := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 6, 5)
	enemySub := makeShip("esub", faction.HumanAlpha, objUnit.ShipTypeSubmarine, 10, 5)
	enemySub.Submarine.DiveState = objUnit.DiveStateSubmerged
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

// TestSubmarineAIDivesOnEnemyAndFires 验证 AI 潜艇在 16 格接敌圈内下潜并锁定目标。
// 场景：潜艇 Submarine.DiveState 默认 0（水面）在 (5,5)，鱼雷对潜射程 6 格，敌驱逐舰在
// (15,5) 距离 10 格，落在 subEngageRadius 内、还在 0.9 倍鱼雷射程 5.4 格外。
// 预期：下发 ShipDive，AttackTarget 设为最近的 enemy，并补一条压上的航路。
// 覆盖：submarineInstructions 返回值断言含下潜指令键，并检查 sub.AttackTarget。
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

// TestSubmarineAIStaysSubmergedWhenNoEnemy 验证无敌情时 AI 潜艇维持水下且不空转。
// 场景：潜艇已是 DiveStateSubmerged、带鱼雷，场上没有任何敌舰。
// 预期：本拍不产生任何指令，尤其不能出现 ShipSurface；没有目标就原地待机。
// 覆盖：snapshot 后调用 submarineInstructions，断言 out 为空且不含上浮指令键。
func TestSubmarineAIStaysSubmergedWhenNoEnemy(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
	sub.Weapon.HasTorpedo = true

	ms := newMission(0, []*objUnit.BattleShip{sub}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.Empty(t, out, "无敌情时没有可执行指令")
	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid),
		"无敌情时也应保持水下，不应为了赶路而上浮")
}

// TestSubmarineAIStaysSubmergedWhileApproaching 验证 AI 潜艇远距离转场时保持水下。
// 场景：ComputerAlpha 潜艇下潜于 (5,5)，带鱼雷、对潜射程 6 格；敌驱逐舰在 (40,5)，
// 相距 35 格，远在 subEngageRadius=16 格的接敌圈之外。
// 预期：submarineInstructions 不产生 ShipSurface（AI 不为赶路上浮）也不产生
// ShipDive（已在水下），但仍下发非空航路继续向敌舰靠拢。上浮只保留给玩家手动操作。
// 覆盖：第 0 拍 snapshot 后调用 submarineInstructions，用 require.NotContains /
// NotEmpty 检查 NameShipSurface、NameShipDive 与指令集。
func TestSubmarineAIStaysSubmergedWhileApproaching(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
	sub.Weapon.HasTorpedo = true
	sub.Weapon.MaxToSubmergedRange = 6
	// 35 格：远在接敌圈之外，属于水下巡航阶段
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 40, 5)

	ms := newMission(0, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid),
		"AI 潜艇不应为了赶路上浮")
	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipDive, sub.Uid),
		"已经在水下时不需要重复下潜")
	require.NotEmpty(t, out, "应继续向敌舰靠拢")
}

// TestSubmarineAIDivesWhenSurfacedWithDistantEnemy 验证水面开局的 AI 潜艇立即回到水下。
// 场景：潜艇 DiveStateSurface 位于 (5,5)，带鱼雷；敌驱逐舰在 (26,5)，距离 21 格，
// 仍在 16 格接敌圈之外，属于水下巡航阶段。
// 预期：本拍出现 ShipDive，且不出现 ShipSurface；水面状态不能作为 AI 潜艇的转场姿态。
// 覆盖：snapshot 后调用 submarineInstructions，断言 NameShipDive 存在、
// NameShipSurface 缺席（下潜由 NextDiveState 触发）。
func TestSubmarineAIDivesWhenSurfacedWithDistantEnemy(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSurface
	sub.Weapon.HasTorpedo = true
	// 21 格：仍在水下巡航阶段
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 26, 5)

	ms := newMission(0, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.Contains(t, out, instr.GenInstrUid(instr.NameShipDive, sub.Uid),
		"AI 潜艇在水面时应立即下潜")
	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid))
}

// TestSubmarineAIGunOnlyStaysSubmerged 验证只有甲板炮的 AI 潜艇也不上浮。
// 场景：潜艇已下潜于 (5,5)，Weapon.HasMainGun=true、对舰射程 12 格；敌驱逐舰在
// (15,5)，距离 10 格，已在甲板炮射程内。
// 预期：submarineInstructions 不下发 ShipSurface——甲板炮在水下不可用，AI 潜艇
// 不应为了开炮破坏隐蔽。
// 覆盖：第 0 拍 snapshot 后调用 submarineInstructions，断言指令集不含
// NameShipSurface 的 GenInstrUid。
func TestSubmarineAIGunOnlyStaysSubmerged(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
	sub.Weapon.HasMainGun = true
	sub.Weapon.MaxToShipRange = 12
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 15, 5)

	ms := newMission(0, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid),
		"AI 潜艇永远保持水下，不为甲板炮上浮")
}

// TestSubmarineAIRetreatsWhenDetected 验证被点亮的潜艇放弃攻击、保持深度撤离。
// 场景：潜艇在 (5,5) 已被 HumanAlpha 点亮到第 100 拍，敌驱逐舰在 (9,5) 距离 4 格，
// 第 10 拍决策时 detected 分支优先于接敌分支。
// 预期：下发一条背向敌舰 12 格的撤离航路，AttackTarget 保持为空；
// 仍在攻击说明潜艇会顶着声呐继续咬人。
// 覆盖：MarkDetected 后 snapshot，断言指令非空且 AttackTarget 为空。
func TestSubmarineAIRetreatsWhenDetected(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
	sub.MarkDetected(faction.HumanAlpha, 100, sub.CurPos)
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 9, 5)

	ms := newMission(10, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 10)

	require.NotEmpty(t, out, "被点亮时应下发撤离航路")
	require.Empty(t, sub.AttackTarget, "撤离时不再咬住目标")
}

// TestASWHunterLeavesFleetAndChasesContact 验证反潜舰离开水面编队专职追击接触。
// 场景：带深弹的驱逐舰 dd 在 (0,0)，8 格外有 1 艘被点亮到第 1000 拍的敌潜艇，
// 第 10 拍先挑猎手再下发反潜航路。
// 预期：selectASWHunters 选中 dd；aswInstructions 产出非空航路，并把
// dd.AttackTarget 设为敌潜艇 uid。失败意味着反潜舰会留在编队里不响应接触。
// 覆盖：Weapon.HasAntiSubWeapon=true，used 传空表，断言 hunters 与指令集。
func TestASWHunterLeavesFleetAndChasesContact(t *testing.T) {
	dd := makeShip("dd", faction.ComputerAlpha, objUnit.ShipTypeDestroyer, 0, 0)
	dd.Weapon.HasAntiSubWeapon = true
	enemySub := makeShip("esub", faction.HumanAlpha, objUnit.ShipTypeSubmarine, 8, 0)
	enemySub.Submarine.DiveState = objUnit.DiveStateSubmerged
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

// TestASWEngageDistanceFollowsWeaponRange 验证反潜贴近距离随自身反潜射程缩放。
// 场景：对潜射程 2 格、对潜射程 8 格、未配置射程三种舰体。
// 预期：aswEngageDistance 依次为 1.2、4.8、aswFallbackEngageRange=4.0；
// 固定值会让 2 格深弹架永远停在射程外。
// 覆盖：直接构造 BattleShip 只设 Weapon.MaxToSubmergedRange，用 InDelta 比较。
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

// TestASWHunterKeepsClosingWhenContactOutsideWeaponRange 验证 2 格深弹架的反潜舰继续压上。
// 场景：HumanAlpha 潜艇在 (4,0) 下潜并被点亮到第 1000 拍；ComputerAlpha 驱逐舰 dd
// 在 (0,0) 带反潜武器，MaxToSubmergedRange=2，aswEngageDistance 只有
// max(1.0, 2*0.6)=1.2 格。
// 预期：接触在 4 格外时 aswInstructions 必须下发航路继续接近；若沿用固定 4 格
// 贴近距离，深弹架会永远停在射程外。
// 覆盖：第 10 拍 snapshot，手动把 handler.asw 设为 {dd:true}，断言 aswInstructions
// 返回非空；失败信息对应“停在旧距离”的回归。
func TestASWHunterKeepsClosingWhenContactOutsideWeaponRange(t *testing.T) {
	sub := makeShip("esub", faction.HumanAlpha, objUnit.ShipTypeSubmarine, 4, 0)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
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

// TestSubmarineAIGunOnlyStaysSubmergedOutOfGunRange 验证只有甲板炮的潜艇在射程外不上浮。
// 场景：潜艇已下潜、带主炮，对舰射程 5 格；敌驱逐舰在 (15,5) 距离 10 格，
// 既在 16 格接敌圈内又在炮口 4.5 格之外。
// 预期：潜艇继续水下压上，但不得出现 ShipSurface——水下开不了甲板炮。
// 覆盖：snapshot 后调用 submarineInstructions，断言不含上浮指令键。
func TestSubmarineAIGunOnlyStaysSubmergedOutOfGunRange(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
	sub.Weapon.HasMainGun = true
	sub.Weapon.MaxToShipRange = 5
	enemy := makeShip("enemy", faction.HumanAlpha, objUnit.ShipTypeDestroyer, 15, 5)

	ms := newMission(0, []*objUnit.BattleShip{sub, enemy}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid),
		"还没进甲板炮射程时不应上浮")
}

// TestSubmarineAIDivesWhenSurfacedEvenWithoutEnemy 验证水面开局的 AI 潜艇立即回到水下。
// 场景：潜艇被关卡配成 DiveStateSurface、带鱼雷，场上没有任何敌舰可打。
// 预期：本拍下发 ShipDive，且不含 ShipSurface；无敌情不该成为停在水面的理由。
// 覆盖：snapshot 后调用 submarineInstructions，断言下潜键存在、上浮键缺席。
func TestSubmarineAIDivesWhenSurfacedEvenWithoutEnemy(t *testing.T) {
	sub := makeShip("sub", faction.ComputerAlpha, objUnit.ShipTypeSubmarine, 5, 5)
	sub.Submarine.DiveState = objUnit.DiveStateSurface
	sub.Weapon.HasTorpedo = true

	ms := newMission(0, []*objUnit.BattleShip{sub}, nil)
	handler := NewHandler(faction.ComputerAlpha)
	snap := handler.snapshot(ms)
	out := handler.submarineInstructions(snap, 0)

	require.Contains(t, out, instr.GenInstrUid(instr.NameShipDive, sub.Uid),
		"即使无敌情，AI 潜艇也应主动回到水下")
	require.NotContains(t, out, instr.GenInstrUid(instr.NameShipSurface, sub.Uid))
}
