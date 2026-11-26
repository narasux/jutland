package manager

import (
	"math"
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

// TestDepthChargeOnlyDamagesSubmarines 验证深弹只对潜艇结算，且下潜潜艇需被本方点亮。
// 场景：SimTick=100，深弹在 (10,10)、半径 1.3；下潜 sub1 被点亮至 110，浮航 sub2 距 0.5，驱逐舰距 0.8。
// 预期：sub1/sub2 均掉血，驱逐舰保持 100 HP；说明类型过滤、探测记忆与半径判定同时生效。
// 覆盖：depthChargeState 放三舰并 rebuildCombatBuckets，调 resolveDepthChargeDamage 后断言各自 CurHP。
func TestDepthChargeOnlyDamagesSubmarines(t *testing.T) {
	submerged := &objUnit.BattleShip{
		Uid: "sub1", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
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

// TestDepthChargeBlastHitsEverySubmarineInRadius 验证普通深弹一次落点会结算半径内所有潜艇，而不是只打瞄准目标。
// 场景：Mk6 300 磅深弹在 (10,10) 引爆，BlastRadius=1.3；两艘下潜潜艇分别在 (10,10) 与 (10.8,10)，
// 都被 HumanAlpha 点亮至 110；另有一艘水面驱逐舰在 (10.3,10)。
// 预期：两艘潜艇都掉血、驱逐舰保持 100 HP；失败意味着普通深弹退回单目标直击，或 AoE 只结算瞄准艇。
// 覆盖：depthChargeState 放三舰并 rebuildCombatBuckets，调 resolveDepthChargeDamage 后逐舰断言 CurHP。
func TestDepthChargeBlastHitsEverySubmarineInRadius(t *testing.T) {
	aimed := &objUnit.BattleShip{
		Uid: "aimed", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	nearby := &objUnit.BattleShip{
		Uid: "nearby", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.8, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	destroyer := &objUnit.BattleShip{
		Uid: "dd", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.3, 10),
	}
	ms := depthChargeState(aimed, nearby, destroyer)
	aimed.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, aimed.CurPos)
	nearby.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, nearby.CurPos)

	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()
	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	m.resolveDepthChargeDamage(bt)

	require.Less(t, aimed.CurHP, 100.0, "瞄准点上的潜艇应受伤")
	require.Less(t, nearby.CurHP, 100.0, "爆炸半径内的另一艘潜艇也应受伤")
	require.Equal(t, 100.0, destroyer.CurHP, "普通深弹仍只结算潜艇")
}

// TestDepthChargeBlindFireDoesNotDamage 验证未探测到水下潜艇时深弹盲炸不结算伤害。
// 场景：sub 在 (10,10) 完全下潜且 Submarine.DetectedBy 为空，HumanAlpha 的深弹在同一点引爆。
// 预期：resolveDepthChargeDamage 因 CanSeeShip 为 false 跳过该艇，CurHP 保持 100。
// 覆盖：仅含一艘潜艇的 Arena，重建桶后调用落点结算，断言血量不变。
func TestDepthChargeBlindFireDoesNotDamage(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	ms := depthChargeState(sub)
	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()

	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	m.resolveDepthChargeDamage(bt)

	require.Equal(t, 100.0, sub.CurHP, "未被探测时盲炸不结算")
}

// TestDepthChargeShouldExplodeTriggers 验证深弹近炸、越过瞄准点与寿命耗尽三类引爆条件。
// 场景：深弹近炸半径 1.0、Rotation=90、目标 (10,10)，依次放在目标点、(7,10)、(11,10) 与 Life=0 的位置。
// 预期：目标点因距离≤1 引爆；未到且未过点时返回 false；越过瞄准点沿弹道 along≥0；寿命归零也引爆。
// 覆盖：空白 depthChargeState 上调 depthChargeShouldExplode，逐个切换 CurPos/Life/TargetPos 断言 true/false。
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

// TestDepthChargePipelineThroughShotBullets 验证深弹走通用 updateShotBullets 分支完成飞行、引爆与移除。
// 场景：3000 HP 下潜潜艇被点亮至 500 拍，深弹以速度 0.03、寿命 40、近炸半径 1.0 进入弹药队列。
// 预期：通用弹药循环识别 TypeDepthCharge 并调用结算，潜艇掉血，已引爆的深弹从 ForwardingBullets 移除。
// 覆盖：构造带 Shooter/BlastRadius/Target 的深弹，执行一次 updateShotBullets 后断言 CurHP<3000 且队列为空。
func TestDepthChargePipelineThroughShotBullets(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 3000, TotalHP: 3000, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
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

// TestASWRocketPipelineOnlyDamagesSubmarines 验证 65 式火箭深弹按火箭弹飞行，但落点仍走深弹 AoE 且只伤潜艇。
// 场景：3000 HP 下潜潜艇被点亮至 500 拍，艇旁 0.3 格停一艘 100 HP 敌驱逐舰；250mm 火箭弹以
// TypeRocket + AntiSubmarine 进入弹药队列，近炸半径 0.25、爆炸半径 0.9，起点即瞄准点。
// 预期：潜艇掉血、驱逐舰保持 100 HP、弹药从 ForwardingBullets 移除；失败意味着火箭深弹退回普通火箭直击，
// 或把水面舰也纳入落点 AoE。
// 覆盖：构造带反潜标记的火箭实例执行一次 updateShotBullets，断言双方 CurHP 与队列长度，覆盖
// TypeRocket 的深弹引爆分支。
func TestASWRocketPipelineOnlyDamagesSubmarines(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 3000, TotalHP: 3000, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	sub.MarkDetected(faction.HumanAlpha, 500, sub.CurPos)
	escort := &objUnit.BattleShip{
		Uid: "escort", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeDestroyer,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.3, 10),
	}
	ms := depthChargeState(sub, escort)
	ms.Core.SimTick = 100

	bt := &objBullet.Bullet{
		Name: "CN/RB/250/1965/Type65", Type: objBullet.TypeRocket, AntiSubmarine: true,
		Damage: 1000, CurPos: objPos.NewR(10, 10), TargetPos: objPos.NewR(10, 10),
		ProximityRadius: 0.25, BlastRadius: 0.9, Speed: 0.03, Life: 40,
		Shooter: "shooter", ShooterObjType: object.TypeShip,
		BelongPlayer: faction.HumanAlpha, TargetObjType: object.TypeShip,
	}
	ms.Arena.ForwardingBullets = []*objBullet.Bullet{bt}

	m := &MissionManager{state: ms}
	m.updateShotBullets()

	require.Less(t, sub.CurHP, 3000.0, "火箭深弹应在落点 AoE 中伤到潜艇")
	require.Equal(t, 100.0, escort.CurHP, "火箭深弹不应伤到水面舰")
	require.Empty(t, ms.Arena.ForwardingBullets, "火箭深弹引爆后不应继续飞行")
}

// TestDepthChargeRackCanFireAhead 验证舰艏正对目标时前方深弹架仍能按左右射界投放。
// 场景：驱逐舰在 (10,10)、航向 90，潜艇在正前方 (11.5,10)；深弹架位置 -0.86，左右射界合为全周。
// 预期：launcher.Fire 返回 1 发弹药且类型为 TypeDepthCharge，证明前方目标未被射界或目标门槛过滤。
// 覆盖：重置 config.G 消除速度倍率影响，用 NewRocketLauncher("UK/DC/5/MkVII") 构造后断言弹药数量与类型。
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
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
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

// TestDepthChargeSinkRecordsShooter 验证深弹击沉潜艇时记录投弹方与武器，战报能显示「谁用什么击沉」。
// 场景：nautilus 50/100 在 (10,10)，gurkha 的 Mk6 深弹以 Shooter=dd、ShooterName=gurkha、ShooterObjType=TypeShip 在同点结算，随后 Stats.RecordShipLost。
// 预期：HP 归零、Damage.KilledByHit=true，Damage.LastAttackerName/Type/Player 与 Damage.LastHit.WeaponType=depth_charge 完整；战报 1 条 ShipSunk 且 ActorName=gurkha、Hit.WeaponType=depth_charge。
// 覆盖：resolveDepthChargeDamage 的击杀字段写入与 RecordShipLost 的战报生成，回归「只剩沉没潜艇、无投弹方」。
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
	require.True(t, sub.Damage.KilledByHit)
	require.Equal(t, "gurkha", sub.Damage.LastAttackerName)
	require.Equal(t, string(objUnit.ShipTypeDestroyer), sub.Damage.LastAttackerType)
	require.Equal(t, faction.HumanAlpha, sub.Damage.LastAttackerPlayer)
	require.Equal(t, string(objBullet.TypeDepthCharge), sub.Damage.LastHit.WeaponType)
	require.Positive(t, sub.Damage.DamageTaken)

	// 击沉记录必须带上投弹方和武器，详情页才能渲染「驱逐舰 X 使用 深水炸弹 击沉 潜艇 Y」。
	ms.Stats.RecordShipLost(sub)
	require.Len(t, ms.Stats.BattleLog, 1)
	sink := ms.Stats.BattleLog[0]
	require.Equal(t, state.BattleLogShipSunk, sink.Action)
	require.Equal(t, "gurkha", sink.ActorName)
	require.Equal(t, string(objBullet.TypeDepthCharge), sink.Hit.WeaponType)
}

// TestDepthChargeSkipsShooterHullOnLaunch 验证深弹从投弹舰舰内出发时不会把本方舰体当成直接命中而原地引爆。
// 场景：porter 在 (10,10)、u_96 在 (14,10)；深弹起点就在本舰坐标，既未接近瞄准点也未越过目标，
// 修复前 insideAnyHull 会命中投弹舰自身，深弹在舰尾自爆、对 1 格外潜艇永远差 0.06 格打不中。
// 预期：起点在本舰舰体内时 depthChargeShouldExplode 返回 false（继续飞行）；
// 把同一条深弹移到目标潜艇坐标时返回 true，说明其他舰体仍可触发直接引爆。
// 覆盖：depthChargeState 放 porter + u_96 并 rebuildCombatBuckets，切换 CurPos 分别断言 false/true。
func TestDepthChargeSkipsShooterHullOnLaunch(t *testing.T) {
	destroyer := newNamedShip(t, "porter", faction.HumanAlpha, 10, 10)
	sub := newNamedShip(t, "u_96", faction.ComputerAlpha, 14, 10)

	ms := depthChargeState(destroyer, sub)
	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()

	bt := newDepthCharge(destroyer.CurPos.Copy(), faction.HumanAlpha)
	bt.Shooter = destroyer.Uid
	bt.Rotation = 90
	bt.TargetPos = sub.CurPos.Copy()
	bt.Life = 100
	require.False(t, m.depthChargeShouldExplode(bt), "深弹在本舰舰体内不应立即引爆")

	bt.CurPos = sub.CurPos.Copy()
	require.True(t, m.depthChargeShouldExplode(bt), "深弹落在其他舰体上应直接引爆")
}

// TestAirDroppedDepthChargeDealsDamage 验证机载深弹从弹药模板继承爆炸半径并真正结算伤害。
// 场景：飞机投下 US/DC/300/1940/Mk6；旧配置漏写 blastRadius，弹体半径 0，
// resolveDepthChargeDamage 的 eachShip(radius=0) 直接跳过，表现为有水花但潜艇不掉血。
// 预期：New 出的深弹 BlastRadius=1.3；潜艇被本方点亮后，落点结算使 CurHP 下降。
// 覆盖：用真实弹药模板构造深弹（不手填 BlastRadius），rebuildCombatBuckets 后调用 resolveDepthChargeDamage。
func TestAirDroppedDepthChargeDealsDamage(t *testing.T) {
	sub := &objUnit.BattleShip{
		Uid: "sub", CurHP: 100, TotalHP: 100, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	ms := depthChargeState(sub)
	sub.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, sub.CurPos)

	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()

	bt := objBullet.New(
		"US/DC/300/1940/Mk6", sub.CurPos, sub.CurPos,
		"plane", object.TypePlane, faction.HumanAlpha, object.TypeShip,
		0, 10, 1,
	)
	require.InDelta(t, 1.3, bt.BlastRadius, 1e-9, "机载深弹应从模板继承爆炸半径")

	m.resolveDepthChargeDamage(bt)
	require.Less(t, sub.CurHP, 100.0, "机载深弹命中点附近应结算伤害")
}

// TestAirDroppedDepthChargeTemplatesHaveBlastRadius 验证三种机载深弹模板都配置了爆炸半径。
// 场景：B-24D / SB2C-4 / SeaOtter 的深弹分别引用英、美、日三套弹药，漏配任意一套都会让该机种投弹无伤害。
// 预期：三套深弹模板的 BlastRadius 都大于 0，300 磅为 1.3、日本 200kg 为 0.9。
// 覆盖：遍历 objBullet.Map 的三个真实名称，用 InDelta 断言半径数值。
func TestAirDroppedDepthChargeTemplatesHaveBlastRadius(t *testing.T) {
	cases := map[string]float64{
		"UK/DC/300/1939/MkVII":  1.3,
		"US/DC/300/1940/Mk6":    1.3,
		"JP/DC/200/1935/Type95": 0.9,
	}
	for name, want := range cases {
		bt := objBullet.Map[name]
		require.NotNil(t, bt, "深弹 %s 应已加载", name)
		require.InDelta(t, want, bt.BlastRadius, 1e-9, "深弹 %s 的爆炸半径", name)
	}
}

// TestDepthChargeBlastFactorFallsOffWithDistance 验证深弹落点的伤害系数随爆心距线性衰减。
// 场景：半径 1.0 时分别取爆心 0、核心区边界 0.3、核心区外中点 0.65、爆炸边缘 1.0、半径外 2.0，
// 外加 radius=0 与 distance=NaN 两种脏数据；这些正是 resolveDepthChargeDamage 逐目标缩放的输入。
// 预期：0 与 0.3 处为 1，0.65 处为 0.675，1.0 与 2.0 处回落到 depthChargeBlastMinFactor=0.35，
// radius=0 和 NaN 距离按满额 1；失败说明衰减曲线、核心区边界或异常兜底被改动。
// 覆盖：表驱动 7 组输入，用 InDelta(1e-9) 断言 depthChargeBlastFactor 的返回值。
func TestDepthChargeBlastFactorFallsOffWithDistance(t *testing.T) {
	cases := []struct {
		name     string
		distance float64
		radius   float64
		want     float64
	}{
		{name: "爆心", distance: 0, radius: 1, want: 1},
		{name: "核心区边界", distance: 0.3, radius: 1, want: 1},
		{name: "核心区外中点", distance: 0.65, radius: 1, want: 0.675},
		{name: "爆炸边缘", distance: 1, radius: 1, want: depthChargeBlastMinFactor},
		{name: "半径外", distance: 2, radius: 1, want: depthChargeBlastMinFactor},
		{name: "零半径按满额", distance: 0.5, radius: 0, want: 1},
		{name: "NaN 距离按满额", distance: math.NaN(), radius: 1, want: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.InDelta(t, tc.want, depthChargeBlastFactor(tc.distance, tc.radius), 1e-9)
		})
	}
}

