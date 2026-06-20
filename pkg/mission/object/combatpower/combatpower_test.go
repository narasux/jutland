package combatpower

import (
	"math"
	"testing"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

func TestExpectedDamageAndWeaponCycles(t *testing.T) {
	bullets := testBullets("test", 100, 0.1)

	if got, want := expectedDamage("test", bullets), 127.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("expectedDamage() = %v, want %v", got, want)
	}

	gun := &objUnit.Gun{BulletName: "test", BulletCount: 2, ReloadTime: 4}
	if got, want := gunDPS(gun, bullets), 63.5; math.Abs(got-want) > 1e-9 {
		t.Fatalf("gunDPS() = %v, want %v", got, want)
	}

	torpedo := &objUnit.TorpedoLauncher{
		BulletName: "test", BulletCount: 4, ReloadTime: 75, ShotInterval: 1,
	}
	if got, want := torpedoDPS(torpedo, bullets), 4*127.0/78; math.Abs(got-want) > 1e-9 {
		t.Fatalf("torpedoDPS() = %v, want %v", got, want)
	}

	rocket := &objUnit.RocketLauncher{
		BulletName: "test", RocketCount: 10, GroupCount: 3,
		ReloadTime: 5, ShotInterval: 0.2, GroupInterval: 1,
	}
	// 10 枚分为 4/4/2 三组：7 个组内间隔、2 个组间间隔。
	if got, want := shipRocketDPS(rocket, bullets), 10*127.0/8.4; math.Abs(got-want) > 1e-9 {
		t.Fatalf("shipRocketDPS() = %v, want %v", got, want)
	}

	planeRocket := &objUnit.PlaneRocketLauncher{BulletName: "test", RocketCount: 6}
	if got, want := planeRocketDPS(planeRocket, bullets), 6*127.0/evaluationWindow; math.Abs(got-want) > 1e-9 {
		t.Fatalf("planeRocketDPS() = %v, want %v", got, want)
	}

	releaser := &objUnit.Releaser{BulletName: "test"}
	if got, want := releaserDPS(releaser, bullets), 127.0/evaluationWindow; math.Abs(got-want) > 1e-9 {
		t.Fatalf("releaserDPS() = %v, want %v", got, want)
	}
}

