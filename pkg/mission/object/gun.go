package object

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// Gun 火炮
type Gun struct {
	// 火炮名称
	Name string `json:"name"`
	// 炮弹类型
	BulletName string `json:"bulletName"`
	// 单次抛射炮弹数量
	BulletCount int `json:"bulletCount"`
	// 装填时间（单位: s)
	ReloadTime float64 `json:"reloadTime"`
	// 射程
	Range float64 `json:"range"`
	// 炮弹散布
	BulletSpread int `json:"bulletSpread"`
	// 炮弹速度
	BulletSpeed float64 `json:"bulletSpeed"`
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64
	// 左射界 (180, 360]
	LeftFiringArc FiringArc
	// 右射界 (0, 180]
	RightFiringArc FiringArc

	// 当前火炮是否可用（如战损 / 禁用）
	Disable bool
	// 装填开始时间（毫秒时间戳)
	ReloadStartAt int64
}

// CanFire 是否可发射
func (g *Gun) CanFire(shipCurRotation float64, curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if g.Disable {
		return false
	}
	// 在重新装填，不可发射
	if !g.Reloaded() {
		return false
	}
	// 不在射程内，不可发射
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	if distance > g.Range {
		return false
	}
	// 不在射界范围内，不可发射
	rotation := geometry.CalcAngleBetweenPoints(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	rotation = math.Mod(rotation-shipCurRotation+360, 360)
	if !g.LeftFiringArc.Contains(rotation) && !g.RightFiringArc.Contains(rotation) {
		return false
	}
	return true
}

// Reloaded 是否已装填完成
func (g *Gun) Reloaded() bool {
	return g.ReloadStartAt+int64(g.ReloadTime*1e3) <= time.Now().UnixMilli()
}

// Fire 发射
func (g *Gun) Fire(ship, enemy *BattleShip) []*Bullet {
	curPos := ship.CurPos.Copy()
	// 炮塔距离战舰中心的距离
	gunOffset := g.PosPercent * ship.Length / constants.MapBlockSize / 2
	curPos.AddRx(math.Sin(ship.CurRotation*math.Pi/180) * gunOffset)
	curPos.SubRy(math.Cos(ship.CurRotation*math.Pi/180) * gunOffset)

	// 考虑提前量（依赖敌舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		ship.CurPos.RX, ship.CurPos.RY, g.BulletSpeed,
		enemy.CurPos.RX, enemy.CurPos.RY, enemy.CurSpeed, enemy.CurRotation,
	)
	targetPos := NewMapPosR(targetRx, targetRY)

	if !g.CanFire(ship.CurRotation, curPos, targetPos) {
		return []*Bullet{}
	}
	g.ReloadStartAt = time.Now().UnixMilli()

	// 散布应该随着距离减小而减小
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	rangePercent := distance / g.Range
	// 炮弹散布的半径
	radius := float64(g.BulletSpread) / constants.MapBlockSize * rangePercent

	shotType := BulletShotTypeArcing
	// 某些情况下使用直射
	if g.Name == "RailGun" {
		shotType = BulletShotTypeDirect
	} else {
		diameter := bulletMap[g.BulletName].Diameter
		if rangePercent < 0.65 || diameter <= 100 ||
			(diameter <= 200 && rangePercent < 0.8) ||
			(diameter <= 300 && rangePercent < 0.65) {
			shotType = BulletShotTypeDirect
		}
	}

	// 火炮炮弹生命值与目标距离相关，15 对于 0.4 速度的炮弹来说，相当于 6 格地图，在大多数火炮散布范围之内
	life := int(distance/g.BulletSpeed) + 15

	shotBullets := []*Bullet{}
	for i := 0; i < g.BulletCount; i++ {
		pos := targetPos.Copy()
		// rand.Intn(3) - 1 算方向，rand.Float64() 算距离
		pos.AddRx(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		pos.AddRy(float64(rand.Intn(3)-1) * rand.Float64() * radius)
		shotBullets = append(shotBullets, NewBullets(
			g.BulletName, curPos, pos, shotType, g.BulletSpeed, life, ship.Uid, ship.BelongPlayer,
		))
	}

	return shotBullets
}

var gunMap = map[string]*Gun{}

func newGun(name string, posPercent float64, leftFireArc, rightFireArc FiringArc) *Gun {
	gun, ok := gunMap[name]
	if !ok {
		log.Fatalf("gun %s no found", name)
	}
	g := deepcopy.Copy(*gun).(Gun)
	g.PosPercent = posPercent
	g.LeftFiringArc = leftFireArc
	g.RightFiringArc = rightFireArc
	return &g
}
