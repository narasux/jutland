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

// 目标刚接到命令、还在起步加速时，鱼雷提前量必须把这段加速算进去。
// 只取发射瞬间的航速会把落点算近：横穿的目标会被整条扇面甩在身后，直行的目标
// 反而永远打不到。回归：修复前正横与反向横穿两个场景都是 0 伤害。
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

// 已经在巡航的横穿目标同样要能打中。
func TestTorpedoHitsCruisingCrossingTarget(t *testing.T) {
	for _, course := range []float64{0, 180} {
		fired, damage := torpedoDuel(t, course, true)
		if fired == 0 || damage <= 0 {
			t.Fatalf("航向 %.0f: 巡航横穿目标 %v 根鱼雷，掉血 %.0f", course, fired, damage)
		}
	}
}

// 沿瞄准线迎头冲来的目标，舰体长轴正对扇面，只有压在瞄准线上的鱼雷有机会命中；
// 每一轮齐射都必须有一根压在瞄准线上，不能只有第一轮。
// 回归：多座发射器按整组航迹顺次外扩时，第二轮起整轮都落在瞄准线外侧，
// porter 的 8 根齐射只命中 1 根，正头对进的目标正好从缺口里穿过去。
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
