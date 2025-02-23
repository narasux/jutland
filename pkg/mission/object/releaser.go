package object

import (
	"log"
	"math"

	"github.com/mohae/deepcopy"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/utils/geometry"
)

// Releaser （飞机）释放器
type Releaser struct {
	// 名称
	Name string `json:"name"`
	// 弹药名称
	BulletName string `json:"bulletName"`
	// 类别
	Type ShipType `json:"type"`
	// 射程
	Range float64 `json:"range"`
	// 弹药速度
	BulletSpeed float64 `json:"bulletSpeed"`
	// 相对位置
	// 0.35 -> 从中心往头部 35% 舰体长度
	// -0.3 -> 从中心往尾部 30% 舰体长度
	PosPercent float64
	// 左射界 (180, 360]
	LeftFiringArc FiringArc
	// 右射界 (0, 180]
	RightFiringArc FiringArc
	// 是否已释放
	Released bool
}

var _ AttackWeapon = (*Releaser)(nil)

// InShotRange 是否在射程 & 射界内
func (r *Releaser) InShotRange(shipCurRotation float64, curPos, targetPos MapPos) bool {
	// 不在射程内，不可发射
	if curPos.Distance(targetPos) > r.Range {
		return false
	}
	// 不在射界范围内，不可发射
	rotation := math.Mod(curPos.Angle(targetPos)-shipCurRotation+360, 360)
	if !r.LeftFiringArc.Contains(rotation) && !r.RightFiringArc.Contains(rotation) {
		return false
	}
	return true
}

// Fire 发射
func (r *Releaser) Fire(shooter Attacker, enemy Hurtable) (bullets []*Bullet) {
	// 已释放 / 对象不是战舰，不可发射
	if r.Released || enemy.ObjType() != ObjectTypeShip {
		return
	}

	sState, eState := shooter.MovementState(), enemy.MovementState()

	// 考虑提前量（依赖敌舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		sState.CurPos.RX, sState.CurPos.RY, r.BulletSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := NewMapPosR(targetRx, targetRY)

	if !r.InShotRange(sState.CurRotation, sState.CurPos, targetPos) {
		return
	}

	// 成功释放弹药
	r.Released = true
	shotType := lo.Ternary(
		GetBulletType(r.BulletName) == BulletTypeBomb,
		BulletShotTypeArcing, BulletShotTypeDirect,
	)
	life := int(r.Range/r.BulletSpeed) + 5

	return []*Bullet{NewBullets(
		r.BulletName, sState.CurPos, targetPos, shotType,
		r.BulletSpeed, life, shooter.ID(), shooter.Player(),
	)}
}

var releaserMap = map[string]*Releaser{}

func newReleaser(name string, posPercent float64, leftFireArc, rightFireArc FiringArc) *Releaser {
	releaser, ok := releaserMap[name]
	if !ok {
		log.Fatalf("releaser %s no found", name)
	}
	r := deepcopy.Copy(*releaser).(Releaser)
	r.PosPercent = posPercent
	r.LeftFiringArc = leftFireArc
	r.RightFiringArc = rightFireArc
	return &r
}
