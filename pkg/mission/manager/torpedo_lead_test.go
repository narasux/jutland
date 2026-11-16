package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

// torpedoDuel 让一艘驱逐舰对着正右舷 3 格的横穿目标打一轮鱼雷，只保留鱼雷武器，
// 返回发射根数与目标掉血。目标用真实 MoveTo 推进，命中走真实 updateShotBullets。
func torpedoDuel(t *testing.T, course float64, presetCruise bool) (fired, damage float64) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(1)
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := mapcfg.GetByName("pearl_harbor")
	gen := objUnit.NewShipUidGenerator(faction.HumanAlpha)
	shooter := objUnit.NewShip(gen, "porter", objPos.NewR(60, 150), 0, faction.HumanAlpha)
	genFoe := objUnit.NewShipUidGenerator(faction.ComputerAlpha)
	target := objUnit.NewShip(genFoe, "maury", objPos.NewR(63, 150), course, faction.ComputerAlpha)
	if presetCruise {
		target.CurSpeed = target.CruiseSpeed()
	}
	shooter.Attack(target.Uid)
	shooter.DisableWeapon(objUnit.WeaponTypeMainGun)
	shooter.DisableWeapon(objUnit.WeaponTypeSecondaryGun)
	shooter.DisableWeapon(objUnit.WeaponTypeAntiAircraftGun)

	mm := &MissionManager{state: &state.MissionState{}}
	mm.state.Arena.Ships = map[string]*objUnit.BattleShip{shooter.Uid: shooter, target.Uid: target}
	mm.state.Arena.Planes = map[string]*objUnit.Plane{}
	mm.state.UI.GameMarks = map[objMark.ID]*objMark.Mark{}
	mm.state.Core.MissionMD.MapCfg = mapCfg

	radians := course * math.Pi / 180
	waypoint := objPos.NewR(63+math.Sin(radians)*40, 150-math.Cos(radians)*40)
	startHP := target.CurHP
	for tick := 0; tick < 400; tick++ {
		target.MoveTo(mapCfg, waypoint, false)
		for _, bullet := range shooter.FireAt(mm.shipFireTargets(shooter)) {
			if bullet.Type == objBullet.TypeTorpedo {
				fired++
			}
			mm.state.Arena.ForwardingBullets = append(mm.state.Arena.ForwardingBullets, bullet)
		}
		mm.updateShotBullets()
	}
	return fired, startHP - target.CurHP
}

// TestTorpedoHitsCrossingTargetGettingUnderWay 验证目标刚起步加速时鱼雷提前量会把加速度算进去。
// 场景：porter 在 (60,150) 只用鱼雷打 3 格外的 maury，目标航向 0°/180°、初始航速 0，400 拍内持续加速横穿。
// 预期：至少发射 1 根鱼雷且目标掉血 >0；回归前提前量只取发射瞬间航速，正横与反向横穿都是 0 伤害。
// 覆盖：真实 porter/maury 模板 + MoveTo 推进，经 shipFireTargets/FireAt/updateShotBullets 统计发射根数与掉血。
func TestTorpedoHitsCrossingTargetGettingUnderWay(t *testing.T) {
	for _, course := range []float64{0, 180} {
		fired, damage := torpedoDuel(t, course, false)
		if fired == 0 {
			t.Fatalf("航向 %.0f: 一发鱼雷都没打出去", course)
		}
		if damage <= 0 {
			t.Fatalf("航向 %.0f: 起步中的横穿目标 %v 根鱼雷全部脱靶", course, fired)
		}
	}
}

// TestTorpedoHitsCruisingCrossingTarget 验证已处于巡航速度的横穿目标同样能被鱼雷提前量命中。
// 场景：与起步用例相同，但 maury 直接设为 CruiseSpeed，航向 0°/180°、在 3 格外横穿射击线。
// 预期：至少 1 根鱼雷发射且目标掉血 >0；固定巡航速度这一稳定分支不能被起步加速修复漏掉。
// 覆盖：torpedoDuel(presetCruise=true) 走真实 MoveTo 与 updateShotBullets，断言 fired、damage 两个返回值。
func TestTorpedoHitsCruisingCrossingTarget(t *testing.T) {
	for _, course := range []float64{0, 180} {
		fired, damage := torpedoDuel(t, course, true)
		if fired == 0 || damage <= 0 {
			t.Fatalf("航向 %.0f: 巡航横穿目标 %v 根鱼雷，掉血 %.0f", course, fired, damage)
		}
	}
}

