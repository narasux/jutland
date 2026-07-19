// Package combatpower 根据单位初始化后的运行时参数计算静态战力。
package combatpower

import (
	"math"
	"sort"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

const (
	// 战力评估默认以 60 秒为观察窗口，既能覆盖飞机有限弹药，也能稳定折算持续火力。
	evaluationWindow = 60.0
	// 最小减伤下限防止 100% 减伤单位在公式里出现除零。
	minimumDamageRate = 0.001
	// 航母航空战力不按满值直接相加，保留 0.7 的折扣是为了避免航空编队把舰体价值完全淹没。
	aviationFactor = 0.7
	// 飞机图鉴统一按 10 架标准编队评估，减少单机小数在过早取整时的信息损失。
	planeFormationSize = 10
	// 舰炮配置以现实公里数填写，初始化时除以 2 转成运行时距离。
	shipProjectionKilometersPerMapUnit = 2.0
	// 飞机航程初始化时除以 14.4 转成运行时距离。
	planeProjectionKilometersPerMapUnit = 14.4
)

type powerAccumulator struct {
	// powerAccumulator 负责把多门炮、多组挂载和多种武器来源汇总成统一的 DPS / 爆发池。
	// 这样可以在最后一步统一换算成战力，避免不同武器在中途各自 round 造成误差放大。
	antiShipDPS        float64
	antiAirDPS         float64
	burstShip          float64
	burstAir           float64
	maxProjectionRange float64
	antiShip           map[string]float64
	antiAir            map[string]float64
	burstToShip        map[string]float64
	burstToAir         map[string]float64
}

func newPowerAccumulator() *powerAccumulator {
	return &powerAccumulator{
		antiShip:    map[string]float64{},
		antiAir:     map[string]float64{},
		burstToShip: map[string]float64{},
		burstToAir:  map[string]float64{},
	}
}

// CalculatePlane 计算飞机标准编队的静态战力。
// 这里输入的是已经完成资源解析的运行时对象，所以不需要再回头查配置或做额外初始化。
func CalculatePlane(plane *objUnit.Plane, bullets map[string]*objBullet.Bullet) objUnit.CombatPowerInfo {
	if plane == nil {
		return objUnit.CombatPowerInfo{}
	}

	acc := newPowerAccumulator()
	for _, gun := range plane.Weapon.Guns {
		if gun.AntiShip {
			effectiveness := gunEffectiveness(gun, false, true)
			acc.addAntiShip(gun.Name, gunDPS(gun, bullets)*effectiveness, gunBurst(gun, bullets)*effectiveness)
		}
		if gun.AntiAircraft {
			effectiveness := gunEffectiveness(gun, true, true)
			acc.addAntiAir(gun.Name, gunDPS(gun, bullets)*effectiveness, gunBurst(gun, bullets)*effectiveness)
		}
	}
	for _, releaser := range plane.Weapon.Bombs {
		effectiveness := weaponEffectiveness(
			0.55, 0, releaser.Range, releaser.LeftFiringArc, releaser.RightFiringArc, false,
		)
		acc.addAntiShip(
			releaser.Name, releaserDPS(releaser, bullets)*effectiveness,
			releaserBurst(releaser, bullets)*effectiveness,
		)
	}
	for _, releaser := range plane.Weapon.Torpedoes {
		effectiveness := weaponEffectiveness(
			0.35, 0, releaser.Range, releaser.LeftFiringArc, releaser.RightFiringArc, false,
		)
		acc.addAntiShip(
			releaser.Name, releaserDPS(releaser, bullets)*effectiveness,
			releaserBurst(releaser, bullets)*effectiveness,
		)
	}
	for _, rocket := range plane.Weapon.Rockets {
		dps := planeRocketDPS(rocket, bullets)
		if rocket.AntiShip {
			effectiveness := weaponEffectiveness(
				0.45, rocket.BulletSpread, rocket.Range,
				rocket.LeftFiringArc, rocket.RightFiringArc, false,
			)
			acc.addAntiShip(rocket.Name, dps*effectiveness, planeRocketBurst(rocket, bullets)*effectiveness)
		}
		if rocket.AntiAircraft {
			effectiveness := weaponEffectiveness(
				0.50, rocket.BulletSpread, rocket.Range,
				rocket.LeftFiringArc, rocket.RightFiringArc, true,
			)
			acc.addAntiAir(rocket.Name, dps*effectiveness, planeRocketBurst(rocket, bullets)*effectiveness)
		}
	}

	// 战斗机继续只计对空；其他机种按实际武器标记同时保留对舰和对空能力。
	// 该规则只影响图鉴战力，不改动实战目标选择、AI 或伤害逻辑。
	if plane.Type == objUnit.PlaneTypeFighter {
		acc.clearAntiShip()
	}

	ehp := planeEHP(plane)
	mobility := planeMobility(plane)
	result := buildPowerInfo(ehp, mobility, plane.Range, planeFormationSize, acc)
	result.Details.MaxProjectionDistanceKM = plane.Range * planeProjectionKilometersPerMapUnit
	return result
}

// CalculateShip 计算舰船本体以及满编舰载机贡献后的静态战力。
// 舰体和舰载机分别先算，再在这里汇总，是为了保留“舰体价值”和“航空价值”的拆分结果。
func CalculateShip(
	ship *objUnit.BattleShip,
	planes map[string]*objUnit.Plane,
	bullets map[string]*objBullet.Bullet,
) objUnit.CombatPowerInfo {
	if ship == nil {
		return objUnit.CombatPowerInfo{}
	}

	acc := newPowerAccumulator()
	for _, guns := range [][]*objUnit.Gun{
		ship.Weapon.MainGuns, ship.Weapon.SecondaryGuns, ship.Weapon.AntiAircraftGuns,
	} {
		for _, gun := range guns {
			if gun.AntiShip {
				effectiveness := gunEffectiveness(gun, false, false)
				acc.addAntiShip(gun.Name, gunDPS(gun, bullets)*effectiveness, gunBurst(gun, bullets)*effectiveness)
			}
			if gun.AntiAircraft {
				effectiveness := gunEffectiveness(gun, true, false)
				acc.addAntiAir(gun.Name, gunDPS(gun, bullets)*effectiveness, gunBurst(gun, bullets)*effectiveness)
			}
			acc.maxProjectionRange = max(acc.maxProjectionRange, gun.Range)
		}
	}
	for _, torpedo := range ship.Weapon.Torpedoes {
		effectiveness := weaponEffectiveness(
			0.35, 0, torpedo.Range, torpedo.LeftFiringArc, torpedo.RightFiringArc, false,
		)
		acc.addAntiShip(
			torpedo.Name, torpedoDPS(torpedo, bullets)*effectiveness,
			torpedoBurst(torpedo, bullets)*effectiveness,
		)
		acc.maxProjectionRange = max(acc.maxProjectionRange, torpedo.Range)
	}
	for _, rocket := range ship.Weapon.Rockets {
		dps := shipRocketDPS(rocket, bullets)
		if rocket.AntiShip {
			effectiveness := weaponEffectiveness(
				0.45, rocket.BulletSpread, rocket.Range,
				rocket.LeftFiringArc, rocket.RightFiringArc, false,
			)
			acc.addAntiShip(rocket.Name, dps*effectiveness, shipRocketBurst(rocket, bullets)*effectiveness)
		}
		if rocket.AntiAircraft {
			effectiveness := weaponEffectiveness(
				0.50, rocket.BulletSpread, rocket.Range,
				rocket.LeftFiringArc, rocket.RightFiringArc, true,
			)
			acc.addAntiAir(rocket.Name, dps*effectiveness, shipRocketBurst(rocket, bullets)*effectiveness)
		}
		acc.maxProjectionRange = max(acc.maxProjectionRange, rocket.Range)
	}

	hull := buildPowerInfo(shipEHP(ship), shipMobility(ship), acc.maxProjectionRange, 1, acc)
	hull.Details.MaxProjectionDistanceKM = acc.maxProjectionRange * shipProjectionKilometersPerMapUnit
	result := hull
	result.Hull = hull.Total

	aviationAntiShip, aviationAntiAir := 0.0, 0.0
	for _, group := range ship.Aircraft.Groups {
		plane, ok := planes[group.Name]
		if !ok || plane == nil || group.MaxCount <= 0 {
			continue
		}
		count := float64(group.MaxCount)
		formationSize := float64(max(1, plane.CombatPower.FormationSize))
		aircraftFactor := count / formationSize * aviationFactor
		aviationAntiShip += float64(plane.CombatPower.AntiShip) * aircraftFactor
		aviationAntiAir += float64(plane.CombatPower.AntiAir) * aircraftFactor
		result.Details.AntiShipDPS += plane.CombatPower.Details.AntiShipDPS * aircraftFactor
		result.Details.AntiAirDPS += plane.CombatPower.Details.AntiAirDPS * aircraftFactor
		result.Details.BurstDamage += plane.CombatPower.Details.BurstDamage * aircraftFactor
		result.Details.MaxProjectionRange = max(result.Details.MaxProjectionRange, plane.Range)
		result.Details.MaxProjectionDistanceKM = max(
			result.Details.MaxProjectionDistanceKM,
			plane.Range*planeProjectionKilometersPerMapUnit,
		)
		addSortedContribution(
			&result.Details.AntiShipContributions, plane.Name, group.MaxCount,
			plane.CombatPower.Details.AntiShipDPS*aircraftFactor,
		)
		addSortedContribution(
			&result.Details.AntiAirContributions, plane.Name, group.MaxCount,
			plane.CombatPower.Details.AntiAirDPS*aircraftFactor,
		)
		addSortedContribution(
			&result.Details.BurstContributions, plane.Name, group.MaxCount,
			plane.CombatPower.Details.BurstDamage*aircraftFactor,
		)
	}

	aviationShip := int(math.Round(aviationAntiShip))
	aviationAir := int(math.Round(aviationAntiAir))
	result.AntiShip += aviationShip
	result.AntiAir += aviationAir
	result.Aviation = weightedTotal(aviationShip, aviationAir)
	result.Total = result.Hull + result.Aviation
	result.Projection = projectionScore(result.Details.MaxProjectionRange)
	result.Burst = burstScore(result.Details.BurstDamage)
	return result
}

func (a *powerAccumulator) addAntiShip(name string, dps, burst float64) {
	// 同名武器可能在不同挂载组里出现，先按名称聚合后再排序更适合图鉴展示。
	if dps > 0 {
		a.antiShipDPS += dps
		a.antiShip[name] += dps
	}
	if burst > 0 {
		a.burstShip += burst
		a.burstToShip[name] += burst
	}
}

func (a *powerAccumulator) addAntiAir(name string, dps, burst float64) {
	// 对空池和对舰池分开记账，最终会各自落到雷达图的不同维度。
	if dps > 0 {
		a.antiAirDPS += dps
		a.antiAir[name] += dps
	}
	if burst > 0 {
		a.burstAir += burst
		a.burstToAir[name] += burst
	}
}

func (a *powerAccumulator) clearAntiShip() {
	// 机种限制命中后，直接清空对舰统计，避免错误的跨目标输出污染结果。
	a.antiShipDPS = 0
	a.burstShip = 0
	a.antiShip = map[string]float64{}
	a.burstToShip = map[string]float64{}
}

func (a *powerAccumulator) burst() (float64, []objUnit.CombatPowerContribution) {
	// 爆发值取对舰/对空中更高的一侧，方便图鉴展示“这单位最强的一次齐射”。
	if a.burstAir > a.burstShip {
		return a.burstAir, sortedContributions(a.burstToAir)
	}
	return a.burstShip, sortedContributions(a.burstToShip)
}

func sortedContributions(values map[string]float64) []objUnit.CombatPowerContribution {
	// 贡献列表按数值从高到低输出，便于图鉴 tooltip 直接告诉玩家主要来源。
	result := make([]objUnit.CombatPowerContribution, 0, len(values))
	for name, value := range values {
		if value <= 0 {
			continue
		}
		result = append(result, objUnit.CombatPowerContribution{Name: name, Value: value})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Value == result[j].Value {
			return result[i].Name < result[j].Name
		}
		return result[i].Value > result[j].Value
	})
	return result
}

