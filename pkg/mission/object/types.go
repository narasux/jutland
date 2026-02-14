package object

// 战舰编组 ID
type GroupID int

const (
	GroupID0 = iota
	GroupID1
	GroupID2
	GroupID3
	GroupID4
	GroupID5
	GroupID6
	GroupID7
	GroupID8
	GroupID9
	GroupIDNone
)

type Type int

const (
	// TypeNone 无
	TypeNone Type = iota
	// TypeShip 战舰
	TypeShip
	// TypePlane 战机
	TypePlane
	// TypeWater 水面
	TypeWater
	// TypeLand 陆地
	TypeLand
)
