package manager

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// TestRamSettingOnlyGatesNormalShips 验证撞击开关只影响普通战舰，alwaysOn 特殊单位始终可撞。
// 场景：config.G.EnableRamming=false；普通舰（alwaysOn=false）与特殊舰（alwaysOn=true）
// 分别以 13.6 节舰艏撞向横向目标，位置使两舰 OBB 重叠 0.18 格。
// 预期：普通舰目标 HP 不变；打开开关后目标掉 1800；特殊舰在开关关闭时也能撞掉 1800。
// 覆盖：三种组合共三次 updateShipContacts，断言目标 CurHP；失败说明开关泄到了特殊单位，
// 或普通战舰在默认关闭时仍然造成撞击伤害。
func TestRamSettingOnlyGatesNormalShips(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	normalAttacker := ramTestShip("a-normal", faction.HumanAlpha, 10, 10, 0, 1800, false)
	normalTarget := ramTestShip("b-target", faction.ComputerAlpha, 10, 9.25, 90, 4000, false)
	m := newRamTestMission(normalAttacker, normalTarget)
	m.updateShipContacts()
	require.Equal(t, 4000.0, normalTarget.CurHP, "normal ram must stay off by default")

	config.G.EnableRamming = true
	m.updateShipContacts()
	require.InDelta(t, 2200, normalTarget.CurHP, 1e-9, "enabled normal ram must deal 1800")

	specialAttacker := ramTestShip("a-special", faction.HumanAlpha, 10, 10, 0, 1800, true)
	specialTarget := ramTestShip("b-special-target", faction.ComputerAlpha, 10, 9.25, 90, 4000, false)
	config.G.EnableRamming = false
	newRamTestMission(specialAttacker, specialTarget).updateShipContacts()
	require.InDelta(t, 2200, specialTarget.CurHP, 1e-9, "alwaysOn ram must ignore the setting")
}

// TestRamCooldownBlocksRepeatedContact 验证同一对舰船在冷却内只结算一次撞击。
// 场景：开启撞击，两舰持续重叠；同一帧连续调用两次 updateShipContacts，再推进 4.1 秒。
// 预期：第二次调用目标 HP 不变；推进过冷却后再次掉 1800。
// 覆盖：SimTick 0/241 两次调用与冷却边界；失败说明重叠时每拍都会扣血，TTK 会被压到不可控。
func TestRamCooldownBlocksRepeatedContact(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	attacker := ramTestShip("a-cooldown", faction.HumanAlpha, 10, 10, 0, 1800, false)
	target := ramTestShip("b-cooldown", faction.ComputerAlpha, 10, 9.25, 90, 4000, false)
	m := newRamTestMission(attacker, target)
	m.updateShipContacts()
	m.updateShipContacts()
	require.InDelta(t, 2200, target.CurHP, 1e-9)

	objUnit.SetSimTick(objUnit.ReloadTicks(attacker.Ram.Cooldown) + 1)
	m.updateShipContacts()
	require.InDelta(t, 400, target.CurHP, 1e-9)
}

// TestRamSkipsFriendlyAndSubmergedShips 验证友军与水下潜艇不会参与撞击接触。
// 场景：开启撞击；友军目标与己方撞击舰重叠；另一组的目标设为完全下潜。
// 预期：两组目标 HP 都不变。
// 覆盖：同阵营过滤与 Submerged 过滤；失败说明友军会被误伤，或水面舰能撞到水下潜艇。
func TestRamSkipsFriendlyAndSubmergedShips(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	friendlyAttacker := ramTestShip("a-friendly", faction.HumanAlpha, 10, 10, 0, 1800, false)
	friendlyTarget := ramTestShip("b-friendly", faction.HumanAlpha, 10, 9.25, 90, 4000, false)
	newRamTestMission(friendlyAttacker, friendlyTarget).updateShipContacts()
	require.Equal(t, 4000.0, friendlyTarget.CurHP)

	subAttacker := ramTestShip("a-sub", faction.HumanAlpha, 10, 10, 0, 1800, false)
	subTarget := ramTestShip("b-sub", faction.ComputerAlpha, 10, 9.25, 90, 4000, false)
	subTarget.Submarine.DiveState = objUnit.DiveStateSubmerged
	newRamTestMission(subAttacker, subTarget).updateShipContacts()
	require.Equal(t, 4000.0, subTarget.CurHP)
}

