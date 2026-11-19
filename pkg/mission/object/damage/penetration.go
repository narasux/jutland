package damage

import "math"

// PenetrationResult is the pure geometric penetration outcome for one hit.
type PenetrationResult struct {
	PenetrationMM    float64 `json:"penetrationMm"`
	ArmorMM          float64 `json:"armorMm"`
	ImpactAngleDeg   float64 `json:"impactAngleDeg"`
	EffectiveArmorMM float64 `json:"effectiveArmorMm"`
	Ratio            float64 `json:"ratio"`
	Penetrated       bool    `json:"penetrated"`
	Overmatch        bool    `json:"overmatch"`
}

// ResolvePenetration compares ammo penetration with angle-adjusted armor.
// Impact angle is measured from the plate normal (0 = perpendicular, 60 =
// oblique); values are clamped to a 0..80 degree envelope. This is a pure
// analysis model and does not alter the production HurtBy path.
func ResolvePenetration(penetrationMM, armorMM, impactAngleDeg float64) PenetrationResult {
	penetrationMM = max(0, penetrationMM)
	armorMM = max(0, armorMM)
	angle := math.Min(80, math.Max(0, impactAngleDeg))
	effective := armorMM
	if armorMM > 0 {
		effective = armorMM / math.Cos(angle*math.Pi/180)
	}
	ratio := 0.0
	if effective > 0 {
		ratio = penetrationMM / effective
	} else if penetrationMM > 0 {
		ratio = math.Inf(1)
	}
	return PenetrationResult{
		PenetrationMM: penetrationMM, ArmorMM: armorMM, ImpactAngleDeg: angle,
		EffectiveArmorMM: effective, Ratio: ratio,
		Penetrated: ratio >= 1, Overmatch: ratio >= 1.5,
	}
}

// PenetrationMultiplier maps a penetration/effective-armor ratio to damage.
// Experimental gameplay hook; the default simulation does not use it.
func PenetrationMultiplier(ratio float64) float64 {
	if ratio <= 0 {
		return 0.2
	}
	if ratio < 1 {
		return 0.2 + 0.8*ratio
	}
	return math.Min(1.35, 1+0.25*(ratio-1))
}
