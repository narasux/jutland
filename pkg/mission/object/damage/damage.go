// Package damage contains pure damage formulas shared by the game and the
// headless damage bench. It deliberately has no Ebiten/resource dependency.
package damage

// ResolveShip applies ship armor and critical multipliers.
// horizontalReduction/verticalReduction are 0..1; plunge is 0 (flat) .. 1
// (vertical). criticalRate is the configured probability; randVal must be
// sampled from the active battle RNG in [0,1). criticalMultiplier is 1, 3 or 10.
func ResolveShip(
	baseDamage, horizontalReduction, verticalReduction, plunge, criticalRate, randVal float64,
) (realDamage float64, criticalMultiplier int) {
	reduction := horizontalReduction*(1-plunge) + verticalReduction*plunge
	realDamage = baseDamage * (1 - reduction)
	criticalMultiplier = 1
	if randVal < criticalRate/10 {
		realDamage *= 10
		criticalMultiplier = 10
	} else if randVal < criticalRate {
		realDamage *= 3
		criticalMultiplier = 3
	}
	return realDamage, criticalMultiplier
}
