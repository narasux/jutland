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
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// TestRamSettingOnlyGatesNormalShips 验证撞击开关只影响普通战舰，alwaysOn 特殊单位始终可撞。
// 场景：config.G.EnableRamming=false；普通舰（alwaysOn=false）与特殊舰（alwaysOn=true）
// 分别以 13.6 节舰艏撞向横向目标；碰撞响应会先把两舰分开，所以打开开关后
// 重新摆回重叠位置并隔一拍再检测。
// 预期：普通舰目标 HP 不变；打开开关后目标掉 1800；特殊舰在开关关闭时也能撞掉 1800。
// 覆盖：开关两态 + 重置接触状态 + alwaysOn 绕过；失败说明开关泄到了特殊单位，
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

	// 碰撞响应已把两舰推开：先记录一次无接触，再摆回重叠，避免被当成持续接触。
	objUnit.SetSimTick(1)
	normalAttacker.CurPos, normalTarget.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 20)
	m.updateShipContacts()
	config.G.EnableRamming = true
	objUnit.SetSimTick(3)
	normalAttacker.CurPos, normalTarget.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 9.25)
	normalAttacker.CurSpeed = 13.6 / objUnit.ShipSpeedScale
	m.updateShipContacts()
	require.InDelta(t, 2200, normalTarget.CurHP, 1e-9, "enabled normal ram must deal 1800")

	specialAttacker := ramTestShip("a-special", faction.HumanAlpha, 10, 10, 0, 1800, true)
	specialTarget := ramTestShip("b-special-target", faction.ComputerAlpha, 10, 9.25, 90, 4000, false)
	config.G.EnableRamming = false
	newRamTestMission(specialAttacker, specialTarget).updateShipContacts()
	require.InDelta(t, 2200, specialTarget.CurHP, 1e-9, "alwaysOn ram must ignore the setting")
}

// TestRamDamagesOncePerContinuousContact 验证持续重叠只在接触开始那一拍结算一次撞击。
// 场景：开启撞击，两舰持续重叠 120 拍；每拍都把位置摆回重叠，模拟被碰撞响应推开后
// 仍贴在一起摩擦的情况。随后分开、度过 4 秒冷却，再次接触。
// 预期：120 拍内目标只掉一次 1800；冷却结束并重新接触后再掉一次。
// 覆盖：RamContactBegan 的接触开始判定、冷却边界与重新接触；失败说明重叠时仍会
// 按冷却反复掉血，或分开后无法再次撞击。
func TestRamDamagesOncePerContinuousContact(t *testing.T) {
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
	for tick := int64(0); tick < 120; tick++ {
		objUnit.SetSimTick(tick)
		attacker.CurPos, target.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 9.25)
		m.updateShipContacts()
		require.InDelta(t, 2200, target.CurHP, 1e-9, "continuous overlap must not tick damage")
	}

	objUnit.SetSimTick(objUnit.ReloadTicks(attacker.Ram.Cooldown) + 120)
	attacker.CurPos, target.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 20)
	m.updateShipContacts()
	attacker.CurSpeed = 13.6 / objUnit.ShipSpeedScale
	objUnit.SetSimTick(objUnit.ReloadTicks(attacker.Ram.Cooldown) + 122)
	attacker.CurPos, target.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 9.25)
	m.updateShipContacts()
	require.InDelta(t, 400, target.CurHP, 1e-9, "separated and cooled down pair may ram again")
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

	// 碰撞响应已分开两舰；摆回重叠并隔一拍，确保这是一次新的接触。
	objUnit.SetSimTick(1)
	attacker.CurPos, target.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 20)
	m.updateShipContacts()
	config.G.EnableRamming = true
	objUnit.SetSimTick(3)
	attacker.CurPos, target.CurPos = objPos.NewR(10, 10), objPos.NewR(10, 9.25)
	attacker.CurSpeed = 13.6 / objUnit.ShipSpeedScale
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