func addSortedContribution(
	target *[]objUnit.CombatPowerContribution, name string, count int64, value float64,
) {
	// 舰载机贡献会在多组挂载之间重复出现，所以这里需要做累加而不是简单 append。
	if value <= 0 {
		return
	}
	for idx := range *target {
		if (*target)[idx].Name == name {
			(*target)[idx].Count += count
			(*target)[idx].Value += value
			sort.Slice(*target, func(i, j int) bool { return (*target)[i].Value > (*target)[j].Value })
			return
		}
	}
	*target = append(*target, objUnit.CombatPowerContribution{Name: name, Count: count, Value: value})
	sort.Slice(*target, func(i, j int) bool { return (*target)[i].Value > (*target)[j].Value })
}

func buildPowerInfo(
	ehp, mobility, maxProjectionRange float64, formationSize int, acc *powerAccumulator,
) objUnit.CombatPowerInfo {
	// 这里统一把“存活能力 + 目标效率 + 机动修正”换算成最终对舰 / 对空战力。
	// 任何辅助字段都只保存在 Details 中，方便图鉴和 tooltip 解释来源。
	formationSize = max(1, formationSize)
	formationFactor := float64(formationSize)
	formationEHP := ehp * formationFactor
	antiShipDPS := acc.antiShipDPS * formationFactor
	antiAirDPS := acc.antiAirDPS * formationFactor
	antiShip := combatScore(formationEHP, antiShipDPS, mobility)
	antiAir := combatScore(formationEHP, antiAirDPS, mobility)
	burstDamage, burstContributions := acc.burst()
	burstDamage *= formationFactor
	return objUnit.CombatPowerInfo{
		FormationSize: formationSize,
		Total:         weightedTotal(antiShip, antiAir),
		AntiShip:      antiShip,
		AntiAir:       antiAir,
		Survival:      nonNegativeRound(math.Sqrt(max(0, formationEHP))),
		Mobility:      nonNegativeRound(100 * mobility),
		Projection:    projectionScore(maxProjectionRange),
		Burst:         burstScore(burstDamage),
		Details: objUnit.CombatPowerDetails{
			EffectiveHP:           formationEHP,
			AntiShipDPS:           antiShipDPS,
			AntiAirDPS:            antiAirDPS,
			MaxProjectionRange:    maxProjectionRange,
			BurstDamage:           burstDamage,
			AntiShipContributions: scaledContributions(sortedContributions(acc.antiShip), formationFactor),
			AntiAirContributions:  scaledContributions(sortedContributions(acc.antiAir), formationFactor),
			BurstContributions:    scaledContributions(burstContributions, formationFactor),
		},
	}
}

