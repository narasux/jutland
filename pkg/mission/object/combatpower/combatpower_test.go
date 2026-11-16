package combatpower

import (
	"math"
	"strconv"
	"testing"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// TestExpectedDamageAndWeaponCycles 验证暴击期望伤害与火炮/鱼雷/火箭/投放器的 DPS 周期换算。
// 场景：模板 Damage=100、暴击率 0.1 得期望 127；构造 2 发/4s 炮、4 发/75s 鱼雷、10 发三组火箭等。
// 预期：炮 63.5；鱼雷按 75+3*1=78s 周期；舰载火箭按 5+7*0.2+2*1=8.4s；飞机挂载按 60s 窗口。
// 覆盖：六处 1e-9 精度断言，覆盖暴击系数、连发间隔、组间间隔与评估窗口。
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

// TestFiringArcCoverageMergesIntervals 验证左右射界按角度并集计算，重叠区不重复计数。
// 场景：表驱动三组：180-360 与 0-180 首尾相接；100-300 与 0-200 半重叠；360-360 与 0-0 为空。
// 预期：覆盖率分别为 1、300/360、0；失败意味着重叠角被重复累计或空射界产生非零。
// 覆盖：每组 t.Run 以 1e-9 容差断言 firingArcCoverage，覆盖拼接、重叠、零长度三种边界。
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

// TestEffectiveHP 验证舰船/飞机的有效生命公式与完全免伤时的固定上限。
// 场景：舰 1000HP、水平减伤 0.5、垂直 0.25；飞机 90HP、减伤 0.5；另取双 100% 减伤的 1HP 单位。
// 预期：舰 EHP=0.6*2000+0.4*1333.33≈1733.33；飞机=90/(3*0.5)=60；免伤取 10000000 且有限。
// 覆盖：三组容差断言加 !Inf/!NaN 检查，锁死 6:4 权重、飞机三倍受伤与免伤分支。
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
	// 完全免疫单位不吃伤害，生存值取固定的最高档，而不是随 1 点 HP 缩水的伪 EHP。
	if got := shipEHP(invulnerable); got != invulnerableEHP || math.IsInf(got, 0) || math.IsNaN(got) {
		t.Fatalf("shipEHP() for full reduction = %v, want finite %d", got, invulnerableEHP)
	}
	if got := planeEHP(&objUnit.Plane{TotalHP: 90, DamageReduction: 1}); got != invulnerableEHP {
		t.Fatalf("planeEHP() for full reduction = %v, want %d", got, invulnerableEHP)
	}
}

// TestAntiShipAndAntiAirIgnoreHullAndMobility 验证火力轴只由武器决定，生存与机动另行计分。
// 场景：同一门 2 发/2s、射程 10 的双用途炮，分别装到 1HP 全免伤高速舰与 100000HP 低速舰。
// 预期：两舰 AntiShip/AntiAir 完全相等，AntiShip=firepowerScore(60)=8；Survival/Total 仍不同。
// 覆盖：比较两舰火力轴与生存/总战力，并用 gunDPS*gunEffectiveness=60 校验 firepowerScore。
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
	// frail 是 100% 免伤，生存值取最高档；tough 只是高 HP，两者仍应不同。
	if frailPower.Survival == toughPower.Survival || frailPower.Total == toughPower.Total {
		t.Fatalf("hull differences must stay visible: %+v vs %+v", frailPower, toughPower)
	}
}

// TestPlaneFormationAndWeaponCapabilities 验证飞机按 10 机编队放大，并区分战斗机与攻击机火力。
// 场景：基座 90HP、减伤 0.5、1 发/1s 双用途炮、航程 1500，分别设为战斗机、俯冲轰炸机、鱼雷机。
// 预期：战斗机编队 EHP=600、对空 DPS≈13.17、Survival=24、投送 1500km；攻击机保留对舰对空。
// 覆盖：校验编队 EHP/DPS、Survival、Projection/Burst、单条对空贡献，并断言战斗机清空对舰。
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
		MaxSpeed: 500.0 / 5400, RotateSpeed: 12, Acceleration: 0.05,
		Range:  1500.0 / objUnit.PlaneRangeDisplayScale,
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