// TestDepthChargeDamageFallsOffWithDistance 验证一次深弹落点对半径内不同距离的潜艇给出不同伤害。
// 场景：Mk6 深弹 Damage=1000、BlastRadius=1.3 在 (10,10) 引爆；两艘下潜潜艇分别在爆心 (10,10)
// 与 (10.8,10)，都被点亮，初始 HP 均为 4000；旧实现给半径内所有目标同样的 1000。
// 预期：爆心艇掉 1000，0.8 格外的艇掉约 707.14（系数 0.7071429），且结算后 bt.Damage 还原为 1000；
// 失败意味着伤害仍不随距离衰减，或临时缩放的弹体伤害没有被还原。
// 覆盖：断言两艇各自掉血量与 bt.Damage 还原，并对照 depthChargeBlastFactor(0.8,1.3) 的乘积。
func TestDepthChargeDamageFallsOffWithDistance(t *testing.T) {
	center := &objUnit.BattleShip{
		Uid: "center", CurHP: 4000, TotalHP: 4000, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	edge := &objUnit.BattleShip{
		Uid: "edge", CurHP: 4000, TotalHP: 4000, Type: objUnit.ShipTypeSubmarine,
		BelongPlayer: faction.ComputerAlpha, CurPos: objPos.NewR(10.8, 10),
		Submarine: objUnit.SubmarineState{DiveState: objUnit.DiveStateSubmerged},
	}
	ms := depthChargeState(center, edge)
	center.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, center.CurPos)
	edge.MarkDetected(faction.HumanAlpha, ms.Core.SimTick+10, edge.CurPos)

	m := &MissionManager{state: ms}
	m.rebuildCombatBuckets()
	bt := newDepthCharge(objPos.NewR(10, 10), faction.HumanAlpha)
	m.resolveDepthChargeDamage(bt)

	require.InDelta(t, 1000.0, 4000-center.CurHP, 1e-9, "爆心应吃到满额伤害")
	require.InDelta(t, 1000.0*depthChargeBlastFactor(0.8, 1.3), 4000-edge.CurHP, 1e-9,
		"0.8 格外的潜艇应吃到衰减后的伤害")
	require.Less(t, 4000-edge.CurHP, 4000-center.CurHP, "远目标的伤害必须小于爆心")
	require.InDelta(t, 1000.0, bt.Damage, 1e-9, "结算后必须还原弹体伤害")
}
