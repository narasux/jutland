package object

import (
	"time"

	"github.com/narasux/jutland/pkg/utils/geometry"
)

type Torpedo struct {
	// 固定参数
	// 百分比位置（如：0.33 -> 距离舰首 1/3)
	PosPercent float64
	// 鱼雷类型
	Bullet Bullet
	// 单次发射鱼雷数量
	BulletCount int
	// 装填时间（单位: s)
	ReloadTime int64
	// 射程
	Range float64
	// 鱼雷速度
	BulletSpeed float64

	// 动态参数
	// 当前鱼雷是否可用（如战损 / 禁用）
	Disable bool
	// 最后一次发射时间（时间戳)
	LastFireTime int64
}

// CanFire 是否可发射
func (t *Torpedo) CanFire(curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if t.Disable {
		return false
	}
	// 在重新装填，不可发射
	if t.LastFireTime+t.ReloadTime > time.Now().Unix() {
		return false
	}
	// 不在射程内，不可发射
	if geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY) > t.Range {
		return false
	}
	return true
}

// Fire 发射
func (t *Torpedo) Fire(ship, enemy *BattleShip) []*Bullet {
	if !t.CanFire(ship.CurPos, enemy.CurPos) {
		return []*Bullet{}
	}
	t.LastFireTime = time.Now().Unix()
	// FIXME 初始化弹药（复数，考虑当前位置，curPos 是战舰位置，还要计算相对位置，需要 uid）
	return []*Bullet{}
}
