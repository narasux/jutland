package object

import (
	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mohae/deepcopy"

	"github.com/narasux/jutland/pkg/resources/images/ship"
)

type ShipName string

const (
	ShipDefault ShipName = "默认战舰"
)

var ships = map[ShipName]*BattleShip{
	ShipDefault: shipDefault,
}

var shipImg = map[ShipName]*ebiten.Image{
	ShipDefault: ship.ShipDefaultZeroImg,
}

// NewShip 新建战舰
func NewShip(name ShipName, pos MapPos, rotation float64) *BattleShip {
	s := deepcopy.Copy(*ships[name]).(BattleShip)
	s.Uid = uuid.New().String()
	s.CurPos = pos
	s.CurRotation = rotation
	return &s
}

// GetShipImg 获取战舰图片
func GetShipImg(name ShipName) *ebiten.Image {
	return shipImg[name]
}

var shipDefault = &BattleShip{
	Name:            ShipDefault,
	Type:            ShipTypeDestroyer,
	TotalHP:         1000,
	DamageReduction: 0.5,
	MaxSpeed:        0.1,
	RotateSpeed:     2,
	Weapon: Weapon{
		Guns: []*Gun{
			newGun(GunMK45, 0.2),
			newGun(GunMK45, 0.7),
		},
		// TODO 鱼雷先欠一下，后面再加
		Torpedoes: []*Torpedo{},
	},
	CurHP:       1000,
	CurPos:      MapPos{MX: 0, MY: 0},
	CurRotation: 0,
	CurSpeed:    0,
}