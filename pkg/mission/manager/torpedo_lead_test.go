package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
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
