// Package combatpower 根据单位初始化后的运行时参数计算静态战力。
package combatpower

import (
	"math"
	"sort"
	"strconv"

	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

const (
	evaluationWindow  = 60.0
	minimumDamageRate = 0.001
	aviationFactor    = 0.7
)

type powerAccumulator struct {
	antiShipDPS float64
	antiAirDPS  float64
	burstShip   float64
	burstAir    float64
	maxRange    float64
	antiShip    map[string]float64
	antiAir     map[string]float64
	burstToShip map[string]float64
	burstToAir  map[string]float64
}

func newPowerAccumulator() *powerAccumulator {
	return &powerAccumulator{
		antiShip:    map[string]float64{},
		antiAir:     map[string]float64{},
		burstToShip: map[string]float64{},
		burstToAir:  map[string]float64{},
	}
}

// CalculatePlane 计算单架飞机的静态战力。
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

	// 当前战斗逻辑中战斗机只攻击飞机，轰炸机与鱼雷机只攻击舰船。
	switch plane.Type {
	case objUnit.PlaneTypeFighter:
		acc.clearAntiShip()
	case objUnit.PlaneTypeDiveBomber, objUnit.PlaneTypeTorpedoBomber:
		acc.clearAntiAir()
	}

	ehp := planeEHP(plane)
	mobility := planeMobility(plane)
	return buildPowerInfo(ehp, mobility, plane.Range, acc)
}

// CalculateShip 计算舰船本体以及满编舰载机贡献后的静态战力。
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
			acc.maxRange = max(acc.maxRange, gun.Range)
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
		acc.maxRange = max(acc.maxRange, torpedo.Range)
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
		acc.maxRange = max(acc.maxRange, rocket.Range)
	}

	hull := buildPowerInfo(shipEHP(ship), shipMobility(ship), acc.maxRange, acc)
	result := hull
	result.Hull = hull.Total

	aviationAntiShip, aviationAntiAir := 0.0, 0.0
	for _, group := range ship.Aircraft.Groups {
		plane, ok := planes[group.Name]
		if !ok || plane == nil || group.MaxCount <= 0 {
			continue
		}
		count := float64(group.MaxCount)
		aviationAntiShip += float64(plane.CombatPower.AntiShip) * count * aviationFactor
		aviationAntiAir += float64(plane.CombatPower.AntiAir) * count * aviationFactor
		result.Details.AntiShipDPS += plane.CombatPower.Details.AntiShipDPS * count * aviationFactor
		result.Details.AntiAirDPS += plane.CombatPower.Details.AntiAirDPS * count * aviationFactor
		result.Details.BurstDamage += plane.CombatPower.Details.BurstDamage * count * aviationFactor
		result.Details.MaxRange = max(result.Details.MaxRange, plane.Range)
		label := plane.DisplayName + " ×" + formatCount(group.MaxCount)
		addSortedContribution(
			&result.Details.AntiShipContributions, label,
			plane.CombatPower.Details.AntiShipDPS*count*aviationFactor,
		)
		addSortedContribution(
			&result.Details.AntiAirContributions, label,
			plane.CombatPower.Details.AntiAirDPS*count*aviationFactor,
		)
		addSortedContribution(
			&result.Details.BurstContributions, label,
			plane.CombatPower.Details.BurstDamage*count*aviationFactor,
		)
	}

	aviationShip := int(math.Round(aviationAntiShip))
	aviationAir := int(math.Round(aviationAntiAir))
	result.AntiShip += aviationShip
	result.AntiAir += aviationAir
	result.Aviation = weightedTotal(aviationShip, aviationAir)
	result.Total = result.Hull + result.Aviation
	result.Range = rangeScore(result.Details.MaxRange)
	result.Burst = burstScore(result.Details.BurstDamage)
	return result
}

func (a *powerAccumulator) addAntiShip(name string, dps, burst float64) {
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
	a.antiShipDPS = 0
	a.burstShip = 0
	a.antiShip = map[string]float64{}
	a.burstToShip = map[string]float64{}
}

func (a *powerAccumulator) clearAntiAir() {
	a.antiAirDPS = 0
	a.burstAir = 0
	a.antiAir = map[string]float64{}
	a.burstToAir = map[string]float64{}
}

func (a *powerAccumulator) burst() (float64, []objUnit.CombatPowerContribution) {
	if a.burstAir > a.burstShip {
		return a.burstAir, sortedContributions(a.burstToAir)
	}
	return a.burstShip, sortedContributions(a.burstToShip)
}

