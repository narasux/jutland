package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// newHomingTestManager 造一枚从 (10,10) 朝正北起飞、目标在正东 4 格的鱼雷。
// 目标静止不动，用来验证纯追踪能否把弹头从错误朝向拐到目标上。
func newHomingTestManager(guided bool) (*MissionManager, *objUnit.BattleShip, *objBullet.Bullet) {
	target := &objUnit.BattleShip{
		Uid:          "target",
		TotalHP:      100,
		CurHP:        100,
		Length:       128,
		Width:        32,
		MaxSpeed:     0.025,
		Acceleration: 0.0001,
		RotateSpeed:  0.3,
		CurPos:       objPos.NewR(14, 10),
		BelongPlayer: faction.ComputerAlpha,
	}
	bt := &objBullet.Bullet{
		Type:          objBullet.TypeTorpedo,
		Damage:        10,
		TargetObjType: object.TypeShip,
		Shooter:       "shooter",
		BelongPlayer:  faction.HumanAlpha,
		CurPos:        objPos.NewR(10, 10),
		Rotation:      0,
		Speed:         0.08,
		Life:          400,
	}
	if guided {
		bt.Guidance = &objBullet.Guidance{
			Mode:     objBullet.GuidanceModePursuit,
			TurnRate: 180,
		}
		bt.TargetShipUid = target.Uid
		bt.GuidePos = target.CurPos
	}
	m := &MissionManager{state: &state.MissionState{
		Arena: state.MissionArenaState{
			Ships:             map[string]*objUnit.BattleShip{target.Uid: target},
			ForwardingBullets: []*objBullet.Bullet{bt},
		},
		// 空地图：测试只关心弹道，不关心地形。
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{MapCfg: &mapcfg.MapCfg{}},
		},
	}}
	return m, target, bt
}

// advanceBullets 推进弹药，直到命中或到达拍数上限。
func advanceBullets(m *MissionManager, maxTicks int) {
	for tick := 0; tick < maxTicks && len(m.state.Arena.ForwardingBullets) > 0; tick++ {
		m.updateShotBullets()
	}
}

// TestGuidedTorpedoCurvesToOffAxisTarget 验证纯追踪制导鱼雷能从错误朝向拐向目标。
// 场景：鱼雷在 (10,10)、Rotation=0 朝正北，静止目标在正东 4 格 (14,10)；GuidanceModePursuit、TurnRate=180°/s、速度 0.08、寿命 400，最多推进 400 拍。
// 预期：target.CurHP 从 100 降到 90（TotalHP=100、Damage=10），即转向后命中。
// 覆盖：guideBullet 的纯追踪分支与 SteerTowards 逐拍修正弹头朝向。
func TestGuidedTorpedoCurvesToOffAxisTarget(t *testing.T) {
	m, target, _ := newHomingTestManager(true)

	advanceBullets(m, 400)

	if target.CurHP >= target.TotalHP {
		t.Fatalf("guided torpedo missed: HP = %.1f, want less than %.1f", target.CurHP, target.TotalHP)
	}
}

// TestUnguidedTorpedoKeepsStraightAndMisses 验证没有 Guidance 的普通鱼雷保持直线，不会拐弯命中侧面目标。
// 场景：与制导用例相同的 (10,10)、Rotation=0 与 (14,10) 静止靶舰，但弹药 Guidance=nil，推进 100 拍。
// 预期：target.CurHP 保持 100，且 bt.Rotation 始终为 0。
// 覆盖：guideBullet 对 !IsGuided 的直航早退与无制导鱼雷弹道。
func TestUnguidedTorpedoKeepsStraightAndMisses(t *testing.T) {
	m, target, bt := newHomingTestManager(false)

	advanceBullets(m, 100)

	if target.CurHP != target.TotalHP {
		t.Fatalf("unguided torpedo hit a side target: HP = %.1f, want %.1f", target.CurHP, target.TotalHP)
	}
	if bt.Rotation != 0 {
		t.Fatalf("unguided torpedo rotation = %v, want 0", bt.Rotation)
	}
}

// TestGuidedTorpedoKeepsLastKnownPointWhenTargetLost 验证目标消失后制导鱼雷继续飞向最后已知点，不 panic 也不重新搜索。
// 场景：制导鱼雷锁定 (14,10) 的靶舰后，把靶舰从 Arena.Ships 删除，再推进 3 拍。
// 预期：bt.GuidePos 仍等于删除前记录的 lastKnown，且 bt.Rotation 不为 0（仍在朝该点转向）。
// 覆盖：guideBullet 对目标缺失的分支——只保留 GuidePos，不读真实位置（避免从航迹泄露潜艇）。
func TestGuidedTorpedoKeepsLastKnownPointWhenTargetLost(t *testing.T) {
	m, target, bt := newHomingTestManager(true)
	lastKnown := bt.GuidePos

	delete(m.state.Arena.Ships, target.Uid)
	advanceBullets(m, 3)

	if bt.GuidePos != lastKnown {
		t.Fatalf("guide pos = %v, want last known %v", bt.GuidePos, lastKnown)
	}
	if bt.Rotation == 0 {
		t.Fatalf("guided torpedo stopped tracking last known point: rotation = %v", bt.Rotation)
	}
}

