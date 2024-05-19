package object

import (
	"math"
	"math/rand"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type GunName string

const (
	GunMK45 GunName = "MK45"
)

type Gun struct {
	// 固定参数
	// 火炮名称
	Name GunName
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64
	// 炮弹类型
	BulletName BulletName
	// 单次抛射炮弹数量
	BulletCount int
	// 装填时间（单位: s)
	ReloadTime int64
	// 射程
	Range float64
	// 炮弹散布
	BulletSpread int
	// 炮弹速度
	BulletSpeed float64

	// 动态参数
	// 当前火炮是否可用（如战损 / 禁用）
	Disable bool
	// 最后一次射击时间（时间戳)
	LastFireTime int64
}

// CanFire 是否可发射
func (g *Gun) CanFire(curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if g.Disable {
		return false
	}
	// 在重新装填，不可发射
	if g.LastFireTime+g.ReloadTime > time.Now().Unix() {
		return false
	}
	// 不在射程内，不可发射
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	if distance > g.Range {
		return false
	}
	return true
}

// Fire 发射
func (g *Gun) Fire(ship, enemy *BattleShip) []*Bullet {
	shotBullets := []*Bullet{}

	curPos, targetPos := ship.CurPos.Copy(), enemy.CurPos.Copy()
	// 炮塔距离战舰中心的距离
	gunOffset := g.PosPercent * ship.Length / constants.MapBlockSize
	curPos.AddRx(math.Sin(ship.CurRotation*math.Pi/180) * gunOffset)
	curPos.SubRy(math.Cos(ship.CurRotation*math.Pi/180) * gunOffset)
	// FIXME 其实还要考虑提前量（依赖敌舰速度，角度）

	if !g.CanFire(curPos, targetPos) {
		return shotBullets
	}
	g.LastFireTime = time.Now().Unix()

	// 散布应该随着距离减小而减小
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	rangePercent := distance / g.Range
	// 炮弹散布的半径
	radius := float64(g.BulletSpread) / constants.MapBlockSize * rangePercent

	for i := 0; i < g.BulletCount; i++ {
		pos := targetPos.Copy()
		// rand.Intn(3) - 1 算方向，rand.Float64() 算距离
		pos.AddRx(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		pos.AddRy(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		shotBullets = append(shotBullets, NewBullets(
			g.BulletName, curPos, pos, g.BulletSpeed, ship.Uid, ship.BelongPlayer,
		))
	}

	return shotBullets
}

var guns = map[GunName]*Gun{
	GunMK45: gunMK45,
}

func newGun(name GunName, posPercent float64) *Gun {
	g := deepcopy.Copy(*guns[name]).(Gun)
	g.PosPercent = posPercent
	return &g
}

var gunMK45 = &Gun{
	Name:         GunMK45,
	PosPercent:   0,
	BulletName:   Gb127T1,
	BulletCount:  1,
	ReloadTime:   1,
	Range:        20,
	BulletSpread: 50,
	BulletSpeed:  0.5,
}
