package unit

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/object"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// RocketLauncher 防空火箭炮，按分组逐发抛出大量直射火箭弹。
type RocketLauncher struct {
	// 火箭炮名称
	Name string `json:"name"`
	// 火箭弹名称
	BulletName string `json:"bulletName"`
	// 单轮装填火箭弹数量
	RocketCount int `json:"rocketCount"`
	// 分组数量
	GroupCount int `json:"groupCount"`
	// 组内单发发射间隔（单位: s）
	ShotInterval float64 `json:"shotInterval"`
	// 分组发射间隔（单位: s）
	GroupInterval float64 `json:"groupInterval"`
	// 装填时间（单位: s）
	ReloadTime float64 `json:"reloadTime"`
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
	// 左射界 (180, 360]
	LeftFiringArc FiringArc
	// 右射界 (0, 180]
	RightFiringArc FiringArc

	// 当前火箭炮是否可用（如战损 / 禁用）
	Disable bool
	// 装填开始时间（毫秒时间戳）
	ReloadStartAt int64
	// 最近发射时间（毫秒时间戳）
	LatestFireAt int64
	// 本次装填已发射数量
	ShotCountBeforeReload int
}

var _ AttackWeapon = (*RocketLauncher)(nil)

// IsAvailableAntiType 是否能反制该类型
func (r *RocketLauncher) IsAvailableAntiType(objType object.Type) bool {
	if r.AntiAircraft && objType == object.TypePlane {
		return true
	}
	if r.AntiShip && objType == object.TypeShip {
		return true
	}
	return false
}

// Reloaded 是否已装填并满足下一枚火箭弹的发射间隔
func (r *RocketLauncher) Reloaded() bool {
	timeNow := time.Now().UnixMilli()
	if timeNow < r.ReloadStartAt+int64(r.ReloadTime*1e3) {
		return false
	}
	if r.ShotCountBeforeReload <= 0 {
		return true
	}

	groupSize := r.groupSize()
	interval := r.ShotInterval
	if r.ShotCountBeforeReload%groupSize == 0 {
		interval = r.GroupInterval
	}
	if timeNow < r.LatestFireAt+int64(interval*1e3) {
		return false
	}
	return r.ShotCountBeforeReload < r.RocketCount
}

// groupSize 返回每组火箭弹数量，最后一组由剩余数量自然截断
func (r *RocketLauncher) groupSize() int {
	return max(1, int(math.Ceil(float64(r.RocketCount)/float64(max(1, r.GroupCount)))))
}

// InShotRange 是否在射程 / 射界内
func (r *RocketLauncher) InShotRange(shipCurRotation float64, curPos, targetPos objPos.MapPos) bool {
	if curPos.Distance(targetPos) > r.Range {
		return false
	}
	rotation := math.Mod(curPos.Angle(targetPos)-shipCurRotation+360, 360)
	return r.LeftFiringArc.Contains(rotation) || r.RightFiringArc.Contains(rotation)
}

// Fire 发射下一枚火箭弹；每组按单发间隔逐发打完
func (r *RocketLauncher) Fire(shooter Attacker, enemy Hurtable) (bullets []*objBullet.Bullet) {
	if r.Disable || !r.Reloaded() || !r.IsAvailableAntiType(enemy.ObjType()) {
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

	r.ShotCountBeforeReload++
	timeNow := time.Now().UnixMilli()
	r.LatestFireAt = timeNow
	if r.ShotCountBeforeReload >= r.RocketCount {
		r.ShotCountBeforeReload = 0
		r.ReloadStartAt = timeNow
	}

	return bullets
}

// RocketLauncherMap 保存全局火箭炮模板。
var RocketLauncherMap = map[string]*RocketLauncher{}

// NewRocketLauncher 基于模板创建带舰体位置与射界的火箭炮实例
func NewRocketLauncher(name string, posPercent float64, leftFireArc, rightFireArc FiringArc) *RocketLauncher {
	launcher, ok := RocketLauncherMap[name]
	if !ok {
		log.Fatalf("rocket launcher %s no found", name)
	}
	r := deepcopy.Copy(*launcher).(RocketLauncher)
	r.PosPercent = posPercent
	r.LeftFiringArc = leftFireArc
	r.RightFiringArc = rightFireArc
	return &r
}
