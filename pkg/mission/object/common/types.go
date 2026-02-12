package common

type ObjectType int

const (
	// ObjectTypeNone 无
	ObjectTypeNone ObjectType = iota
	// ObjectTypeShip 战舰
	ObjectTypeShip
	// ObjectTypePlane 战机
	ObjectTypePlane
	// ObjectTypeWater 水面
	ObjectTypeWater
	// ObjectTypeLand 陆地
	ObjectTypeLand
)

// ShipType 战舰类型
type ShipType string

const (
	// ShipTypeDefault 默认
	ShipTypeDefault ShipType = "default"
	// ShipTypeAircraftCarrier 航空母舰
	ShipTypeAircraftCarrier ShipType = "aircraft_carrier"
	// ShipTypeBattleShip 战列舰
	ShipTypeBattleShip ShipType = "battleship"
	// ShipTypeCruiser 巡洋舰
	ShipTypeCruiser ShipType = "cruiser"
	// ShipTypeDestroyer 驱逐舰
	ShipTypeDestroyer ShipType = "destroyer"
	// ShipTypeTorpedoBoat 鱼雷艇
	ShipTypeTorpedoBoat ShipType = "torpedo_boat"
	// ShipTypeCargo 货轮
	ShipTypeCargo ShipType = "cargo"
)