// TestMultiMountTorpedoVolleyCoversClosingTarget 验证多座发射器每一轮齐射都至少有一根鱼雷压在瞄准线上。
// 场景：porter 双座四联装对沿瞄准线迎头冲来的 maury 打完整轮，目标在正东 3 格、航向 270°、巡航速度，共 240 拍。
// 预期：发射 >0 且命中 ≥3；回归前第二轮起整轮落在瞄准线外侧，8 根齐射只中 1 根，正头对进的目标从缺口穿过。
// 覆盖：torpedoClosingDuel 用真实 MoveTo/FireAt/updateShotBullets，按 HitObjType==TypeShip 统计命中根数。
func TestMultiMountTorpedoVolleyCoversClosingTarget(t *testing.T) {
	fired, hits := torpedoClosingDuel(t)
	if fired == 0 {
		t.Fatal("一发鱼雷都没打出去")
	}
	if hits < 3 {
		t.Fatalf("%d 根齐射只命中 %d 根，瞄准线上没有每轮都留鱼雷", fired, hits)
	}
}

// torpedoClosingDuel 让一艘多座鱼雷驱逐舰对着沿瞄准线迎头冲来的目标打满一轮鱼雷，
// 返回发射根数与命中根数。目标用真实 MoveTo 推进，命中走真实 updateShotBullets。
func torpedoClosingDuel(t *testing.T) (fired, hits int) {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(1)
	t.Cleanup(func() { config.G = oldSettings })

	mapCfg := mapcfg.GetByName("pearl_harbor")
	gen := objUnit.NewShipUidGenerator(faction.HumanAlpha)
	shooter := objUnit.NewShip(gen, "porter", objPos.NewR(60, 150), 0, faction.HumanAlpha)
	genFoe := objUnit.NewShipUidGenerator(faction.ComputerAlpha)
	// 目标在正东 3 格、航向正西，舰体长轴正对鱼雷扇面。
	target := objUnit.NewShip(genFoe, "maury", objPos.NewR(63, 150), 270, faction.ComputerAlpha)
	target.CurSpeed = target.CruiseSpeed()
	shooter.Attack(target.Uid)
	shooter.DisableWeapon(objUnit.WeaponTypeMainGun)
	shooter.DisableWeapon(objUnit.WeaponTypeSecondaryGun)
	shooter.DisableWeapon(objUnit.WeaponTypeAntiAircraftGun)

	mm := &MissionManager{state: &state.MissionState{}}
	mm.state.Arena.Ships = map[string]*objUnit.BattleShip{shooter.Uid: shooter, target.Uid: target}
	mm.state.Arena.Planes = map[string]*objUnit.Plane{}
	mm.state.UI.GameMarks = map[objMark.ID]*objMark.Mark{}
	mm.state.Core.MissionMD.MapCfg = mapCfg

	launched := []*objBullet.Bullet{}
	waypoint := objPos.NewR(23, 150)
	for tick := 0; tick < 240; tick++ {
		// 装填 / 发射间隔按拍计算，模拟主循环每拍推进一次时钟。
		objUnit.SetSimTick(int64(tick + 2))
		target.MoveTo(mapCfg, waypoint, false)
		for _, bullet := range shooter.FireAt(mm.shipFireTargets(shooter)) {
			if bullet.Type == objBullet.TypeTorpedo {
				launched = append(launched, bullet)
			}
			mm.state.Arena.ForwardingBullets = append(mm.state.Arena.ForwardingBullets, bullet)
		}
		mm.updateShotBullets()
	}
	for _, bullet := range launched {
		if bullet.HitObjType == object.TypeShip {
			hits++
		}
	}
	return len(launched), hits
}