func TestFiringArcCoverageMergesIntervals(t *testing.T) {
	tests := []struct {
		name        string
		left, right objUnit.FiringArc
		want        float64
	}{
		{
			name:  "full coverage",
			left:  objUnit.FiringArc{Start: 180, End: 360},
			right: objUnit.FiringArc{Start: 0, End: 180},
			want:  1,
		},
		{
			name:  "overlap",
			left:  objUnit.FiringArc{Start: 100, End: 300},
			right: objUnit.FiringArc{Start: 0, End: 200},
			want:  300.0 / 360,
		},
		{
			name:  "empty",
			left:  objUnit.FiringArc{Start: 360, End: 360},
			right: objUnit.FiringArc{Start: 0, End: 0},
			want:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firingArcCoverage(tt.left, tt.right); math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("firingArcCoverage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveHP(t *testing.T) {
	ship := &objUnit.BattleShip{
		TotalHP: 1000, HorizontalDamageReduction: 0.5, VerticalDamageReduction: 0.25,
	}
	if got, want := shipEHP(ship), 0.6*2000+0.4*(1000.0/0.75); math.Abs(got-want) > 1e-9 {
		t.Fatalf("shipEHP() = %v, want %v", got, want)
	}

	plane := &objUnit.Plane{TotalHP: 90, DamageReduction: 0.5}
	if got, want := planeEHP(plane), 60.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("planeEHP() = %v, want %v", got, want)
	}

	invulnerable := &objUnit.BattleShip{
		TotalHP: 1, HorizontalDamageReduction: 1, VerticalDamageReduction: 1,
	}
	if got := shipEHP(invulnerable); got != 1000 || math.IsInf(got, 0) || math.IsNaN(got) {
		t.Fatalf("shipEHP() for full reduction = %v, want finite 1000", got)
	}
}

func TestPlaneTargetRestriction(t *testing.T) {
	bullets := testBullets("gun", 10, 0)
	gun := &objUnit.Gun{
		BulletName: "gun", BulletCount: 1, ReloadTime: 1, Range: 10,
		AntiShip: true, AntiAircraft: true,
		LeftFiringArc:  objUnit.FiringArc{Start: 180, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
	}
	base := objUnit.Plane{
		TotalHP: 90, DamageReduction: 0.5,
		MaxSpeed: 500.0 / 5400, RotateSpeed: 12, Acceleration: 0.05, Range: 1500.0 / 14.4,
		Weapon: objUnit.PlaneWeapon{Guns: []*objUnit.Gun{gun}},
	}

	fighter := base
	fighter.Type = objUnit.PlaneTypeFighter
	fighterPower := CalculatePlane(&fighter, bullets)
	if fighterPower.AntiShip != 0 || fighterPower.AntiAir <= 0 {
		t.Fatalf("fighter power = %+v, want only anti-air power", fighterPower)
	}
	if fighterPower.Range <= 0 || fighterPower.Burst <= 0 || len(fighterPower.Details.AntiAirContributions) != 1 {
		t.Fatalf("fighter extended dimensions not populated: %+v", fighterPower)
	}

	bomber := base
	bomber.Type = objUnit.PlaneTypeDiveBomber
	bomberPower := CalculatePlane(&bomber, bullets)
	if bomberPower.AntiShip <= 0 || bomberPower.AntiAir != 0 {
		t.Fatalf("bomber power = %+v, want only anti-ship power", bomberPower)
	}
}

func TestCarrierAddsFullAirWingWithAvailabilityFactor(t *testing.T) {
	plane := &objUnit.Plane{
		DisplayName: "测试机", Range: 20,
		CombatPower: objUnit.CombatPowerInfo{
			Total: 85, AntiShip: 100, AntiAir: 50,
			Details: objUnit.CombatPowerDetails{
				AntiShipDPS: 10, AntiAirDPS: 5, BurstDamage: 100,
			},
		},
	}
	carrier := &objUnit.BattleShip{
		TotalHP:  1000,
		Aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "plane", MaxCount: 10}}},
	}

	power := CalculateShip(carrier, map[string]*objUnit.Plane{"plane": plane}, nil)
	if power.Hull != 0 || power.AntiShip != 700 || power.AntiAir != 350 {
		t.Fatalf("carrier power = %+v, want 70%% of ten-plane wing", power)
	}
	if power.Aviation != 595 || power.Total != 595 {
		t.Fatalf("carrier totals = %+v, want aviation and total 595", power)
	}
	if power.Range != 200 || power.Burst <= 0 || len(power.Details.BurstContributions) != 1 {
		t.Fatalf("carrier range, burst or contribution details missing: %+v", power)
	}
}

func TestPowerResultsAreFiniteAndNonNegative(t *testing.T) {
	bullets := testBullets("impact", 99999, 0.5)
	gun := &objUnit.Gun{
		BulletName: "impact", BulletCount: 3, ReloadTime: 0.001, Range: 1.5,
		AntiShip:       true,
		LeftFiringArc:  objUnit.FiringArc{Start: 180, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
	}
	ship := &objUnit.BattleShip{
		TotalHP: 1, HorizontalDamageReduction: 1, VerticalDamageReduction: 1,
		Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{gun}},
	}
	power := CalculateShip(ship, nil, bullets)
	for name, value := range map[string]int{
		"total": power.Total, "antiShip": power.AntiShip, "antiAir": power.AntiAir,
		"survival": power.Survival, "mobility": power.Mobility,
	} {
		if value < 0 {
			t.Fatalf("%s = %d, want non-negative", name, value)
		}
	}
	if power.Total == 0 {
		t.Fatalf("power = %+v, want armed unit to have combat power", power)
	}

	if got := CalculatePlane(&objUnit.Plane{}, nil); got.Total != 0 {
		t.Fatalf("empty plane power = %+v, want zero total", got)
	}
}

func TestWeightedTotalKeepsArmedUnitVisible(t *testing.T) {
	if got := weightedTotal(0, 1); got != 1 {
		t.Fatalf("weightedTotal(0, 1) = %d, want minimum visible power 1", got)
	}
	if got := weightedTotal(0, 0); got != 0 {
		t.Fatalf("weightedTotal(0, 0) = %d, want 0", got)
	}
}

func testBullets(name string, damage, criticalRate float64) map[string]*objBullet.Bullet {
	return map[string]*objBullet.Bullet{
		name: {Name: name, Damage: damage, CriticalRate: criticalRate},
	}
}