// TestCarrierScalesStandardFormationByAircraftCount 验证航母按载机数缩放标准编队的航空战力。
// 场景：飞机 FormationSize=10、AntiShip=100、AntiAir=50，航母无舰炮；载机数分别取 5、10、20。
// 预期：AntiShip 35/70/140，AntiAir 18/35/70，Aviation 与 Total 30/60/119，DPS 3.5/7/14。
// 覆盖：子测试断言三组火力/总战力/DPS、Hull 与投送为 0，且 plane 贡献的 Count 等于载机数。
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

// TestShipProjectionUsesWeaponRangeOnly 验证舰船投送只取自身武器射程，不并入舰载机航程。
// 场景：战舰挂远程侦察机时，若把飞机航程算进投送，投送会从 9~21 格跳到 200+，费用被抬高一个量级。
// 预期：炮射程 9 的船投送 == projectionScore(9)；射程 21 的船投送更大；挂侦察机不改变射程 9 船的投送。
// 覆盖：构造短程/远程/带侦察机三艘船，调用 CalculateShip 比较 Projection。
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

// TestPowerResultsAreFiniteAndNonNegative 验证极端参数下战力结果有限、非负且不会归零。
// 场景：弹药 Damage=99999、暴击率 0.5（期望 234997.65），3 发/0.001s、射程 1.5，装在 1HP 免伤舰上。
// 预期：Total/AntiShip/AntiAir/Survival/Mobility 均 ≥0 且 Total>0；空飞机 CalculatePlane 返回 0。
// 覆盖：遍历 5 个整数字段查负值，另断言武装单位总战力非零与空对象零值。
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

// TestWeightedTotalKeepsArmedUnitVisible 验证加权取整归零时保留最小战力 1。
// 场景：weightedTotal(0,1) 只有 1 分对空；weightedTotal(0,0) 完全无火力。
// 预期：前者返回 1，后者返回 0；失败意味着防空单位显示为 0 或空单位被强行记 1。
// 覆盖：两次精确断言，覆盖有输入但取整归零、无输入不兜底两条分支。
func TestWeightedTotalKeepsArmedUnitVisible(t *testing.T) {
	if got := weightedTotal(0, 1); got != 1 {
		t.Fatalf("weightedTotal(0, 1) = %d, want minimum visible power 1", got)
	}
	if got := weightedTotal(0, 0); got != 0 {
		t.Fatalf("weightedTotal(0, 0) = %d, want 0", got)
	}
}

// TestCaliberEffectivenessDiscountsSmallShells 验证对舰口径效能以 127mm 为满值并单调衰减。
// 场景：分别计算 0、127、460、25、1mm，再比较 25/76/127mm 的效能大小。
// 预期：0/127/460 为 1；25mm=pow(25/127,0.7)≈0.3207；1mm 触底 0.10；25<76<127。
// 覆盖：五处容差/相等断言加一组单调比较，覆盖未知口径、满值、上限夹取与下限。
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

// TestGunEffectivenessAppliesCaliberToAntiShipOnly 验证口径惩罚只作用于对舰，不影响对空。
// 场景：同参数炮（1 发/5s、射程 10、散布 50）分别装 25mm 与 200mm 弹，计算对舰、对空效能。
// 预期：对舰 mg=main*caliberEffectiveness(25)≈0.4*0.3207 且 mg<main；对空两者完全相等。
// 覆盖：两次 1e-9 断言加 mg<main 比较，锁死口径仅在 antiAir=false 分支生效。
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
	mg, main := gunEffectiveness(
		mount("mg"),
		bullets,
		false,
		false,
	), gunEffectiveness(
		mount("main"),
		bullets,
		false,
		false,
	)
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
