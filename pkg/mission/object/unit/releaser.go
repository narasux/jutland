package unit

import (
	"log"
	"math"

	"github.com/mohae/deepcopy"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
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

func (r *Releaser) shotParameters(
	shooter Attacker, enemy Hurtable,
) (UnitMovementState, objPos.MapPos, float64, bool) {
	// 已释放 / 对象不是战舰，不可发射
	if r.Released || enemy.ObjType() != object.TypeShip {
		return UnitMovementState{}, objPos.MapPos{}, 0, false
	}

	sState, eState := shooter.MovementState(), enemy.MovementState()

	// 应用全局速度倍率
	bulletSpeed := r.BulletSpeed * config.G.SpeedMultiplier

	// 考虑提前量（依赖敌舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		sState.CurPos.RX, sState.CurPos.RY, bulletSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)
	if !r.InShotRange(sState.CurRotation, sState.CurPos, targetPos) {
		return UnitMovementState{}, objPos.MapPos{}, 0, false
	}
	return sState, targetPos, bulletSpeed, true
}

// pathCrossesLand 检查鱼雷投放点到预计命中点之间是否有陆地阻挡。
// 目标后方的岸线不应阻止攻击靠岸停泊的战舰。
func (r *Releaser) pathCrossesLand(
	shooter Attacker, enemy Hurtable, terrain *mapcfg.MapData,
) bool {
	sState, targetPos, _, ok := r.shotParameters(shooter, enemy)
	if !ok {
		return false
	}

	// 每 1/4 个地图格取样一次，鱼雷射程很短，这里最多只会检查几十个点。
	steps := max(1, int(math.Ceil(sState.CurPos.Distance(targetPos)*4)))
	for step := 0; step <= steps; step++ {
		progress := float64(step) / float64(steps)
		pos := objPos.NewR(
			sState.CurPos.RX+(targetPos.RX-sState.CurPos.RX)*progress,
			sState.CurPos.RY+(targetPos.RY-sState.CurPos.RY)*progress,
		)
		if terrain.IsLand(pos.MX, pos.MY) {
			return true
		}
	}
	return false
}

// Fire 发射
func (r *Releaser) Fire(shooter Attacker, enemy Hurtable) (bullets []*objBullet.Bullet) {
	sState, targetPos, bulletSpeed, ok := r.shotParameters(shooter, enemy)
	if !ok {
		return
	}

	// 成功释放弹药
	r.Released = true
	bulletType := objBullet.GetType(r.BulletName)
	shotType := lo.Ternary(
		bulletType == objBullet.TypeBomb,
		objBullet.ShotTypeArcing, objBullet.ShotTypeDirect,
	)
	life := int(r.Range/bulletSpeed) + 5
	if bulletType == objBullet.TypeTorpedo {
		// 航空鱼雷只行驶到预计命中点附近，额外一帧确保到达后仍会结算碰撞。
		life = int(math.Ceil(sState.CurPos.Distance(targetPos)/bulletSpeed)) + 1
	}

	return []*objBullet.Bullet{objBullet.New(
		r.BulletName, sState.CurPos, targetPos,
		shooter.ID(), shooter.ObjType(), shooter.Player(),
		shotType, enemy.ObjType(), bulletSpeed, life,
	)}
}

// ReleaserMap 保存按配置名称索引的炸弹和航空鱼雷释放器模板。
var ReleaserMap = map[string]*Releaser{}

// NewReleaser 从模板创建独立释放器实例，并设置安装位置和左右投放射界。
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
