package mission

type Mission string

const (
	// MissionDefault 默认关卡
	MissionDefault Mission = "default"
)

type MissionStatus string

const (
	// MissionRunning 任务进行中
	MissionRunning MissionStatus = "running"
	// MissionSuccess 任务成功
	MissionSuccess MissionStatus = "success"
	// MissionFailed 任务失败
	MissionFailed MissionStatus = "failed"
	// MissionError 任务错误
	MissionError MissionStatus = "error"
)

type ActionType string

const (
	// 无动作
	DoNothing ActionType = "doNothing"

	// HoverScreenMiddle 悬停在屏幕中间
	HoverScreenMiddle ActionType = "hoverScreenMiddle"

	// HoverScreenTop 悬停在屏幕顶部
	HoverScreenTop ActionType = "hoverScreenTop"

	// HoverScreenBottom 悬停在屏幕底部
	HoverScreenBottom ActionType = "hoverScreenBottom"

	// HoverScreenLeft 悬停在屏幕左侧
	HoverScreenLeft ActionType = "hoverScreenLeft"

	// HoverScreenRight 悬停在屏幕右侧
	HoverScreenRight ActionType = "hoverScreenRight"

	// HoverScreenTopLeft 悬停在屏幕左上角
	HoverScreenTopLeft ActionType = "hoverScreenTopLeft"

	// HoverScreenTopRight 悬停在屏幕右上角
	HoverScreenTopRight ActionType = "hoverScreenTopRight"

	// HoverScreenBottomLeft 悬停在屏幕左下角
	HoverScreenBottomLeft ActionType = "hoverScreenBottomLeft"

	// HoverScreenBottomRight 悬停在屏幕右下角
	HoverScreenBottomRight ActionType = "hoverScreenBottomRight"

	// SelectScreenArea 选取一个区域
	SelectScreenArea ActionType = "selectArea"
)

type ShipType string

const (
	// 航空母舰
	ShipTypeCarrier ShipType = "carrier"
	// 战列舰
	ShipTypeBattleship ShipType = "battleship"
	// 巡洋舰
	ShipTypeCruiser ShipType = "cruiser"
	// 驱逐舰
	ShipTypeDestroyer ShipType = "destroyer"
	// 护卫舰
	ShipTypeFrigate ShipType = "frigate"
	// 潜艇
	ShipTypeSubmarine ShipType = "submarine"
)

type WeaponType string

const (
	// 所有
	WeaponTypeAll WeaponType = "all"
	// 火炮
	WeaponTypeGun WeaponType = "gun"
	// 鱼雷
	WeaponTypeTorpedo WeaponType = "torpedo"
	// 导弹
	WeaponTypeMissile WeaponType = "missile"
)
