package object

import (
	"log"

	"github.com/mohae/deepcopy"
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
}

var _ AttackWeapon = (*Releaser)(nil)

// Fire 发射
func (lc *Releaser) Fire(shooter Attacker, enemy Hurtable) []*Bullet {
	return []*Bullet{}
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