// TestSpecialShipRamsPlaneThroughWeaponCap 验证水滴式 canHitPlanes 撞击可一击击落飞机。
// 场景：特殊舰 alwaysOn=true、canHitPlanes=true，以 40 节撞向正前方的 100 HP 飞机；
// 普通舰-对空单发上限是 60 HP，撞击必须绕过该上限。
// 预期：飞机 CurHP=0；失败说明撞机被 60% 上限截断，特殊单位的对空能力名存实亡。
// 覆盖：舰-机接触、舰艏朝向、接近速度与 Plane.HurtBy 的 TypeRam 放行分支。
func TestSpecialShipRamsPlaneThroughWeaponCap(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	ship := ramTestShip("waterdrop-test", faction.HumanAlpha, 10, 10, 0, 1800, true)
	ship.Ram.CanHitPlanes = true
	ship.Ram.Damage = 300000
	ship.MaxSpeed = 50.0 / objUnit.ShipSpeedScale
	ship.CurSpeed = 40.0 / objUnit.ShipSpeedScale
	plane := &objUnit.Plane{
		Uid: "plane-test", Name: "plane-test", TotalHP: 100, CurHP: 100,
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(10, 9.2), CurRotation: 90,
		Length: 20, Width: 20,
	}
	ms := newRamTestState(ship)
	ms.Arena.Planes = map[string]*objUnit.Plane{plane.Uid: plane}
	ms.Arena.PutPlane(plane)
	m := &MissionManager{state: ms, instructionSet: NewInstructionSet()}
	m.SetSimulationOptions(SimulationOptions{Headless: true})
	m.updateShipContacts()
	require.Zero(t, plane.CurHP)
}

// ramTestShip 构造一艘撞击测试舰：舰长 200m、宽 40m，航速与最大速度均为 13.6 节。
func ramTestShip(
	uid string, player faction.Player, x, y, rot, ramDamage float64, alwaysOn bool,
) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: uid, Name: uid, Type: objUnit.ShipTypeDestroyer,
		CurHP: 4000, TotalHP: 4000, Tonnage: 4000,
		BelongPlayer: player,
		CurPos:       objPos.NewR(x, y), CurRotation: rot,
		Length: 200, Width: 40,
		MaxSpeed: 13.6 / objUnit.ShipSpeedScale,
		CurSpeed: 13.6 / objUnit.ShipSpeedScale,
		Ram: objUnit.RamConfig{
			Damage: ramDamage, Cooldown: 4, SelfDamageRatio: 0, MinSpeed: 4, BowArc: 60,
			AlwaysOn: alwaysOn,
		},
	}
}

func newRamTestState(ships ...*objUnit.BattleShip) *state.MissionState {
	ms := &state.MissionState{
		Core:   state.MissionCoreState{MissionStatus: state.MissionRunning},
		Player: state.MissionPlayerState{CurPlayer: faction.HumanAlpha},
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{},
			Planes:            map[string]*objUnit.Plane{},
			ShipUidGenerators: map[faction.Player]*objUnit.ShipUidGenerator{},
		},
		UI: state.MissionUIState{GameMarks: map[objMark.ID]*objMark.Mark{}},
	}
	ms.Stats = state.NewBattleStats()
	for _, ship := range ships {
		ms.Arena.PutShip(ship)
	}
	return ms
}

func newRamTestMission(ships ...*objUnit.BattleShip) *MissionManager {
	m := &MissionManager{state: newRamTestState(ships...), instructionSet: NewInstructionSet()}
	m.SetSimulationOptions(SimulationOptions{Headless: true})
	return m
}

