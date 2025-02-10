package object

import (
	"log"
	"math"
	"time"

	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

type TorpedoLauncher struct {
	// 发射器名称
	Name string `json:"name"`
	// 鱼雷类型
	BulletName string `json:"bulletName"`
	// 鱼雷数量
	BulletCount int `json:"bulletCount"`
	// 发射间隔（单位: s)
	ShotInterval float64 `json:"shotInterval"`
	// 装填时间（单位: s)
	ReloadTime float64 `json:"reloadTime"`
	// 射程
	Range float64 `json:"range"`
	// 鱼雷速度
	BulletSpeed float64 `json:"bulletSpeed"`
	// 相对位置
	// 0.35 -> 从中心往舰首 35% 舰体长度
	// -0.3 -> 从中心往舰尾 30% 舰体长度
	PosPercent float64
	// 左射界 (180, 360]
	LeftFiringArc FiringArc
	// 右射界 (0, 180]
	RightFiringArc FiringArc

	// 动态参数
	// 当前鱼雷是否可用（如战损 / 禁用）
	Disable bool
	// 开始装填时间（时间戳)
	ReloadStartAt int64
	// 最近发射时间（时间戳）
	LatestFireAt int64
	// 本次装填鱼雷已发射数量
	ShotCountBeforeReload int
}

// CanFire 是否可发射
func (lc *TorpedoLauncher) CanFire(shipCurRotation float64, curPos, targetPos MapPos) bool {
	// 未启用，不可发射
	if lc.Disable {
		return false
	}
	// 不在射程内，不可发射
	distance := geometry.CalcDistance(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	if distance > lc.Range {
		return false
	}
	// 不在射界范围内，不可发射
	rotation := geometry.CalcAngleBetweenPoints(curPos.RX, curPos.RY, targetPos.RX, targetPos.RY)
	rotation = math.Mod(rotation-shipCurRotation+360, 360)
	if !lc.LeftFiringArc.Contains(rotation) && !lc.RightFiringArc.Contains(rotation) {
		return false
	}
	// 检查是否已经重新装填完成
	return lc.Reloaded()
}

// Reloaded 是否在重新装填 / 发射间隔
func (lc *TorpedoLauncher) Reloaded() bool {
	// 注：鱼雷是需要考虑发射间隔的，比如每秒一发之类，全部打完才是重新装填
	timeNow := time.Now().UnixMilli()
	// 在重新装填，不可发射
	if timeNow < lc.ReloadStartAt+int64(lc.ReloadTime*1e3) {
		return false
	}
	// 小于发射间隔也是不行的
	if timeNow < lc.LatestFireAt+int64(lc.ShotInterval*1e3) {
		return false
	}
	return lc.ShotCountBeforeReload < lc.BulletCount
}

// Fire 发射
func (lc *TorpedoLauncher) Fire(self Attacker, enemy Hurtable) []*Bullet {
	sState := self.ManeuverState()
	eState := enemy.ManeuverState()

	curPos := sState.CurPos.Copy()
	// 炮塔距离战舰中心的距离
	gunOffset := lc.PosPercent * self.GeometricSize().Length / constants.MapBlockSize / 2
	curPos.AddRx(math.Sin(sState.CurRotation*math.Pi/180) * gunOffset)
	curPos.SubRy(math.Cos(sState.CurRotation*math.Pi/180) * gunOffset)

	// 考虑提前量（依赖敌舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		sState.CurPos.RX, sState.CurPos.RY, lc.BulletSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := NewMapPosR(targetRx, targetRY)

	if !lc.CanFire(sState.CurRotation, curPos, targetPos) {
		return []*Bullet{}
	}
	// 鱼雷不是齐射的，是一个一个来的
	lc.ShotCountBeforeReload++

	timeNow := time.Now().UnixMilli()
	lc.LatestFireAt = timeNow
	// 弹药打完了，重新装填
	if lc.ShotCountBeforeReload >= lc.BulletCount {
		lc.ShotCountBeforeReload = 0
		lc.ReloadStartAt = timeNow
	}

	// 鱼雷的生命值就是最大射程（+5 预留）
	life := int(lc.Range/lc.BulletSpeed) + 5
	// 注：鱼雷只有直射的情况，哪来的曲射？
	return []*Bullet{NewBullets(
		lc.BulletName, curPos, targetPos, BulletShotTypeDirect, lc.BulletSpeed, life, self.ID(), self.Player(),
	)}
}

var torpedoLauncherMap = map[string]*TorpedoLauncher{}

func newTorpedoLauncher(name string, posPercent float64, leftFireArc, rightFireArc FiringArc) *TorpedoLauncher {
	launcher, ok := torpedoLauncherMap[name]
	if !ok {
		log.Fatalf("torpedo launcher %s no found", name)
	}
	lc := deepcopy.Copy(*launcher).(TorpedoLauncher)
	lc.PosPercent = posPercent
	lc.LeftFiringArc = leftFireArc
	lc.RightFiringArc = rightFireArc
	return &lc
}
