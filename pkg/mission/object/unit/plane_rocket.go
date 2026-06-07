package unit

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// PlaneRocketLauncher 飞机挂载火箭发射器，按固定间隔逐发打完整个挂载。
type PlaneRocketLauncher struct {
	// 火箭发射器名称
	Name string `json:"name"`
	// 火箭弹名称
	BulletName string `json:"bulletName"`
	// 总挂载火箭弹数量
	RocketCount int `json:"rocketCount"`
	// 单发发射间隔（单位: s）
	ShotInterval float64 `json:"shotInterval"`
	// 射程
	Range float64 `json:"range"`
	// 火箭弹散布
	BulletSpread int `json:"bulletSpread"`
	// 火箭弹速度
	BulletSpeed float64 `json:"bulletSpeed"`
	// 能否反舰
	AntiShip bool `json:"antiShip"`
	// 能否防空
	AntiAircraft bool `json:"antiAircraft"`
	// 近炸触发半径
	ProximityRadius float64 `json:"proximityRadius"`
	// 爆炸伤害半径
	BlastRadius float64 `json:"blastRadius"`
	// 相对位置
	PosPercent float64
	// 左射界
	LeftFiringArc FiringArc
	// 右射界
	RightFiringArc FiringArc

	// 已发射数量，飞机火箭弹不在空中重装
	ShotCount int
	// 最近发射时间（毫秒时间戳）
	LatestFireAt int64
}

var _ AttackWeapon = (*PlaneRocketLauncher)(nil)

// Exhausted 是否已经打完整个挂载。
func (r *PlaneRocketLauncher) Exhausted() bool {
	return r.ShotCount >= r.RocketCount
}

// Reloaded 是否满足下一枚火箭弹发射间隔；飞机火箭不重装，仅检查挂载余量。
func (r *PlaneRocketLauncher) Reloaded() bool {
	if r.Exhausted() {
		return false
	}
	return time.Now().UnixMilli() >= r.LatestFireAt+int64(r.ShotInterval*1e3)
}

// InShotRange 是否在射程 / 射界内。
func (r *PlaneRocketLauncher) InShotRange(planeCurRotation float64, curPos, targetPos objPos.MapPos) bool {
	if curPos.Distance(targetPos) > r.Range {
		return false
	}
	rotation := math.Mod(curPos.Angle(targetPos)-planeCurRotation+360, 360)
	return r.LeftFiringArc.Contains(rotation) || r.RightFiringArc.Contains(rotation)
}

// Fire 发射下一枚飞机火箭弹；目标类型由飞机当前目标规则决定。
func (r *PlaneRocketLauncher) Fire(shooter Attacker, enemy Hurtable) (bullets []*objBullet.Bullet) {
	if !r.Reloaded() {
		return nil
	}

	sState, eState := shooter.MovementState(), enemy.MovementState()
	curPos := sState.CurPos.Copy()
	rocketOffset := r.PosPercent * shooter.GeometricSize().Length / constants.MapBlockSize / 2
	curPos.AddRx(math.Sin(sState.CurRotation*math.Pi/180) * rocketOffset)
	curPos.SubRy(math.Cos(sState.CurRotation*math.Pi/180) * rocketOffset)

	bulletSpeed := r.BulletSpeed * config.G.SpeedMultiplier
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		sState.CurPos.RX, sState.CurPos.RY, bulletSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)
	if !r.InShotRange(sState.CurRotation, curPos, targetPos) {
		return nil
	}

	distance := curPos.Distance(targetPos)
	life := int(r.Range/bulletSpeed) + 5
	rangePercent := distance / r.Range
	radius := float64(r.BulletSpread) / constants.MapBlockSize * rangePercent

	pos := targetPos.Copy()
	pos.AddRx(float64(rand.Intn(3)-1) * rand.Float64() * radius)
	pos.AddRy(float64(rand.Intn(3)-1) * rand.Float64() * radius)
	bt := objBullet.New(
		r.BulletName, curPos, pos,
		shooter.ID(), shooter.ObjType(), shooter.Player(),
		objBullet.ShotTypeDirect, enemy.ObjType(), bulletSpeed, life,
	)
	bt.ProximityRadius = r.ProximityRadius
	bt.BlastRadius = r.BlastRadius
	bullets = append(bullets, bt)

	r.ShotCount++
	r.LatestFireAt = time.Now().UnixMilli()
	return bullets
}

// PlaneRocketLauncherMap 保存全局飞机火箭发射器模板。
var PlaneRocketLauncherMap = map[string]*PlaneRocketLauncher{}

// NewPlaneRocketLauncher 基于模板创建带飞机挂点位置与射界的火箭发射器实例。
func NewPlaneRocketLauncher(name string, posPercent float64, leftFireArc, rightFireArc FiringArc) *PlaneRocketLauncher {
	launcher, ok := PlaneRocketLauncherMap[name]
	if !ok {
		log.Fatalf("plane rocket launcher %s no found", name)
	}
	r := deepcopy.Copy(*launcher).(PlaneRocketLauncher)
	r.PosPercent = posPercent
	r.LeftFiringArc = leftFireArc
	r.RightFiringArc = rightFireArc
	return &r
}