// TestDerivedRamOnlyWhenSettingEnabled 验证按舰种推导的默认撞击同样受开关控制。
// 场景：战列舰未显式配置 ram，只拿到 DefaultRamProfile 推导值；航母不推导撞击；
// 两舰分别以 13.6 节舰艏撞向 4000 HP 目标，先关闭再打开 enableRamming。
// 预期：开关关闭时战列舰目标 HP 不变；打开后目标掉约 1152；航母即使开关打开也无效。
// 覆盖：默认配置 + 开关两态 + 非战斗舰种排除；失败说明默认撞击绕过了开关，或航母/货轮被误加撞击。
func TestDerivedRamOnlyWhenSettingEnabled(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	attacker := ramTestShip("a-derived", faction.HumanAlpha, 10, 10, 0, 0, false)
	attacker.Type = objUnit.ShipTypeBattleShip
	attacker.TotalHP, attacker.Tonnage = 4000, 4000
	attacker.HorizontalDamageReduction = 0.4
	profile, ok := objUnit.DefaultRamProfile(
		attacker.Type, attacker.TotalHP,
		attacker.HorizontalDamageReduction, attacker.VerticalDamageReduction,
	)
	require.True(t, ok)
	attacker.Ram = profile
	target := ramTestShip("b-derived", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	target.HorizontalDamageReduction = 0.2
	m := newRamTestMission(attacker, target)
	m.updateShipContacts()
	require.Equal(t, 4000.0, target.CurHP)

	config.G.EnableRamming = true
	m.updateShipContacts()
	require.InDelta(t, 2848, target.CurHP, 1e-9)

	carrier := ramTestShip("a-carrier", faction.HumanAlpha, 10, 10, 0, 0, false)
	carrier.Type = objUnit.ShipTypeAircraftCarrier
	carrier.Ram = objUnit.RamConfig{}
	carrierTarget := ramTestShip("b-carrier", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	newRamTestMission(carrier, carrierTarget).updateShipContacts()
	require.Equal(t, 4000.0, carrierTarget.CurHP)
}

// TestHighSpeedRamSweepsThroughTarget 验证 1000 节级高速舰不会在两拍之间穿过目标。
// 场景：撞击舰本拍从 x=8 移动到 x=10（CurSpeed=2 格/拍），目标停在 x=9.2；
// 当前 OBB 已错开，只有扫掠路径能发现接触。舰艏判定必须用本拍起点，否则接触点会跑到舰艉。
// 预期：5,000,000 HP 目标掉血，证明扫掠 + 起点舰艏判定生效；失败说明水滴会直接穿透目标。
// 覆盖：updateShipContacts 的高速扫掠分支、ramContactOrigin 与 contactInBow 的起点参考。
func TestHighSpeedRamSweepsThroughTarget(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	attacker := ramTestShip("a-fast", faction.HumanAlpha, 10, 10, 90, 300000, true)
	attacker.CurSpeed = 2.0
	attacker.MaxSpeed = 1000.0 / objUnit.ShipSpeedScale
	attacker.Ram.MinSpeed = 1
	target := ramTestShip("b-fast-target", faction.ComputerAlpha, 9.2, 10, 0, 0, false)
	target.TotalHP, target.CurHP = 5000000, 5000000
	target.Length, target.Width = 200, 40
	m := newRamTestMission(attacker, target)
	m.updateShipContacts()
	require.Less(t, target.CurHP, 5000000.0)
}

// TestRamSelfDamageAppearsInBattleLog 验证撞击反作用伤害写入战况详情。
// 场景：开启撞击，撞击舰 4000 HP、damage=1800、自伤比 0.25，撞 4000 HP 目标；
// 目标减伤 0，因此目标掉 1800、撞击舰自伤 450。
// 预期：战报同时有一条目标命中与一条 BattleLogShipSelfHit；自伤条目的 Actor/Target
// 都是撞击舰、WeaponType=ram、Damage=450，撞击舰 DamageTaken=450。
// 覆盖：applyRamSelfDamage 的日志写入与伤害累计；失败说明自伤只扣血、不进战况详情。
func TestRamSelfDamageAppearsInBattleLog(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	attacker := ramTestShip("a-self", faction.HumanAlpha, 10, 10, 0, 1800, false)
	attacker.Ram.SelfDamageRatio = 0.25
	target := ramTestShip("b-self", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	m := newRamTestMission(attacker, target)
	m.updateShipContacts()

	require.InDelta(t, 3550, attacker.CurHP, 1e-9)
	require.InDelta(t, 450, attacker.Damage.DamageTaken, 1e-9)
	var selfHit *state.BattleLogEntry
	for idx := range m.state.Stats.BattleLog {
		if m.state.Stats.BattleLog[idx].Action == state.BattleLogShipSelfHit {
			selfHit = &m.state.Stats.BattleLog[idx]
		}
	}
	require.NotNil(t, selfHit)
	require.Equal(t, attacker.Name, selfHit.ActorName)
	require.Equal(t, attacker.Name, selfHit.TargetName)
	require.Equal(t, faction.HumanAlpha, selfHit.ActorPlayer)
	require.Equal(t, "ram", selfHit.Hit.WeaponType)
	require.InDelta(t, 450, selfHit.Hit.Damage, 1e-9)
}
