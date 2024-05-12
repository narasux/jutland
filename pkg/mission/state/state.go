package state

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/layout"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// Camera 相机（当前视野）
type Camera struct {
	// 相机左上角位置
	Pos    object.MapPos
	Width  int
	Height int
}

// Contains 判断坐标是否在视野内
func (c *Camera) Contains(pos object.MapPos) bool {
	return !(pos.MX < c.Pos.MX || pos.MX > c.Pos.MX+c.Width || pos.MY < c.Pos.MY || pos.MY > c.Pos.MY+c.Height)
}

// MissionState 任务状态（包含地图，资源，进度，对象等）
type MissionState struct {
	Mission md.Mission
	// 任务关卡状态
	MissionStatus MissionStatus
	// 任务关卡元数据
	MissionMD md.MissionMetadata
	// 屏幕布局
	Layout layout.ScreenLayout
	// 相机
	Camera Camera
	// 当前玩家
	CurPlayer faction.Player
	// 友军伤害
	FriendlyFire bool
	// 战舰信息
	Ships map[string]*object.BattleShip
	// 已发射的弹药信息（炮弹 / 鱼雷）
	Bullets map[string]*object.Bullet
	// 被选中的战舰信息（Uid）
	SelectedShips []string
	// 战舰尾流
	ShipTrails []*object.ShipTrail
}

// NewMissionState ...
func NewMissionState(mission md.Mission) *MissionState {
	missionMD := md.Get(mission)
	misLayout := layout.NewScreenLayout()
	// 初始化战舰
	ships := map[string]*object.BattleShip{}
	for _, shipMD := range missionMD.InitShips {
		ship := object.NewShip(shipMD.ShipName, shipMD.Pos, shipMD.Rotation, faction.HumanAlpha)
		ships[ship.Uid] = ship
	}
	return &MissionState{
		Mission:       mission,
		MissionStatus: MissionRunning,
		MissionMD:     missionMD,
		Layout:        misLayout,
		Camera: Camera{
			Pos: missionMD.InitCameraPos,
			// 地图资源，多展示一行 & 列，避免出现黑边
			Width:  misLayout.Camera.Width/mapblock.BlockSize + 1,
			Height: misLayout.Camera.Height/mapblock.BlockSize + 1,
		},
		CurPlayer: faction.HumanAlpha,
		// TODO 后续允许设置开启友军伤害，游戏性 up！
		FriendlyFire:  false,
		Ships:         ships,
		Bullets:       map[string]*object.Bullet{},
		SelectedShips: []string{},
	}
}
