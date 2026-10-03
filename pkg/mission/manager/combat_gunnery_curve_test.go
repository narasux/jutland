package manager

import (
	"math"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// 改造前，直射 / 曲射在固定射程百分比上硬切换，命中率会从 100% 掉到 45%；
// 现在落角连续过渡，原交界点两侧的命中率必须接近，且整体随距离单调下降。
const gunneryCurveBullet = "test/gunnery-curve-shell"

func gunneryCurveSetup(t *testing.T) *objUnit.Gun {
	t.Helper()
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	objUnit.SetSimTick(1)
	t.Cleanup(func() { config.G = oldSettings })

	objBullet.Map[gunneryCurveBullet] = &objBullet.Bullet{
		Name: gunneryCurveBullet, Type: objBullet.TypeShell,
		Diameter: 406, Damage: 100, CriticalRate: 0,
	}
	t.Cleanup(func() { delete(objBullet.Map, gunneryCurveBullet) })

	return &objUnit.Gun{
		Name: "test-gunnery-curve", BulletName: gunneryCurveBullet,
		BulletCount: 1, ReloadTime: 0,
		Range: 19, BulletSpread: 150, BulletSpeed: 0.2,
		AntiShip: true, PosPercent: 0.4,
		LeftFiringArc:  objUnit.FiringArc{Start: 180, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
	}
}

func newGunneryShooter() *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: "shooter", Length: 270, Width: 33,
		CurPos: objPos.NewR(5, 24), CurRotation: 90, CurSpeed: 0,
		MaxSpeed: 0.03, TotalHP: 1e9, CurHP: 1e9, BelongPlayer: faction.HumanAlpha,
	}
}

// crossTarget 是一条横穿射击线的战列舰（前往 -Y 方向），每拍推进一次。
func newCrossTarget(x, y float64) *objUnit.BattleShip {
	return &objUnit.BattleShip{
		Uid: "target", Length: 270, Width: 33,
		CurPos: objPos.NewR(x, y), CurRotation: 0, CurSpeed: 33.0 / 1200,
		MaxSpeed: 33.0 / 1200, Acceleration: 0.3, RotateSpeed: 0.85,
		TotalHP: 1e9, CurHP: 1e9, BelongPlayer: faction.ComputerAlpha,
	}
}

func stepCrossTarget(ship *objUnit.BattleShip) {
	ship.CurPos.AddRx(math.Sin(ship.CurRotation*math.Pi/180) * ship.CurSpeed)
	ship.CurPos.SubRy(math.Cos(ship.CurRotation*math.Pi/180) * ship.CurSpeed)
}

// gunneryHitRate 在指定射程百分比上开火 shots 次，返回命中率。
func gunneryHitRate(t *testing.T, gun *objUnit.Gun, rangePercent float64, shots int) float64 {
	t.Helper()
	shooter := newGunneryShooter()
	targetX := 5 + 19*rangePercent

	hits := 0
	for i := 0; i < shots; i++ {
		gun.ReloadStartTick = 0
		target := newCrossTarget(targetX, 24)
		bullets := gun.Fire(shooter, target)
		if len(bullets) == 0 {
			t.Fatalf("no bullet fired at rangePercent=%v", rangePercent)
		}
		mm := &MissionManager{state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships:             map[string]*objUnit.BattleShip{target.Uid: target},
				ForwardingBullets: bullets,
			},
		}}
		startHP := target.CurHP
		for tick := 0; tick < 900 && len(mm.state.Arena.ForwardingBullets) > 0; tick++ {
			mm.updateShotBullets()
			stepCrossTarget(target)
		}
		if target.CurHP < startHP {
			hits++
		}
	}
	return float64(hits) / float64(shots)
}

// 原交界点（0.65R）两侧的命中率必须接近：这是本次改造的核心验收标准。
func TestHitChanceHasNoCliffAtFormerSwitchRange(t *testing.T) {
	gun := gunneryCurveSetup(t)

	const shots = 600
	below := gunneryHitRate(t, gun, 0.62, shots)
	above := gunneryHitRate(t, gun, 0.68, shots)
	if math.Abs(below-above) > 0.25 {
		t.Fatalf(
			"hit chance jumps across the former switch range: 0.62R=%.3f 0.68R=%.3f",
			below, above,
		)
	}
	// 两侧都必须明显低于"必中"，否则说明危险界没有生效
	if above > 0.9 {
		t.Fatalf("hit chance at 0.68R = %.3f, danger space should cap long range accuracy", above)
	}
}

// 近距离仍是平射（危险界不限制），命中率随距离单调不增。
func TestHitChanceFallsWithRange(t *testing.T) {
	gun := gunneryCurveSetup(t)

	const shots = 600
	close := gunneryHitRate(t, gun, 0.40, shots)
	mid := gunneryHitRate(t, gun, 0.65, shots)
	far := gunneryHitRate(t, gun, 0.95, shots)
	if close < 0.95 {
		t.Fatalf("close range hit chance = %.3f, flat fire should still be near certain", close)
	}
	if !(close > mid && mid >= far) {
		t.Fatalf("hit chance should not rise with range: %.3f %.3f %.3f", close, mid, far)
	}
	if far > 0.9 {
		t.Fatalf("far hit chance = %.3f, danger space should cap long range accuracy", far)
	}
}

// 落角随射程变陡、危险界随之收窄：炮弹自身的参数必须符合这条规律。
func TestFiredShellPlungeFollowsRange(t *testing.T) {
	gun := gunneryCurveSetup(t)
	shooter := newGunneryShooter()

	plungeAt := func(rangePercent float64) float64 {
		gun.ReloadStartTick = 0
		target := newCrossTarget(5+19*rangePercent, 24)
		bullets := gun.Fire(shooter, target)
		if len(bullets) == 0 {
			t.Fatalf("no bullet fired at rangePercent=%v", rangePercent)
		}
		return bullets[0].Plunge
	}
	if got := plungeAt(0.30); got != 0 {
		t.Fatalf("plunge at 0.30R = %v, want flat fire", got)
	}
	near, mid, far := plungeAt(0.50), plungeAt(0.65), plungeAt(0.95)
	if !(near < mid && mid < far) {
		t.Fatalf("plunge should rise with range: %.3f %.3f %.3f", near, mid, far)
	}
	if far != 1 {
		t.Fatalf("plunge at 0.95R = %v, want fully plunging", far)
	}
	_ = object.TypeShip
}
