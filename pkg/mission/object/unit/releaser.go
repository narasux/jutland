package unit

import (
	"fmt"
	"log"
	"math"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// aerialTorpedoReleaseRangeRatio 预留 20% 最大射程作为敌舰规避缓冲。
const aerialTorpedoReleaseRangeRatio = 0.8

// Releaser （飞机）释放器
type Releaser struct {
	// 名称
	Name string `json:"name"`
	// 弹药名称
	BulletName string `json:"bulletName"`
	// BulletType 是弹药类型缓存，初始化时写入，避免热路径反复查询
	BulletType objBullet.Type `json:"-"`
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

// inRange 是否在射程内；航空鱼雷只投放至 80% 最大射程，预留敌舰规避缓冲。
func (r *Releaser) inRange(curPos, targetPos objPos.MapPos) bool {
	maxRange := r.Range
	if r.bulletType() == objBullet.TypeTorpedo {
		maxRange *= aerialTorpedoReleaseRangeRatio
	}
	return curPos.Distance(targetPos) <= maxRange
}

// inArc 是否在左右投放射界内
func (r *Releaser) inArc(shipCurRotation float64, curPos, targetPos objPos.MapPos) bool {
	rotation := math.Mod(curPos.Angle(targetPos)-shipCurRotation+360, 360)
	return r.LeftFiringArc.Contains(rotation) || r.RightFiringArc.Contains(rotation)
}

// InShotRange 是否在射程 & 射界内
func (r *Releaser) InShotRange(shipCurRotation float64, curPos, targetPos objPos.MapPos) bool {
	return r.inRange(curPos, targetPos) && r.inArc(shipCurRotation, curPos, targetPos)
}

// targetAcceptable 目标是否满足投放门槛：战舰始终可攻击；地面飞机（停放/滑行）
// 只能被炸弹攻击（鱼雷无法在陆地使用）。
func (r *Releaser) targetAcceptable(enemy Hurtable) bool {
	switch enemy.ObjType() {
	case object.TypeShip:
		return true
	case object.TypePlane:
		plane, ok := enemy.(*Plane)
		return ok && plane.IsOnGround() && r.bulletType() != objBullet.TypeTorpedo
	default:
		return false
	}
}

// detailLines 生成单枚释放器的调试文本：第一行是身份与挂载，第二行是当前能否
// 投放的逐项判定。enemy 为空时只输出第一行。
//
// 判定口径与 Fire 一致：提前量落点、目标门槛、射程、射界；terrain 非空时再追加
// 航空鱼雷的陆地阻挡判定。飞机级的投放间隔闸门不在本函数范围内：Plane.Fire 每拍
// 只放行一次投放，闸门未开时即使 canDrop 为真也不会投，由调用方单独展示。
func (r *Releaser) detailLines(
	shooter Attacker, enemy Hurtable, terrain *mapcfg.MapData,
) []string {
	state := "ready"
	if r.Released {
		state = "released"
	}
	identity := fmt.Sprintf(
		"%s (%s, %s): %s, range %.2f, speed %.2f",
		r.Name, r.BulletName, r.bulletType(), state, r.Range, r.BulletSpeed,
	)
	if enemy == nil {
		return []string{identity}
	}
	if !r.targetAcceptable(enemy) {
		return []string{identity, "targetOK false (not droppable target)"}
	}

	sState, eState := shooter.MovementState(), enemy.MovementState()
	// 与 Fire 相同的口径：弹速要乘全局速度倍率，落点取提前量位置。
	bulletSpeed := r.BulletSpeed
	if config.G != nil {
		bulletSpeed *= config.G.SpeedMultiplier
	}
	targetPos := eState.CurPos.Copy()
	if bulletSpeed > 0 {
		_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
			sState.CurPos.RX, sState.CurPos.RY, bulletSpeed,
			eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
		)
		targetPos = objPos.NewR(targetRx, targetRY)
	}
	inRange := r.inRange(sState.CurPos, targetPos)
	inArc := r.inArc(sState.CurRotation, sState.CurPos, targetPos)
	// dist 是到提前量落点的距离，不是到目标的距离：弹速慢时落点会远得多，
	// inRange 也按落点判定，所以飞机可能贴近目标却始终投不下去。
	verdict := fmt.Sprintf(
		"dist %.2f (lead point), inRange %t, inArc %t, canDrop %t",
		sState.CurPos.Distance(targetPos), inRange, inArc, !r.Released && inRange && inArc,
	)
	if r.bulletType() == objBullet.TypeTorpedo && terrain != nil {
		verdict += fmt.Sprintf(", landBlocked %t", r.pathCrossesLand(shooter, enemy, terrain))
	}
	return []string{identity, verdict}
}

func (r *Releaser) shotParameters(
	shooter Attacker, enemy Hurtable,
) (UnitMovementState, objPos.MapPos, float64, bool) {
	// 已释放，不可发射
	if r.Released {
		return UnitMovementState{}, objPos.MapPos{}, 0, false
	}
	// 目标门槛不通过，不可发射
	if !r.targetAcceptable(enemy) {
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

// bulletType 查询释放器装载的弹药类型（炸弹 / 航空鱼雷）。
func (r *Releaser) bulletType() objBullet.Type {
	if r.BulletType != "" {
		return r.BulletType
	}
	return objBullet.GetType(r.BulletName)
}

// Fire 发射
func (r *Releaser) Fire(shooter Attacker, enemy Hurtable) (bullets []*objBullet.Bullet) {
	sState, targetPos, bulletSpeed, ok := r.shotParameters(shooter, enemy)
	if !ok {
		return
	}

	// 成功释放弹药
	r.Released = true
	bulletType := r.bulletType()
	// 炸弹垂直落下，航空鱼雷平射
	plunge := 0.0
	if bulletType == objBullet.TypeBomb {
		plunge = 1
	}
	life := int(r.Range/bulletSpeed) + 5
	if bulletType == objBullet.TypeTorpedo {
		// 航空鱼雷只行驶到预计命中点附近，额外一帧确保到达后仍会结算碰撞。
		life = int(math.Ceil(sState.CurPos.Distance(targetPos)/bulletSpeed)) + 1
	}

	return []*objBullet.Bullet{objBullet.New(
		r.BulletName, sState.CurPos, targetPos,
		shooter.ID(), shooter.ObjType(), shooter.Player(),
		enemy.ObjType(), bulletSpeed, life, plunge,
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
	r.BulletType = objBullet.GetType(r.BulletName)
	return &r
}
