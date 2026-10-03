package combatpower

import (
	"math"
	"strconv"
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

func TestAntiShipAndAntiAirIgnoreHullAndMobility(t *testing.T) {
	bullets := testBullets("gun", 100, 0)
	gun := &objUnit.Gun{
		BulletName: "gun", BulletCount: 2, ReloadTime: 2, Range: 10,
		AntiShip: true, AntiAircraft: true,
		LeftFiringArc:  objUnit.FiringArc{Start: 180, End: 360},
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
	}
	// 两艘船使用完全相同的武器，只有舰体生存和机动不同。
	frail := &objUnit.BattleShip{
		TotalHP: 1, HorizontalDamageReduction: 1, VerticalDamageReduction: 1,
		MaxSpeed: 1000, RotateSpeed: 360, Acceleration: 25,
		Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{gun}},
	}
	tough := &objUnit.BattleShip{
		TotalHP:  100000,
		MaxSpeed: 50, RotateSpeed: 2, Acceleration: 1,
		Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{gun}},
	}
	frailPower, toughPower := CalculateShip(frail, nil, bullets), CalculateShip(tough, nil, bullets)

	if frailPower.AntiShip != toughPower.AntiShip || frailPower.AntiAir != toughPower.AntiAir {
		t.Fatalf(
			"identical weapons must give identical anti-ship / anti-air: %+v vs %+v",
			frailPower, toughPower,
		)
	}
	effectiveDPS := gunDPS(gun, bullets) * gunEffectiveness(gun, bullets, false, false)
	if got, want := frailPower.AntiShip, firepowerScore(effectiveDPS); got != want {
		t.Fatalf("anti-ship = %d, want sqrt(effective DPS) = %d", got, want)
	}
	// 舰体差异不能一起被抹平：生存和综合战力仍要区分两者。
	if frailPower.Survival == toughPower.Survival || frailPower.Total == toughPower.Total {
		t.Fatalf("hull differences must stay visible: %+v vs %+v", frailPower, toughPower)
	}
}

func TestPlaneFormationAndWeaponCapabilities(t *testing.T) {
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
	if fighterPower.FormationSize != planeFormationSize {
		t.Fatalf("fighter formation size = %d, want %d", fighterPower.FormationSize, planeFormationSize)
	}
	if fighterPower.AntiShip != 0 || fighterPower.AntiAir <= 0 {
		t.Fatalf("fighter power = %+v, want only anti-air power", fighterPower)
	}
	wantEHP := planeEHP(&fighter) * planeFormationSize
	if math.Abs(fighterPower.Details.EffectiveHP-wantEHP) > 1e-9 {
		t.Fatalf("fighter formation EHP = %v, want %v", fighterPower.Details.EffectiveHP, wantEHP)
	}
	wantAntiAirDPS := gunDPS(gun, bullets) * gunEffectiveness(gun, bullets, true, true) * planeFormationSize
	if math.Abs(fighterPower.Details.AntiAirDPS-wantAntiAirDPS) > 1e-9 {
		t.Fatalf("fighter formation anti-air DPS = %v, want %v", fighterPower.Details.AntiAirDPS, wantAntiAirDPS)
	}
	if got, want := fighterPower.Survival, nonNegativeRound(math.Sqrt(wantEHP)); got != want {
		t.Fatalf("fighter survival = %d, want %d", got, want)
	}
	if fighterPower.Projection <= 0 || fighterPower.Burst <= 0 || len(fighterPower.Details.AntiAirContributions) != 1 {
		t.Fatalf("fighter extended dimensions not populated: %+v", fighterPower)
	}
	if got := fighterPower.Details.AntiAirContributions[0].Value; math.Abs(got-wantAntiAirDPS) > 1e-9 {
		t.Fatalf("fighter contribution = %v, want %v", got, wantAntiAirDPS)
	}
	if fighterPower.Details.MaxProjectionDistanceKM != 1500 {
		t.Fatalf("fighter projection distance = %.0f km, want 1500", fighterPower.Details.MaxProjectionDistanceKM)
	}

	bomber := base
	bomber.Type = objUnit.PlaneTypeDiveBomber
	bomberPower := CalculatePlane(&bomber, bullets)
	if bomberPower.AntiShip <= 0 || bomberPower.AntiAir <= 0 {
		t.Fatalf("bomber power = %+v, want weapon-based anti-ship and anti-air power", bomberPower)
	}

	torpedoBomber := base
	torpedoBomber.Type = objUnit.PlaneTypeTorpedoBomber
	torpedoBomberPower := CalculatePlane(&torpedoBomber, bullets)
	if torpedoBomberPower.AntiShip <= 0 || torpedoBomberPower.AntiAir <= 0 {
		t.Fatalf("torpedo bomber power = %+v, want weapon-based anti-ship and anti-air power", torpedoBomberPower)
	}
}

