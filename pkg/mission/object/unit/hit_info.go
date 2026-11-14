package unit

import (
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
)

// HitInfo 最近一次命中使用的武器与实际伤害，供战报详情展示。
type HitInfo struct {
	// WeaponType shell / torpedo / bomb / rocket / laser
	WeaponType string
	// Diameter 口径整数（旧字段；精确值优先看 Caliber）
	Diameter int
	// Caliber 弹药配置中的精确口径文本，例如 7.62 / 12.7 / 203
	Caliber string
	// Weight 炸弹重量（kg），仅炸弹使用
	Weight int
	// Damage 本次实际伤害
	Damage float64
	// Critical 本次命中的暴击类型：普通 / 三倍 / 十倍
	Critical objBullet.CriticalType
}