func scaledContributions(
	contributions []objUnit.CombatPowerContribution,
	factor float64,
) []objUnit.CombatPowerContribution {
	if factor == 1 {
		return contributions
	}
	for idx := range contributions {
		contributions[idx].Value *= factor
	}
	return contributions
}

func weightedTotal(antiShip, antiAir int) int {
	// 综合战力不是简单平均，而是偏向对舰能力，因为大多数单位的主用途仍然是打舰船。
	total := nonNegativeRound(0.7*float64(antiShip) + 0.3*float64(antiAir))
	if total == 0 && (antiShip > 0 || antiAir > 0) {
		return 1
	}
	return total
}

func combatScore(ehp, dps, mobility float64) int {
	// 采用 sqrt(EHP * DPS) 的形式，让“更硬”与“更能打”共同抬高分数，但不会被单项极端拉爆。
	if ehp <= 0 || dps <= 0 || mobility <= 0 {
		return 0
	}
	return nonNegativeRound(math.Sqrt(ehp*dps) * mobility / 10)
}

func projectionScore(maxProjectionRange float64) int {
	// 投送距离不直接参与主战斗分数，而是作为图鉴中的独立维度展示。
	return nonNegativeRound(maxProjectionRange * 10)
}

func burstScore(damage float64) int {
	// 爆发值用于表达一次齐射/齐投的体感威胁，不和持续 DPS 混在一起。
	return nonNegativeRound(math.Sqrt(max(0, damage)))
}

