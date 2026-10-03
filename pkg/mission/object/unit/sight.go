package unit

// 舰种和机种的默认视距，单位是地图格。
// 有主炮的船在初始化时再封顶到主炮射程。
const (
	SightRangeScout       = 36
	SightRangeFighter     = 22
	SightRangeBomber      = 26
	SightRangeAirfield    = 12
	SightRangeTorpedoBoat = 7
	SightRangeFrigate     = 11
	SightRangeDestroyer   = 13
	SightRangeCruiser     = 16
	SightRangeBattleship  = 18
	SightRangeCarrier     = 17
	SightRangeCargo       = 12
	SightRangeHospital    = 15
	SightRangeRepair      = 11
	// SightRangeSpecial 小黄鸭、水滴这类 type 为 default 的特殊船。
	// 它们不套舰种表，也不按主炮射程封顶——水滴的主炮射程极短，一封顶就什么都看不见了。
	SightRangeSpecial = 16
)

// ResolveShipSight 得到舰船视距。配置大于 0 时直接使用。
// 否则用舰种基数；有主炮时不超过主炮射程，避免比自己的炮看得更远。
// 特殊船（default）用固定基数，不封顶。
func ResolveShipSight(shipType ShipType, configured, mainGunRange float64) float64 {
	// 单艘写了大于 0 的视距时，不再用舰种基数，也不再封顶。
	if configured > 0 {
		return configured
	}
	if shipType == ShipTypeDefault {
		return SightRangeSpecial
	}
	// 没写视距就用舰种基数。有主炮时取得更小的那个，避免比自己的炮看得更远。
	base := defaultShipSight(shipType)
	if mainGunRange > 0 && mainGunRange < base {
		return mainGunRange
	}
	return base
}

func defaultShipSight(shipType ShipType) float64 {
	switch shipType {
	case ShipTypeTorpedoBoat:
		return SightRangeTorpedoBoat
	case ShipTypeFrigate:
		return SightRangeFrigate
	case ShipTypeDestroyer:
		return SightRangeDestroyer
	case ShipTypeCruiser:
		return SightRangeCruiser
	case ShipTypeBattleShip:
		return SightRangeBattleship
	case ShipTypeAircraftCarrier:
		return SightRangeCarrier
	case ShipTypeCargo:
		return SightRangeCargo
	case ShipTypeHospital:
		return SightRangeHospital
	case ShipTypeRepair:
		return SightRangeRepair
	default:
		return 0
	}
}

// ResolvePlaneSight 得到飞机视距。配置大于 0 时直接使用。
// 侦察机用统一视距，不因曾经归在 other 里而变成 0。
func ResolvePlaneSight(planeType PlaneType, configured float64) float64 {
	if configured > 0 {
		return configured
	}
	switch planeType {
	case PlaneTypeScout:
		return SightRangeScout
	case PlaneTypeFighter:
		return SightRangeFighter
	case PlaneTypeDiveBomber, PlaneTypeLevelBomber, PlaneTypeTorpedoBomber, PlaneTypeAttacker:
		return SightRangeBomber
	default:
		return 0
	}
}

// ProvidesSight 在空飞机才照雾。甲板滑跑和着舰回收跟着载舰，不再单独照。
func (p *Plane) ProvidesSight() bool {
	if p == nil || p.CurHP <= 0 {
		return false
	}
	switch p.FlightPhase {
	case PlaneFlightPhaseCruising, PlaneFlightPhaseLandingStaging, PlaneFlightPhaseLandingApproach, "":
		return true
	default:
		return false
	}
}
