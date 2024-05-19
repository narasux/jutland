package state

import (
	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/layout"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
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

// GameOptions 游戏选项
type GameOptions struct {
	// 友军伤害
	FriendlyFire bool
	// 展示状态（HP / 武器禁用）
	ForceDisplayState bool
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
	// 游戏选项
	GameOpts GameOptions

	// 是否正在选择区域
	IsAreaSelecting bool
	// 是否正在编组
	IsGrouping bool

	// 战舰信息
	Ships map[string]*object.BattleShip
	// 被摧毁的战舰
	DestroyedShips []*object.BattleShip
	// 正在前进的弹药信息（炮弹 / 鱼雷）
	ForwardingBullets []*object.Bullet
	// 已到达预期位置的弹药信息（炮弹 / 鱼雷）
	ArrivedBullets []*object.Bullet
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
		ship := object.NewShip(shipMD.ShipName, shipMD.Pos, shipMD.Rotation, shipMD.BelongPlayer)
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
			Width:  misLayout.Camera.Width/constants.MapBlockSize + 1,
			Height: misLayout.Camera.Height/constants.MapBlockSize + 1,
		},
		CurPlayer: faction.HumanAlpha,
		GameOpts: GameOptions{
			// 默认展示游戏单位的状态
			ForceDisplayState: true,
			// TODO 后续允许设置开启友军伤害，游戏性 up！
			FriendlyFire: false,
		},
		Ships:             ships,
		ForwardingBullets: []*object.Bullet{},
		ArrivedBullets:    []*object.Bullet{},
		SelectedShips:     []string{},
	}
}