func expectedDamage(bulletName string, bullets map[string]*objBullet.Bullet) float64 {
	// 暴击期望按 Damage × (1 + 2.7 × CritRate) 计算。
	// 这样做比只看基础伤害更接近实际战斗表现，但仍然保持可解释性。
	bullet, ok := bullets[bulletName]
	if !ok || bullet == nil || bullet.Damage <= 0 {
		return 0
	}
	return bullet.Damage * (1 + 2.7*bullet.CriticalRate)
}

func gunDPS(gun *objUnit.Gun, bullets map[string]*objBullet.Bullet) float64 {
	// 火炮直接按“每轮伤害 / 装填周期”折算持续 DPS。
	if gun == nil || gun.BulletCount <= 0 || gun.ReloadTime <= 0 {
		return 0
	}
	return float64(gun.BulletCount) * expectedDamage(gun.BulletName, bullets) / gun.ReloadTime
}

func gunBurst(gun *objUnit.Gun, bullets map[string]*objBullet.Bullet) float64 {
	// 火炮爆发就是单轮齐射总伤害。
	if gun == nil || gun.BulletCount <= 0 {
		return 0
	}
	return float64(gun.BulletCount) * expectedDamage(gun.BulletName, bullets)
}

func torpedoDPS(launcher *objUnit.TorpedoLauncher, bullets map[string]*objBullet.Bullet) float64 {
	// 鱼雷按首发装填 + 连发间隔计算完整发射周期。
	if launcher == nil || launcher.BulletCount <= 0 {
		return 0
	}
	cycle := launcher.ReloadTime + float64(max(0, launcher.BulletCount-1))*launcher.ShotInterval
	if cycle <= 0 {
		return 0
	}
	return float64(launcher.BulletCount) * expectedDamage(launcher.BulletName, bullets) / cycle
}

