package damage

import "math"

// RamResult 一次撞击的结算结果。
type RamResult struct {
	// TargetDamage 被撞方承受的伤害（已乘目标水平减伤）
	TargetDamage float64 `json:"targetDamage"`
	// SelfDamage 撞击方承受的反作用伤害
	SelfDamage float64 `json:"selfDamage"`
	// BaseDamage 目标减伤前的撞击伤害
	BaseDamage float64 `json:"baseDamage"`
	// ClosingSpeed 沿接触法线的接近速度（节）
	ClosingSpeed float64 `json:"closingSpeed"`
	// SpeedFactor 速度系数
	SpeedFactor float64 `json:"speedFactor"`
	// AngleFactor 角度系数
	AngleFactor float64 `json:"angleFactor"`
}

// ResolveRam 计算撞击伤害。所有速度使用节；bearingDeg 是撞击舰航向与
// 「撞击舰 -> 被撞舰」方位的夹角（度），targetReduction 是被撞方的水平减伤。
// 输入非法时返回零伤害，不产生 NaN。
func ResolveRam(
	ramDamage, closingSpeed, attackerMaxSpeed, bearingDeg, selfDamageRatio,
	attackerTonnage, targetTonnage, targetReduction float64,
) RamResult {
	result := RamResult{}
	if !math.IsNaN(closingSpeed) && closingSpeed > 0 {
		result.ClosingSpeed = closingSpeed
	}
	if math.IsNaN(ramDamage) || math.IsNaN(closingSpeed) || math.IsNaN(attackerMaxSpeed) ||
		math.IsNaN(bearingDeg) || math.IsNaN(selfDamageRatio) ||
		math.IsNaN(attackerTonnage) || math.IsNaN(targetTonnage) || math.IsNaN(targetReduction) {
		return result
	}
	if ramDamage <= 0 || closingSpeed <= 0 {
		return result
	}
	maxSpeed := max(attackerMaxSpeed, 1)
	speedFactor := min(max(closingSpeed/maxSpeed, 0), 1.5)
	result.SpeedFactor = speedFactor * speedFactor

	cosBearing := math.Cos(bearingDeg * math.Pi / 180)
	angleFactor := min(max(cosBearing, 0), 1)
	result.AngleFactor = math.Pow(angleFactor, 1.5)

	result.BaseDamage = ramDamage * result.SpeedFactor * result.AngleFactor
	reduction := min(max(targetReduction, 0), 1)
	result.TargetDamage = result.BaseDamage * (1 - reduction)

	massRatio := 1.0
	if attackerTonnage > 0 && targetTonnage > 0 {
		massRatio = min(max(targetTonnage/attackerTonnage, 0.1), 1)
	}
	result.SelfDamage = result.BaseDamage * min(max(selfDamageRatio, 0), 1) * massRatio
	return result
}