func TestCarrierScalesStandardFormationByAircraftCount(t *testing.T) {
	plane := &objUnit.Plane{
		Name: "plane", Range: 20,
		CombatPower: objUnit.CombatPowerInfo{
			FormationSize: 10, Total: 85, AntiShip: 100, AntiAir: 50,
			Details: objUnit.CombatPowerDetails{
				AntiShipDPS: 10, AntiAirDPS: 5, BurstDamage: 100,
				AntiShipThreat: 100, AntiAirThreat: 50,
			},
		},
	}
	tests := []struct {
		count                   int64
		wantShip, wantAir       int
		wantAviation, wantTotal int
		wantDPS                 float64
	}{
		{count: 5, wantShip: 35, wantAir: 18, wantAviation: 30, wantTotal: 30, wantDPS: 3.5},
		{count: 10, wantShip: 70, wantAir: 35, wantAviation: 60, wantTotal: 60, wantDPS: 7},
		{count: 20, wantShip: 140, wantAir: 70, wantAviation: 119, wantTotal: 119, wantDPS: 14},
	}

	for _, tt := range tests {
		t.Run(strconv.FormatInt(tt.count, 10), func(t *testing.T) {
			carrier := &objUnit.BattleShip{
				TotalHP:  1000,
				Aircraft: objUnit.ShipAircraft{Groups: []objUnit.PlaneGroup{{Name: "plane", MaxCount: tt.count}}},
			}
			power := CalculateShip(carrier, map[string]*objUnit.Plane{"plane": plane}, nil)
			if power.Hull != 0 || power.AntiShip != tt.wantShip || power.AntiAir != tt.wantAir {
				t.Fatalf("carrier power = %+v, want anti-ship %d and anti-air %d", power, tt.wantShip, tt.wantAir)
			}
			if power.Aviation != tt.wantAviation || power.Total != tt.wantTotal {
				t.Fatalf("carrier totals = %+v, want aviation %d and total %d", power, tt.wantAviation, tt.wantTotal)
			}
			if math.Abs(power.Details.AntiShipDPS-tt.wantDPS) > 1e-9 {
				t.Fatalf("carrier anti-ship DPS = %v, want %v", power.Details.AntiShipDPS, tt.wantDPS)
			}
			// 投送只取舰船自身武器射程：这艘航母没有舰炮，舰载机航程不得并入舰船投送。
			if power.Projection != 0 || power.Details.MaxProjectionDistanceKM != 0 {
				t.Fatalf(
					"carrier projection = %d / %.0f km, want 0: aircraft range must not leak into ship projection",
					power.Projection, power.Details.MaxProjectionDistanceKM,
				)
			}
			if power.Burst <= 0 || len(power.Details.BurstContributions) != 1 {
				t.Fatalf("carrier burst or contribution details missing: %+v", power)
			}
			contribution := power.Details.AntiShipContributions[0]
			if contribution.Name != "plane" || contribution.Count != tt.count {
				t.Fatalf("carrier contribution = %+v, want stable plane ID and count %d", contribution, tt.count)
			}
		})
	}
}