func torpedoBurst(launcher *objUnit.TorpedoLauncher, bullets map[string]*objBullet.Bullet) float64 {
	if launcher == nil || launcher.BulletCount <= 0 {
		return 0
	}
	return float64(launcher.BulletCount) * expectedDamage(launcher.BulletName, bullets)
}

func shipRocketDPS(launcher *objUnit.RocketLauncher, bullets map[string]*objBullet.Bullet) float64 {
	// 舰载火箭存在分组发射，所以周期要同时考虑组内连发和组间间隔。
	if launcher == nil || launcher.RocketCount <= 0 {
		return 0
	}
	groupCount := min(max(1, launcher.GroupCount), launcher.RocketCount)
	groupSize := int(math.Ceil(float64(launcher.RocketCount) / float64(groupCount)))
	actualGroups := int(math.Ceil(float64(launcher.RocketCount) / float64(groupSize)))
	cycle := launcher.ReloadTime +
		float64(launcher.RocketCount-actualGroups)*launcher.ShotInterval +
		float64(max(0, actualGroups-1))*launcher.GroupInterval
	if cycle <= 0 {
		return 0
	}
	return float64(launcher.RocketCount) * expectedDamage(launcher.BulletName, bullets) / cycle
}

func shipRocketBurst(launcher *objUnit.RocketLauncher, bullets map[string]*objBullet.Bullet) float64 {
	if launcher == nil || launcher.RocketCount <= 0 {
		return 0
	}
	return float64(launcher.RocketCount) * expectedDamage(launcher.BulletName, bullets)
}

func releaserDPS(releaser *objUnit.Releaser, bullets map[string]*objBullet.Bullet) float64 {
	// 飞机炸弹和鱼雷在图鉴里按 60 秒窗口折算，避免低弹药挂载被高估。
	if releaser == nil {
		return 0
	}
	return expectedDamage(releaser.BulletName, bullets) / evaluationWindow
}

func releaserBurst(releaser *objUnit.Releaser, bullets map[string]*objBullet.Bullet) float64 {
	if releaser == nil {
		return 0
	}
	return expectedDamage(releaser.BulletName, bullets)
}

func planeRocketDPS(launcher *objUnit.PlaneRocketLauncher, bullets map[string]*objBullet.Bullet) float64 {
	// 飞机火箭同样按固定窗口折算，便于与炸弹、鱼雷放在同一战力框架下比较。
	if launcher == nil || launcher.RocketCount <= 0 {
		return 0
	}
	return float64(launcher.RocketCount) * expectedDamage(launcher.BulletName, bullets) / evaluationWindow
}

func planeRocketBurst(launcher *objUnit.PlaneRocketLauncher, bullets map[string]*objBullet.Bullet) float64 {
	if launcher == nil || launcher.RocketCount <= 0 {
		return 0
	}
	return float64(launcher.RocketCount) * expectedDamage(launcher.BulletName, bullets)
}

func gunEffectiveness(gun *objUnit.Gun, antiAir, planeShooter bool) float64 {
	// 命中系数把基础命中率、散布、射界和射程合在一起，得到一个可比较的有效输出倍率。
	if gun == nil {
		return 0
	}
	hitRate := 0.60
	if antiAir {
		hitRate = 0.11
		if planeShooter {
			hitRate = 0.55
		}
	}
	return weaponEffectiveness(
		hitRate, gun.BulletSpread, gun.Range,
		gun.LeftFiringArc, gun.RightFiringArc, antiAir,
	)
}