// TestShipContactSeparatesAndSlowsHeadOn 验证敌舰接触会按吨位分离并产生正撞掉速。
// 场景：开启撞击伤害与舰体碰撞检查，两艘 4000 吨、200m 长的驱逐舰中心距 0.75 格
// （最小平移深度 0.1875 格），撞击舰以 13.6 节舰艏正撞静止目标。
// 预期：一拍后两舰不再接触（中心距约 0.9375 格），撞击舰航速降到 20%，
// 静止目标保持不动；失败说明碰撞响应没有推开舰体，或掉速把航速清成 0。
// 覆盖：resolveShipContact 的法线方向、吨位权重、单拍分离上限与 applyContactSpeedLoss；
// 舰体碰撞开关关闭时掉速/分离不生效另由 TestShipCollisionSeparatesEnemyHullsOnlyWhenEnabled 覆盖。
func TestShipContactSeparatesAndSlowsHeadOn(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = true
	config.G.EnableShipCollision = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	attacker := ramTestShip("a-separate", faction.HumanAlpha, 10, 10, 0, 1800, false)
	target := ramTestShip("b-separate", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	target.CurSpeed = 0
	before := attacker.CurPos.Distance(target.CurPos)
	m := newRamTestMission(attacker, target)
	m.updateShipContacts()

	after := attacker.CurPos.Distance(target.CurPos)
	require.Greater(t, after, before, "contact must separate the hulls")
	require.Greater(t, attacker.CurPos.RY, 10.0, "attacker must be pushed away from the target")
	require.Less(t, target.CurPos.RY, 9.25, "target must be pushed away from the attacker")
	_, stillTouching := contactBetweenShips(attacker, target)
	require.False(t, stillTouching, "one tick of separation must clear a shallow contact")
	require.InDelta(t, 13.6*0.2/objUnit.ShipSpeedScale, attacker.CurSpeed, 1e-9)
	require.Zero(t, target.CurSpeed, "stationary target must not be accelerated by separation")
}

// TestRamImpactPointStaysOnAttackerBow 验证撞击表现锚点落在撞击舰舰艏而不是接触点原坐标。
// 场景：撞击舰舰长 200m（半长 0.78125 格）、宽 40m，航向正北；接触点取在舰体后方
// 0.4 格、横向偏右 0.2 格（模拟高速扫掠 / 深穿透后 SAT 接触点落到舰体中段）。
// 预期：返回点相对中心的前向投影等于 0.9 × 半舰长，横向偏移被夹到半舰宽 0.15625 格；
// 失败说明撞击闪光 / 伤害数字仍会画到舰体中部或目标中心。
// 覆盖：ramImpactPoint 的纵向固定与横向夹取，不验证像素绘制。
func TestRamImpactPointStaysOnAttackerBow(t *testing.T) {
	ship := ramTestShip("a-impact", faction.HumanAlpha, 10, 10, 0, 1800, false)
	x, y := ramImpactPoint(ship, 10.2, 10.4)

	const halfLen, halfWid = 200.0 / 128 / 2, 40.0 / 128 / 2
	forward := (y-10.0)*-1 + (x-10.0)*0
	require.InDelta(t, 0.9*halfLen, forward, 1e-9)
	require.InDelta(t, halfWid, x-10.0, 1e-9)
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

// TestShipCollisionSeparatesEnemyHullsOnlyWhenEnabled 验证舰体碰撞开关控制敌方舰船分离，
// 且分离不依赖撞击伤害开关。
// 场景：两艘不同阵营舰船 OBB 重叠；先关闭舰体碰撞检查调用一次，再打开后调用一次。
// 预期：关闭时双方位置完全不变；打开后双方被推开、不再接触，但撞击伤害关闭所以 HP 不变。
// 覆盖：同一对舰连续两次 updateShipContacts，比较前后 CurPos 与 shipRectContactAt，断言 HP 未变。
func TestShipCollisionSeparatesEnemyHullsOnlyWhenEnabled(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	config.G.EnableShipCollision = false
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	a := ramTestShip("a-collision", faction.HumanAlpha, 10, 10, 0, 1800, false)
	b := ramTestShip("b-collision", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	m := newRamTestMission(a, b)
	beforeA, beforeB := a.CurPos.Copy(), b.CurPos.Copy()

	m.updateShipContacts()
	require.Equal(t, beforeA, a.CurPos, "collision check off must not move ships")
	require.Equal(t, beforeB, b.CurPos, "collision check off must not move ships")

	config.G.EnableShipCollision = true
	m.updateShipContacts()
	require.NotEqual(t, beforeA, a.CurPos, "enemy hulls must separate when enabled")
	require.NotEqual(t, beforeB, b.CurPos, "enemy hulls must separate when enabled")
	require.False(t, shipRectContactAt(a, a.CurPos.RX, a.CurPos.RY, b).OK)
	require.Equal(t, 4000.0, a.CurHP, "collision separation must not deal ram damage")
	require.Equal(t, 4000.0, b.CurHP, "collision separation must not deal ram damage")
}

// TestShipCollisionSkipsFriendlyHulls 验证舰体碰撞检查只作用于不同阵营，友军仍可穿行。
// 场景：关闭撞击伤害、打开舰体碰撞检查，两艘同阵营舰船 OBB 重叠。
// 预期：双方位置与 HP 都不变，仍保持接触；失败说明友军被误分离，AI 编队会互相顶开。
// 覆盖：同阵营一对舰调用 updateShipContacts，比较 CurPos、CurHP 与 contact 结果。
func TestShipCollisionSkipsFriendlyHulls(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	config.G.EnableShipCollision = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	a := ramTestShip("a-friendly-collision", faction.HumanAlpha, 10, 10, 0, 0, false)
	b := ramTestShip("b-friendly-collision", faction.HumanAlpha, 10, 9.25, 90, 0, false)
	m := newRamTestMission(a, b)
	beforeA, beforeB := a.CurPos.Copy(), b.CurPos.Copy()

	m.updateShipContacts()
	require.Equal(t, beforeA, a.CurPos, "friendly hulls must keep passing through")
	require.Equal(t, beforeB, b.CurPos, "friendly hulls must keep passing through")
	require.True(t, shipRectContactAt(a, a.CurPos.RX, a.CurPos.RY, b).OK)
	require.Equal(t, 4000.0, a.CurHP)
	require.Equal(t, 4000.0, b.CurHP)
}

// TestRamAndCollisionSwitchesAreIndependent 验证撞击伤害与舰体碰撞可以分别开关。
// 场景：同一对重叠敌舰，先只开撞击伤害（碰撞检查关），再只开舰体碰撞检查（撞击伤害关）。
// 预期：只开撞击伤害时目标掉血但双方位置不变；只开舰体碰撞时双方分开但都不掉血。
// 覆盖：两组独立状态各调用一次 updateShipContacts，分别断言 CurHP 与 CurPos。
func TestRamAndCollisionSwitchesAreIndependent(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	ramA := ramTestShip("a-ram-only", faction.HumanAlpha, 10, 10, 0, 1800, false)
	ramB := ramTestShip("b-ram-only", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	config.G.EnableRamming = true
	config.G.EnableShipCollision = false
	ramMission := newRamTestMission(ramA, ramB)
	beforeRamA, beforeRamB := ramA.CurPos.Copy(), ramB.CurPos.Copy()
	ramMission.updateShipContacts()
	require.InDelta(t, 2200, ramB.CurHP, 1e-9, "ram damage must still apply")
	require.Equal(t, beforeRamA, ramA.CurPos, "ram-only mode must not separate hulls")
	require.Equal(t, beforeRamB, ramB.CurPos, "ram-only mode must not separate hulls")

	collisionA := ramTestShip("a-collision-only", faction.HumanAlpha, 10, 10, 0, 1800, false)
	collisionB := ramTestShip("b-collision-only", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	config.G.EnableRamming = false
	config.G.EnableShipCollision = true
	collisionMission := newRamTestMission(collisionA, collisionB)
	beforeCollisionA, beforeCollisionB := collisionA.CurPos.Copy(), collisionB.CurPos.Copy()
	collisionMission.updateShipContacts()
	require.Equal(t, 4000.0, collisionB.CurHP, "collision-only mode must not deal ram damage")
	require.NotEqual(t, beforeCollisionA, collisionA.CurPos, "collision-only mode must separate hulls")
	require.NotEqual(t, beforeCollisionB, collisionB.CurPos, "collision-only mode must separate hulls")
}

// TestShipCollisionSeparatesDeepOverlapOverTicks 验证深穿透不会被一拍弹开：
// 单拍单舰位移被 maxContactSeparation 截断，剩余重叠留到后续拍继续处理。
// 场景：两艘同向 200m 等质量舰中心相距 0.5 格，沿舰长方向重叠 1.0625 格（超过 2 倍上限），
// 关闭撞击伤害、只开舰体碰撞检查。
// 预期：第一拍每舰理论位移 1.0625/2=0.53125 被截断到 0.5 格，两舰仍接触；
// 第二拍后完全分离。失败说明深穿透被一拍弹开，或上限没有生效。
// 覆盖：连续两次 updateShipContacts，断言 CurPos.RY 位移恰为 maxContactSeparation，
// 并用 shipRectContactAt 断言第一拍仍接触、第二拍已分离。
func TestShipCollisionSeparatesDeepOverlapOverTicks(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	config.G.EnableShipCollision = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	a := ramTestShip("a-deep", faction.HumanAlpha, 10, 10, 0, 0, false)
	b := ramTestShip("b-deep", faction.ComputerAlpha, 10, 9.5, 0, 0, false)
	// 加宽舰体，让 SAT 的最小平移方向落在舰长方向（深度 0.5625 > 0.5），
	// 而不是先沿较窄的舰宽分开。
	a.Width, b.Width = 400, 400
	m := newRamTestMission(a, b)

	m.updateShipContacts()
	// 每舰理论位移 0.53125 超过上限，本拍只走 0.5。
	require.InDelta(t, 10+maxContactSeparation, a.CurPos.RY, 1e-9)
	require.InDelta(t, 9.5-maxContactSeparation, b.CurPos.RY, 1e-9)
	require.True(t, shipRectContactAt(a, a.CurPos.RX, a.CurPos.RY, b).OK)

	m.updateShipContacts()
	require.False(t, shipRectContactAt(a, a.CurPos.RX, a.CurPos.RY, b).OK)
}

// TestShipCollisionDistributesByMass 验证分离位移按吨位反比分配，重舰几乎不动。
// 场景：8000 吨与 2000 吨的两艘敌舰 OBB 重叠，开启舰体碰撞检查。
// 预期：两舰位移比接近 1:4（轻舰走 4 倍距离），而不是各走一半。
// 覆盖：调用一次 updateShipContacts，用位移长度比断言质量分配。
func TestShipCollisionDistributesByMass(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	config.G.EnableShipCollision = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	heavy := ramTestShip("a-heavy", faction.HumanAlpha, 10, 10, 0, 0, false)
	light := ramTestShip("b-light", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	heavy.Tonnage, heavy.TotalHP = 8000, 8000
	light.Tonnage, light.TotalHP = 2000, 2000
	heavyStart, lightStart := heavy.CurPos.Copy(), light.CurPos.Copy()
	newRamTestMission(heavy, light).updateShipContacts()

	heavyShift := heavy.CurPos.Distance(heavyStart)
	lightShift := light.CurPos.Distance(lightStart)
	require.Greater(t, heavyShift, 0.0)
	require.Greater(t, lightShift, 0.0)
	require.InDelta(t, 4, lightShift/heavyShift, 0.01)
}

// TestShipCollisionRejectsPushOntoLand 验证分离候选位置会把舰体推上岸时整对取消，
// 不会只推开其中一方或把船顶进陆地。
// 场景：8×3 地图 x=5 整列为陆地；两艘 200m 舰中心在 x=4.8/4.9 的水面，横向重叠 0.2125 格；
// 分离会把右侧舰（b）向东推过 x=5，而左侧舰（a）的分离方向仍在水面。
// 预期：b 的位移被放弃、位置与调用前完全一致；a 仍被推开；两舰舰体都不压上岸。
// 覆盖：设置 Core.MissionMD.MapCfg 后调用 updateShipContacts，比较 CurPos，并用
// HullLandOverlapSamples 断言两舰的压岸采样数都为 0。
func TestShipCollisionRejectsPushOntoLand(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	config.G.EnableShipCollision = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	rows := mapcfg.MapData{"SSSSSLSS", "SSSSSLSS", "SSSSSLSS"}
	a := ramTestShip("a-coast", faction.HumanAlpha, 4.8, 1.5, 0, 0, false)
	b := ramTestShip("b-coast", faction.ComputerAlpha, 4.9, 1.5, 0, 0, false)
	a.Length, b.Length = 200, 200
	m := newRamTestMission(a, b)
	m.state.Core.MissionMD.MapCfg = &mapcfg.MapCfg{
		Map: rows, Cells: rows.ToGridCells(), Width: 8, Height: 3,
	}
	beforeA, beforeB := a.CurPos.Copy(), b.CurPos.Copy()

	m.updateShipContacts()
	require.Equal(t, beforeB, b.CurPos, "blocked ship must keep position instead of being pushed onto land")
	require.NotEqual(t, beforeA, a.CurPos, "the other ship must still be separated")
	require.Zero(
		t,
		objUnit.HullLandOverlapSamples(m.state.Core.MissionMD.MapCfg, a.CurPos, a.CurRotation, a.Length),
		"separation must not push a ship onto land",
	)
	require.Zero(
		t,
		objUnit.HullLandOverlapSamples(m.state.Core.MissionMD.MapCfg, b.CurPos, b.CurRotation, b.Length),
		"separation must not push a ship onto land",
	)
}

// TestShipCollisionSkipsSubmergedSubmarine 验证舰体碰撞检查不会把水下潜艇当水面障碍推开。
// 场景：关闭撞击伤害、打开舰体碰撞检查，敌方潜艇完全下潜并与水面舰 OBB 重叠。
// 预期：双方位置都不变；失败意味着潜艇会被水面舰的分离逻辑顶动，或水下隐身被碰撞暴露。
// 覆盖：设置 DiveStateSubmerged 后调用 updateShipContacts，比较 CurPos。
func TestShipCollisionSkipsSubmergedSubmarine(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	config.G.EnableRamming = false
	config.G.EnableShipCollision = true
	objUnit.SetSimTick(0)
	t.Cleanup(func() {
		config.G = oldSettings
		objUnit.SetSimTick(0)
	})

	surface := ramTestShip("a-surface", faction.HumanAlpha, 10, 10, 0, 0, false)
	sub := ramTestShip("b-sub-collision", faction.ComputerAlpha, 10, 9.25, 90, 0, false)
	sub.Submarine.DiveState = objUnit.DiveStateSubmerged
	beforeSurface, beforeSub := surface.CurPos.Copy(), sub.CurPos.Copy()
	newRamTestMission(surface, sub).updateShipContacts()
	require.Equal(t, beforeSurface, surface.CurPos)
	require.Equal(t, beforeSub, sub.CurPos)
}