// TestShipProjectionUsesWeaponRangeOnly 锁定舰船投送射程的单位口径。
// 舰船投送 = 自身武器射程（配置值/2 后的地图格），不并入舰载机航程（配置值/8 后的地图格）。
// 两者混用会让挂着远程侦察机的舰船投送从 9~21 跳到 200+，把船只费用抬高一个量级。
func TestShipProjectionUsesWeaponRangeOnly(t *testing.T) {
	bullets := testBullets("gun", 100, 0)
	newShip := func(gunRange float64) *objUnit.BattleShip {
		gun := &objUnit.Gun{
			BulletName: "gun", BulletCount: 1, ReloadTime: 1, Range: gunRange,
			AntiShip: true, AntiAircraft: true,
			LeftFiringArc:  objUnit.FiringArc{Start: 180, End: 360},
			RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
		}
		return &objUnit.BattleShip{
			TotalHP: 1000, MaxSpeed: 100, RotateSpeed: 10, Acceleration: 10,
			Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{gun}},
		}
	}
	// 无武装远程侦察机：不产生 aviation，航程（地图格）远超舰炮射程。
	scout := &objUnit.Plane{Name: "scout", Type: objUnit.PlaneTypeScout, Range: 261.125}
	planes := map[string]*objUnit.Plane{"scout": scout}

	shortPower := CalculateShip(newShip(9), planes, bullets)
	longPower := CalculateShip(newShip(21), planes, bullets)
	withScout := newShip(9)
	withScout.Aircraft = objUnit.ShipAircraft{
		Groups: []objUnit.PlaneGroup{{Name: "scout", MaxCount: 2}},
	}
	scoutPower := CalculateShip(withScout, planes, bullets)

	if got, want := shortPower.Projection, projectionScore(9); got != want {
		t.Fatalf("short-range ship projection = %d, want %d", got, want)
	}
	if longPower.Projection <= shortPower.Projection {
		t.Fatalf(
			"projection must follow weapon range: %d (range 21) vs %d (range 9)",
			longPower.Projection, shortPower.Projection,
		)
	}
	if scoutPower.Projection != shortPower.Projection {
		t.Fatalf(
			"scout aircraft must not change ship projection: %d with scout vs %d without",
			scoutPower.Projection, shortPower.Projection,
		)
	}
	if got, want := scoutPower.Details.MaxProjectionRange, 9.0; got != want {
		t.Fatalf("ship MaxProjectionRange = %v, want %v", got, want)
	}
	wantKM := 9 * shipProjectionKilometersPerMapUnit
	if got := scoutPower.Details.MaxProjectionDistanceKM; got != wantKM {
		t.Fatalf("ship projection distance = %v km, want %v km", got, wantKM)
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

func TestCaliberEffectivenessDiscountsSmallShells(t *testing.T) {
	// 未知口径不做折算，127mm 及以上视为满值。
	if got := caliberEffectiveness(0); got != 1 {
		t.Fatalf("caliberEffectiveness(0) = %v, want 1", got)
	}
	if got := caliberEffectiveness(127); got != 1 {
		t.Fatalf("caliberEffectiveness(127) = %v, want 1", got)
	}
	if got := caliberEffectiveness(460); got != 1 {
		t.Fatalf("caliberEffectiveness(460) = %v, want 1", got)
	}
	// 小口径按幂次衰减，且不超过下限。
	if got, want := caliberEffectiveness(25), math.Pow(25.0/127, caliberExponent); math.Abs(got-want) > 1e-9 {
		t.Fatalf("caliberEffectiveness(25) = %v, want %v", got, want)
	}
	if got := caliberEffectiveness(1); got != caliberFloor {
		t.Fatalf("caliberEffectiveness(1) = %v, want floor %v", got, caliberFloor)
	}
	if !(caliberEffectiveness(25) < caliberEffectiveness(76) &&
		caliberEffectiveness(76) < caliberEffectiveness(127)) {
		t.Fatal("caliber effectiveness should increase with caliber")
	}
}

func TestGunEffectivenessAppliesCaliberToAntiShipOnly(t *testing.T) {
	bullets := map[string]*objBullet.Bullet{
		"mg":   {Name: "mg", Damage: 2, Diameter: 25},
		"main": {Name: "main", Damage: 400, Diameter: 200},
	}
	mount := func(bulletName string) *objUnit.Gun {
		return &objUnit.Gun{
			BulletName: bulletName, BulletCount: 1, ReloadTime: 5, Range: 10, BulletSpread: 50,
			LeftFiringArc:  objUnit.FiringArc{Start: 180, End: 360},
			RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
		}
	}
	mg, main := gunEffectiveness(mount("mg"), bullets, false, false), gunEffectiveness(mount("main"), bullets, false, false)
	if want := main * caliberEffectiveness(25); math.Abs(mg-want) > 1e-9 {
		t.Fatalf("small caliber anti-ship effectiveness = %v, want %v", mg, want)
	}
	if mg >= main {
		t.Fatalf("small caliber anti-ship effectiveness %v should be below main gun %v", mg, main)
	}
	// 对空不吃口径惩罚，两种口径只差基础命中率之外的参数。
	if got, want := gunEffectiveness(mount("mg"), bullets, true, false),
		gunEffectiveness(mount("main"), bullets, true, false); math.Abs(got-want) > 1e-9 {
		t.Fatalf("anti-air effectiveness should not depend on caliber: %v vs %v", got, want)
	}
}

func testBullets(name string, damage, criticalRate float64) map[string]*objBullet.Bullet {
	return map[string]*objBullet.Bullet{
		name: {Name: name, Damage: damage, CriticalRate: criticalRate},
	}
}