// TestHomingTurnPerTickScalesWithSpeedMultiplier 验证 1x/2x 全局速度下每秒转角换算成每拍转角后转弯半径一致。
// 场景：config.G 设为默认设置，SpeedMultiplier 分别为 SpeedStandardMultiplier=1.0 与 SpeedFastMultiplier=2.0，调用 homingTurnPerTick(180)。
// 预期：标准档返回 180/60×1=3°/拍，快档返回 180/60×2=6°/拍（容差 1e-9）。
// 覆盖：homingTurnPerTick 中 constants.MaxTPS=60 与 SpeedMultiplier 的换算，测试结束恢复 config.G。
func TestHomingTurnPerTickScalesWithSpeedMultiplier(t *testing.T) {
	oldSettings := config.G
	t.Cleanup(func() { config.G = oldSettings })

	config.G = config.NewDefaultGameSettings()
	config.G.SpeedMultiplier = config.SpeedStandardMultiplier
	standard := homingTurnPerTick(180)
	if math.Abs(standard-3) > 1e-9 {
		t.Fatalf("standard turn per tick = %v, want 3", standard)
	}

	config.G.SpeedMultiplier = config.SpeedFastMultiplier
	fast := homingTurnPerTick(180)
	if math.Abs(fast-6) > 1e-9 {
		t.Fatalf("fast turn per tick = %v, want 6", fast)
	}
}

// TestGuidedTorpedoHitsMovingTarget 验证目标持续机动脱离时制导鱼雷仍能追上并命中。
// 场景：初始布局同上，靶舰每拍 MoveTo 沿 (14,60) 航路向南加速脱离，最多 400 拍或弹药清空即止。
// 预期：target.CurHP 小于 100，即鱼雷在目标移动中完成追踪命中。
// 覆盖：guideBullet 逐拍更新 GuidePos 与纯追踪转向对机动目标的收敛。
func TestGuidedTorpedoHitsMovingTarget(t *testing.T) {
	m, target, _ := newHomingTestManager(true)
	mapCfg := m.state.Core.MissionMD.MapCfg
	waypoint := objPos.NewR(14, 60)

	for tick := 0; tick < 400 && len(m.state.Arena.ForwardingBullets) > 0; tick++ {
		target.MoveTo(mapCfg, waypoint, false)
		m.updateShotBullets()
	}

	if target.CurHP >= target.TotalHP {
		t.Fatalf("guided torpedo missed a moving target: HP = %.1f, want less than %.1f",
			target.CurHP, target.TotalHP)
	}
}

// TestLongBeachGuidedTorpedoHitsCrossingTarget 验证真实配置端到端：长滩的 Mk44 制导鱼雷能命中横穿目标。
// 场景：pearl_harbor 地图上，long_beach 在 (60,150)，maury 在 (63,150) 以 90° 航向驶向 (103,150)（40 格）；长滩禁用主炮/副炮/防空炮，仅靠 324mm Mk32 发射管（射程 16、Mk44 turnRate=150°/s、伤害 1400）开火，跑 600 拍。
// 预期：至少发射 1 条鱼雷，每条 IsGuided 且 TargetShipUid 指向 maury；maury.CurHP 低于开局值。
// 覆盖：shipFireTargets→FireAt 的发射链路把目标 uid、初始 GuidePos 写入实弹，并对制导鱼雷按 homingLifeRatio=1.5 放宽寿命。
func TestLongBeachGuidedTorpedoHitsCrossingTarget(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(1)
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := mapcfg.GetByName("pearl_harbor")
	gen := objUnit.NewShipUidGenerator(faction.HumanAlpha)
	shooter := objUnit.NewShip(gen, "long_beach", objPos.NewR(60, 150), 0, faction.HumanAlpha)
	genFoe := objUnit.NewShipUidGenerator(faction.ComputerAlpha)
	target := objUnit.NewShip(genFoe, "maury", objPos.NewR(63, 150), 90, faction.ComputerAlpha)
	shooter.Attack(target.Uid)
	shooter.DisableWeapon(objUnit.WeaponTypeMainGun)
	shooter.DisableWeapon(objUnit.WeaponTypeSecondaryGun)
	shooter.DisableWeapon(objUnit.WeaponTypeAntiAircraftGun)

	mm := &MissionManager{state: &state.MissionState{}}
	mm.state.Arena.Ships = map[string]*objUnit.BattleShip{shooter.Uid: shooter, target.Uid: target}
	mm.state.Arena.Planes = map[string]*objUnit.Plane{}
	mm.state.Core.MissionMD.MapCfg = mapCfg

	radians := 90 * math.Pi / 180
	waypoint := objPos.NewR(63+math.Sin(radians)*40, 150-math.Cos(radians)*40)
	startHP := target.CurHP
	fired := 0
	for tick := 0; tick < 600; tick++ {
		target.MoveTo(mapCfg, waypoint, false)
		for _, bullet := range shooter.FireAt(mm.shipFireTargets(shooter)) {
			if bullet.Type != objBullet.TypeTorpedo {
				continue
			}
			fired++
			if !bullet.IsGuided() || bullet.TargetShipUid != target.Uid {
				t.Fatalf("launched torpedo is not guided at target: %+v", bullet)
			}
			mm.state.Arena.ForwardingBullets = append(mm.state.Arena.ForwardingBullets, bullet)
		}
		mm.updateShotBullets()
	}

	if fired == 0 {
		t.Fatal("long beach fired no guided torpedoes")
	}
	if target.CurHP >= startHP {
		t.Fatalf("guided torpedoes (%d) all missed: HP = %.1f, want less than %.1f",
			fired, target.CurHP, startHP)
	}
}
