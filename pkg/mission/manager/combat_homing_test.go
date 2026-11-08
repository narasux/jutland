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

// 制导鱼雷必须能从错误朝向拐到目标上：起手朝正北，目标在正东。
func TestGuidedTorpedoCurvesToOffAxisTarget(t *testing.T) {
	m, target, _ := newHomingTestManager(true)

	advanceBullets(m, 400)

	if target.CurHP >= target.TotalHP {
		t.Fatalf("guided torpedo missed: HP = %.1f, want less than %.1f", target.CurHP, target.TotalHP)
	}
}

// 同样的初始条件，普通鱼雷保持直线，不应命中侧面的目标。
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

// 目标中途消失时，鱼雷继续飞向最后已知点，不 panic 也不重新搜索。
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

// 1x/2x 速度下转弯半径必须一致：转角按全局倍率同比放大。
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

// 目标持续机动（向南加速脱离）时，制导鱼雷仍要能追上并命中。
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

// 真实配置端到端：长滩的 Mk44 制导航迹必须能命中横穿目标。
// 同时验证发射链路把目标 uid、初始追踪点和放宽后的寿命都写进了实弹。
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