func sortedContributions(values map[string]float64) []objUnit.CombatPowerContribution {
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

func addSortedContribution(target *[]objUnit.CombatPowerContribution, name string, value float64) {
	if value <= 0 {
		return
	}
	for idx := range *target {
		if (*target)[idx].Name == name {
			(*target)[idx].Value += value
			sort.Slice(*target, func(i, j int) bool { return (*target)[i].Value > (*target)[j].Value })
			return
		}
	}
	*target = append(*target, objUnit.CombatPowerContribution{Name: name, Value: value})
	sort.Slice(*target, func(i, j int) bool { return (*target)[i].Value > (*target)[j].Value })
}

func formatCount(count int64) string {
	return strconv.FormatInt(count, 10)
}

func buildPowerInfo(
	ehp, mobility, maxRange float64, acc *powerAccumulator,
) objUnit.CombatPowerInfo {
	antiShip := combatScore(ehp, acc.antiShipDPS, mobility)
	antiAir := combatScore(ehp, acc.antiAirDPS, mobility)
	burstDamage, burstContributions := acc.burst()
	return objUnit.CombatPowerInfo{
		Total:    weightedTotal(antiShip, antiAir),
		AntiShip: antiShip,
		AntiAir:  antiAir,
		Survival: nonNegativeRound(math.Sqrt(max(0, ehp))),
		Mobility: nonNegativeRound(100 * mobility),
		Range:    rangeScore(maxRange),
		Burst:    burstScore(burstDamage),
		Details: objUnit.CombatPowerDetails{
			EffectiveHP:           ehp,
			AntiShipDPS:           acc.antiShipDPS,
			AntiAirDPS:            acc.antiAirDPS,
			MaxRange:              maxRange,
			BurstDamage:           burstDamage,
			AntiShipContributions: sortedContributions(acc.antiShip),
			AntiAirContributions:  sortedContributions(acc.antiAir),
			BurstContributions:    burstContributions,
		},
	}
}

func weightedTotal(antiShip, antiAir int) int {
	total := nonNegativeRound(0.7*float64(antiShip) + 0.3*float64(antiAir))
	if total == 0 && (antiShip > 0 || antiAir > 0) {
		return 1
	}
	return total
}

func combatScore(ehp, dps, mobility float64) int {
	if ehp <= 0 || dps <= 0 || mobility <= 0 {
		return 0
	}
	return nonNegativeRound(math.Sqrt(ehp*dps) * mobility / 10)
}

func rangeScore(maxRange float64) int {
	return nonNegativeRound(maxRange * 10)
}

func burstScore(damage float64) int {
	return nonNegativeRound(math.Sqrt(max(0, damage)))
}

func expectedDamage(bulletName string, bullets map[string]*objBullet.Bullet) float64 {
	bullet, ok := bullets[bulletName]
	if !ok || bullet == nil || bullet.Damage <= 0 {
		return 0
	}
	return bullet.Damage * (1 + 2.7*bullet.CriticalRate)
}

func gunDPS(gun *objUnit.Gun, bullets map[string]*objBullet.Bullet) float64 {
	if gun == nil || gun.BulletCount <= 0 || gun.ReloadTime <= 0 {
		return 0
	}
	return float64(gun.BulletCount) * expectedDamage(gun.BulletName, bullets) / gun.ReloadTime
}

func gunBurst(gun *objUnit.Gun, bullets map[string]*objBullet.Bullet) float64 {
	if gun == nil || gun.BulletCount <= 0 {
		return 0
	}
	return float64(gun.BulletCount) * expectedDamage(gun.BulletName, bullets)
}

func torpedoDPS(launcher *objUnit.TorpedoLauncher, bullets map[string]*objBullet.Bullet) float64 {
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
	if ship == nil || ship.TotalHP <= 0 {
		return 0
	}
	horizontal := ship.TotalHP / max(minimumDamageRate, 1-ship.HorizontalDamageReduction)
	vertical := ship.TotalHP / max(minimumDamageRate, 1-ship.VerticalDamageReduction)
	return 0.6*horizontal + 0.4*vertical
}

func planeEHP(plane *objUnit.Plane) float64 {
	if plane == nil || plane.TotalHP <= 0 {
		return 0
	}
	return plane.TotalHP / (3 * max(minimumDamageRate, 1-plane.DamageReduction))
}

func shipMobility(ship *objUnit.BattleShip) float64 {
	if ship == nil {
		return 0.75
	}
	value := math.Pow(nonNegativeRatio(ship.MaxSpeed, 0.05), 0.12) *
		math.Pow(nonNegativeRatio(ship.RotateSpeed, 2), 0.08) *
		math.Pow(nonNegativeRatio(ship.Acceleration, 0.0005), 0.04)
	return clamp(value, 0.75, 1.25)
}

func planeMobility(plane *objUnit.Plane) float64 {
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
	if value <= 0 || reference <= 0 {
		return 0
	}
	return value / reference
}

func nonNegativeRound(value float64) int {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return int(math.Round(value))
}

func clamp(value, minimum, maximum float64) float64 {
	return min(maximum, max(minimum, value))
}
