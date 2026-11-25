package unit

// RamConfig 描述一艘舰船的冲角与撞击能力。
type RamConfig struct {
	// Damage 满速、正角度撞上同吨位目标时的基础伤害（HP）。
	// <= 0 表示该舰没有撞击攻击能力。
	Damage float64 `json:"damage"`
	// Cooldown 同一对舰船两次撞击结算的最小间隔（秒）。默认 3。
	Cooldown float64 `json:"cooldown"`
	// SelfDamageRatio 撞击方按基础伤害承受的反作用比例，0~1。默认 0.1。
	SelfDamageRatio float64 `json:"selfDamageRatio"`
	// MinSpeed 有效撞击的最低接近速度（节）。默认 3；低于该值只接触不结算。
	MinSpeed float64 `json:"minSpeed"`
	// BowArc 舰艏撞击判定的半角（度）。默认 60；舰艏指向目标偏离超过该角度不算撞击。
	BowArc float64 `json:"bowArc"`
	// CanHitPlanes 是否允许撞击飞机。默认 false；仅特殊飞行单位（水滴）开启。
	CanHitPlanes bool `json:"canHitPlanes"`
	// AlwaysOn 无视游戏设置里的撞击总开关，始终启用。特殊单位（水滴 / 巨大剑鱼）使用。
	AlwaysOn bool `json:"alwaysOn"`
	// Derived 该配置是按舰种 / 吨位 / 装甲推导出来的默认撞击能力，而非 JSON 显式配置。
	// 默认能力不进静态战力与费用，避免打开体验开关就改变 AI 估值与造价。
	Derived bool `json:"-"`
}

// Normalize 补齐默认值并夹取到可用范围。
func (r *RamConfig) Normalize() {
	if r.Damage < 0 {
		r.Damage = 0
	}
	if r.Cooldown <= 0 {
		r.Cooldown = 3
	}
	r.Cooldown = max(r.Cooldown, 0.5)
	r.SelfDamageRatio = min(max(r.SelfDamageRatio, 0), 1)
	r.MinSpeed = max(r.MinSpeed, 0)
	if r.BowArc <= 0 {
		r.BowArc = 60
	}
	r.BowArc = min(max(r.BowArc, 15), 90)
}

// Enabled 该舰是否配置了撞击攻击能力。
func (r RamConfig) Enabled() bool {
	return r.Damage > 0
}

// RamReady 同一对舰船的撞击冷却是否已经结束。
func (s *BattleShip) RamReady(targetUID string) bool {
	if s.ramCooldownUntil == nil {
		return true
	}
	return SimTick() >= s.ramCooldownUntil[targetUID]
}

// RamContactBegan 记录本拍与目标存在接触，并返回这次接触是否刚开始。
// 上一拍仍处于接触时返回 false（同一拍重复调用也算持续接触），
// 因此持续重叠的两舰只会在接触开始的那一拍结算一次撞击。
func (s *BattleShip) RamContactBegan(targetUID string) bool {
	tick := SimTick()
	if s.ramContactTick == nil {
		s.ramContactTick = map[string]int64{}
	}
	prev, ok := s.ramContactTick[targetUID]
	s.ramContactTick[targetUID] = tick
	return !ok || prev < tick-1
}

// MarkRamHit 记录一次撞击，在配置的冷却时间内不再对同一目标结算。
func (s *BattleShip) MarkRamHit(targetUID string) {
	if s.ramCooldownUntil == nil {
		s.ramCooldownUntil = map[string]int64{}
	}
	until := SimTick() + ReloadTicks(s.Ram.Cooldown)
	if until <= SimTick() {
		until = SimTick() + 1
	}
	s.ramCooldownUntil[targetUID] = until
}

// 按舰种推导撞击能力时使用的系数。
const (
	// 撞击基础伤害占吨位（totalHP）的比例。
	ramBattleshipDamageRatio = 0.30
	ramCruiserDamageRatio    = 0.24
	ramDestroyerDamageRatio  = 0.30
	ramFrigateDamageRatio    = 0.26
	// 重甲加成：装甲值 0.9 时撞击伤害最多 +45%。
	ramArmorDamageBonus = 0.5
	// 重甲减自伤：装甲值 0.9 时自伤比例最多 -31.5%。
	ramArmorSelfReduction = 0.35
	// 默认撞击的最低接近速度（节）与舰艏判定半角（度）。
	ramDefaultMinSpeed = 4
	ramDefaultBowArc   = 60
)

// DefaultRamProfile 按舰种、吨位与装甲推导默认撞击配置。
// 只有战列舰 / 巡洋舰 / 驱逐舰 / 护卫舰参与；航母、货轮、维修船、医疗船、
// 潜艇、鱼雷艇与特殊单位返回 ok=false。重甲会提高撞击伤害、降低自伤比例。
func DefaultRamProfile(
	shipType ShipType, totalHP, horizontalReduction, verticalReduction float64,
) (RamConfig, bool) {
	var damageRatio, selfBase, cooldown float64
	switch shipType {
	case ShipTypeBattleShip:
		damageRatio, selfBase, cooldown = ramBattleshipDamageRatio, 0.15, 4
	case ShipTypeCruiser:
		damageRatio, selfBase, cooldown = ramCruiserDamageRatio, 0.22, 4
	case ShipTypeDestroyer:
		damageRatio, selfBase, cooldown = ramDestroyerDamageRatio, 0.30, 5
	case ShipTypeFrigate:
		damageRatio, selfBase, cooldown = ramFrigateDamageRatio, 0.28, 5
	default:
		return RamConfig{}, false
	}
	if totalHP <= 0 {
		return RamConfig{}, false
	}
	armor := min(max(max(horizontalReduction, verticalReduction), 0), 0.9)
	return RamConfig{
		Damage:          totalHP * damageRatio * (1 + ramArmorDamageBonus*armor),
		Cooldown:        cooldown,
		SelfDamageRatio: selfBase * (1 - ramArmorSelfReduction*armor),
		MinSpeed:        ramDefaultMinSpeed,
		BowArc:          ramDefaultBowArc,
		Derived:         true,
	}, true
}
