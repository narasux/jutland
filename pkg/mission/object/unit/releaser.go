package unit

import (
	"log"
	"math"

	"github.com/mohae/deepcopy"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
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
func (r *Releaser) InShotRange(shipCurRotation float64, curPos, targetPos objPos.MapPos) bool {
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
func (r *Releaser) Fire(shooter Attacker, enemy Hurtable) (bullets []*objBullet.Bullet) {
	// 已释放 / 对象不是战舰，不可发射
	if r.Released || enemy.ObjType() != object.TypeShip {
		return
	}

	sState, eState := shooter.MovementState(), enemy.MovementState()

	// 考虑提前量（依赖敌舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		sState.CurPos.RX, sState.CurPos.RY, r.BulletSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)

	if !r.InShotRange(sState.CurRotation, sState.CurPos, targetPos) {
		return
	}

	// 成功释放弹药
	r.Released = true
	shotType := lo.Ternary(
		objBullet.GetType(r.BulletName) == objBullet.BulletTypeBomb,
		objBullet.BulletShotTypeArcing, objBullet.BulletShotTypeDirect,
	)
	life := int(r.Range/r.BulletSpeed) + 5

	return []*objBullet.Bullet{objBullet.New(
		r.BulletName, sState.CurPos, targetPos,
		shooter.ID(), shooter.ObjType(), shooter.Player(),
		shotType, enemy.ObjType(), r.BulletSpeed, life,
	)}
}

var ReleaserMap = map[string]*Releaser{}

func NewReleaser(name string, posPercent float64, leftFireArc, rightFireArc FiringArc) *Releaser {
	releaser, ok := ReleaserMap[name]
	if !ok {
		log.Fatalf("releaser %s no found", name)
	}
	r := deepcopy.Copy(*releaser).(Releaser)
	r.PosPercent = posPercent
	r.LeftFiringArc = leftFireArc
	r.RightFiringArc = rightFireArc
	return &r
}
