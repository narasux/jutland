package object

import (
	"github.com/mohae/deepcopy"
)

type GunName string

const (
	GunMK45 GunName = "MK45"
)

var guns = map[GunName]*Gun{
	GunMK45: gunMK45,
}

func newGun(name GunName, posPercent float64) *Gun {
	g := deepcopy.Copy(*guns[name]).(Gun)
	g.PosPercent = posPercent
	return &g
}

var gunMK45 = &Gun{
	Name:         GunMK45,
	PosPercent:   0,
	BulletName:   Gb127T1,
	BulletCount:  1,
	ReloadTime:   1,
	Range:        20,
	BulletSpread: 50,
	BulletSpeed:  0.5,
}
