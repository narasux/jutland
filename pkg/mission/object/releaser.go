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
}

var releaserMap = map[string]*Releaser{}

func newReleaser(name string) *Releaser {
	releaser, ok := releaserMap[name]
	if !ok {
		log.Fatalf("releaser %s no found", name)
	}
	r := deepcopy.Copy(*releaser).(Releaser)
	return &r
}
