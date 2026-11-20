package unit

import (
	"github.com/narasux/jutland/pkg/mission/faction"
)

// DamageState 单位在本局承受的伤害与最近一次攻击者记录，战舰和飞机共用。
type DamageState struct {
	// 最近一次命中本单位的攻击者，用于战况记录（击沉 / 击落时读取）
	LastAttackerName    string
	LastAttackerType    string
	LastAttackerPlayer  faction.Player
	LastAttackerIsPlane bool
	// LastHit 最近一次命中的武器与伤害
	LastHit HitInfo
	// DamageTaken 本局累计承受伤害
	DamageTaken float64
	// KilledByHit 是否由最近一次命中直接击沉 / 击落（用于区分战沉与普通损失）
	KilledByHit bool
}
