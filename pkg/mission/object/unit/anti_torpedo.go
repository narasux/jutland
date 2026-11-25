package unit

import (
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

// 反鱼雷火控按舰种推导，不写进 ships.json5：这是玩法平衡，不是史实换算。
//
// 小舰（驱逐 / 护卫 / 雷击艇）本来就承担近距反雷与护航任务，火控组织快、
// 炮手盯海面；主力舰的防空炮组是为空袭准备的，组织一次对海拦截更慢，
// 单发引爆质量也更差。这样驱逐舰不会被一条鱼雷直接判死，主力舰也不会
// 因为炮座多就变成鱼雷免疫。
const (
	// 驱逐舰 / 护卫舰 / 鱼雷艇：每 0.5 秒可以重新组织一次拦截，
	// 在 3.5 格射程约 0.8 秒的窗口里通常能打出两轮，是反鱼雷主力。
	antiTorpedoFastInterval = 0.5
	antiTorpedoFastChance   = 0.08
	// 巡洋舰：整舰每秒一次，单发质量中等。
	antiTorpedoCruiserInterval = 1.0
	antiTorpedoCruiserChance   = 0.05
	// 战列舰 / 航母：火控更慢、单发质量更低，只够最后一道近防。
	antiTorpedoCapitalInterval = 1.5
	antiTorpedoCapitalChance   = 0.03
	// 其他舰种（货轮 / 维修船 / 医疗船 / 潜艇 / 未标注）用基准档。
	antiTorpedoDefaultInterval = 1.0
	antiTorpedoDefaultChance   = 0.05
)

// AntiTorpedoProfile 一艘舰的反鱼雷火控质量。
type AntiTorpedoProfile struct {
	// FireInterval 整舰组织一次拦截的间隔（秒）
	FireInterval float64
	// HitChance 单发炮弹穿过鱼雷命中框后的基准引爆概率。
	// 实际概率还要乘鱼雷自身的诱爆系数（见 Bullet.TorpedoDetonationFactor）。
	HitChance float64
}

// DefaultAntiTorpedoProfile 按舰种推导反鱼雷火控参数；未知舰种回落到基准档。
func DefaultAntiTorpedoProfile(shipType ShipType) AntiTorpedoProfile {
	switch shipType {
	case ShipTypeDestroyer, ShipTypeFrigate, ShipTypeTorpedoBoat:
		return AntiTorpedoProfile{antiTorpedoFastInterval, antiTorpedoFastChance}
	case ShipTypeCruiser:
		return AntiTorpedoProfile{antiTorpedoCruiserInterval, antiTorpedoCruiserChance}
	case ShipTypeBattleShip, ShipTypeAircraftCarrier:
		return AntiTorpedoProfile{antiTorpedoCapitalInterval, antiTorpedoCapitalChance}
	default:
		return AntiTorpedoProfile{antiTorpedoDefaultInterval, antiTorpedoDefaultChance}
	}
}

// CanBearAntiTorpedo 本舰是否至少有一座可用的反鱼雷炮能把炮口转向该位置。
// 只查射界，不查射程与提前量：用于择敌时跳过「在射程内但所有炮座都转不过去」
// 的鱼雷，避免火控盯着一枚打不到的雷反复空转、把真正冲本舰来的雷晾在一边。
func (s *BattleShip) CanBearAntiTorpedo(pos objPos.MapPos) bool {
	bearing := math.Mod(s.CurPos.Angle(pos)-s.CurRotation+360, 360)
	return gunsBearingAntiTorpedo(s.Weapon.AntiAircraftGuns, bearing) ||
		gunsBearingAntiTorpedo(s.Weapon.SecondaryGuns, bearing) ||
		gunsBearingAntiTorpedo(s.Weapon.MainGuns, bearing)
}

// gunsBearingAntiTorpedo 该组炮里是否有可反鱼雷的炮座覆盖该相对方位。
func gunsBearingAntiTorpedo(guns []*Gun, bearing float64) bool {
	for _, gun := range guns {
		if !gun.AntiTorpedo || gun.Disable {
			continue
		}
		if gun.LeftFiringArc.Contains(bearing) || gun.RightFiringArc.Contains(bearing) {
			return true
		}
	}
	return false
}