func weaponEffectiveness(
	hitRate float64,
	spread int,
	weaponRange float64,
	leftArc, rightArc objUnit.FiringArc,
	antiAir bool,
) float64 {
	// 对舰与对空使用不同的散布、射程基准，避免大口径舰炮和近程防空炮互相失真。
	spreadRef, rangeRef := 100.0, 10.0
	if antiAir {
		spreadRef, rangeRef = 20, 3
	}
	spreadFactor := clamp(spreadRef/(spreadRef+float64(max(0, spread))), 0.35, 1)
	arcFactor := math.Sqrt(firingArcCoverage(leftArc, rightArc))
	if weaponRange <= 0 {
		return 0
	}
	rangeFactor := clamp(math.Pow(weaponRange/rangeRef, 0.15), 0.8, 1.25)
	return hitRate * spreadFactor * arcFactor * rangeFactor
}

func firingArcCoverage(leftArc, rightArc objUnit.FiringArc) float64 {
	// 射界按并集角度计算，重叠区只算一次。
	intervals := [][2]float64{}
	for _, arc := range []objUnit.FiringArc{leftArc, rightArc} {
		start, end := clamp(arc.Start, 0, 360), clamp(arc.End, 0, 360)
		if end > start {
			intervals = append(intervals, [2]float64{start, end})
		}
	}
	if len(intervals) == 0 {
		return 0
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i][0] < intervals[j][0] })
	total, start, end := 0.0, intervals[0][0], intervals[0][1]
	for _, interval := range intervals[1:] {
		if interval[0] <= end {
			end = max(end, interval[1])
			continue
		}
		total += end - start
		start, end = interval[0], interval[1]
	}
	total += end - start
	return clamp(total/360, 0, 1)
}

func shipEHP(ship *objUnit.BattleShip) float64 {
	// 舰船 EHP 分成水平与垂直两套减伤，按 6:4 加权，近似反映不同武器类型的穿透差异。
	if ship == nil || ship.TotalHP <= 0 {
		return 0
	}
	horizontal := ship.TotalHP / max(minimumDamageRate, 1-ship.HorizontalDamageReduction)
	vertical := ship.TotalHP / max(minimumDamageRate, 1-ship.VerticalDamageReduction)
	return 0.6*horizontal + 0.4*vertical
}

func planeEHP(plane *objUnit.Plane) float64 {
	// 飞机通常会承受更高的集中打击，所以用三倍受伤系数把有效生存压回可比较的范围。
	if plane == nil || plane.TotalHP <= 0 {
		return 0
	}
	return plane.TotalHP / (3 * max(minimumDamageRate, 1-plane.DamageReduction))
}

func shipMobility(ship *objUnit.BattleShip) float64 {
	// 舰船机动由速度、转向和加速度共同决定，并限制在一个窄区间内，避免小差异放大。
	if ship == nil {
		return 0.75
	}
	value := math.Pow(nonNegativeRatio(ship.MaxSpeed, 0.05), 0.12) *
		math.Pow(nonNegativeRatio(ship.RotateSpeed, 2), 0.08) *
		math.Pow(nonNegativeRatio(ship.Acceleration, 0.0005), 0.04)
	return clamp(value, 0.75, 1.25)
}

func planeMobility(plane *objUnit.Plane) float64 {
	// 飞机机动比舰船更敏感，所以额外把航程也纳入到机动折算里。
	if plane == nil {
		return 0.75
	}
	value := math.Pow(nonNegativeRatio(plane.MaxSpeed, 500.0/5400), 0.12) *
		math.Pow(nonNegativeRatio(plane.RotateSpeed, 12), 0.08) *
		math.Pow(nonNegativeRatio(plane.Acceleration, 0.05), 0.03) *
		math.Pow(nonNegativeRatio(plane.Range, 1500.0/14.4), 0.05)
	return clamp(value, 0.75, 1.30)
}

func nonNegativeRatio(value, reference float64) float64 {
	// 所有比值都先做非负保护，避免非法配置把幂函数输入推成 NaN。
	if value <= 0 || reference <= 0 {
		return 0
	}
	return value / reference
}

func nonNegativeRound(value float64) int {
	// 任何非法或负数结果都压回 0，保证图鉴和测试里不会出现 NaN / Inf 的后续传播。
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return int(math.Round(value))
}

func clamp(value, minimum, maximum float64) float64 {
	// 通用夹取函数，所有经验系数都通过它收口。
	return min(maximum, max(minimum, value))
}
