package state

import (
	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/layout"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	obj "github.com/narasux/jutland/pkg/mission/object"
)

// Camera 相机（当前视野）
type Camera struct {
	// 相机左上角位置
	Pos    obj.MapPos
	Width  int
	Height int
}

// Contains 判断坐标是否在视野内
func (c *Camera) Contains(pos obj.MapPos) bool {
	return !(pos.MX < c.Pos.MX || pos.MX > c.Pos.MX+c.Width || pos.MY < c.Pos.MY || pos.MY > c.Pos.MY+c.Height)
}

// GameOptions 游戏选项
type GameOptions struct {
	// 友军伤害
	FriendlyFire bool
	// 展示状态（HP / 武器禁用）
	ForceDisplayState bool
	// 展示伤害数值
	DisplayDamageNumber bool
	// 地图展示模式
	MapDisplayMode MapDisplayMode
}

// UserInputBlocked 特定情况下，屏蔽用户输入
func (opts *GameOptions) UserInputBlocked() bool {
	return opts.MapDisplayMode == MapDisplayModeFull
}

// MissionState 任务状态（包含地图，资源，进度，对象等）
type MissionState struct {
	Mission string
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
	Ships map[string]*obj.BattleShip
	// 战舰 Uid 生成器
	ShipUidGenerators map[faction.Player]*obj.ShipUidGenerator
	// 被选中的战舰信息（Uid）
	SelectedShips []string
	// 当前被选中的编组
	SelectedGroupID obj.GroupID
	// 被摧毁的战舰
	DestroyedShips []*obj.BattleShip
	// 战舰尾流
	Trails []*obj.Trail
	// 正在前进的弹药信息（炮弹 / 鱼雷）
	ForwardingBullets []*obj.Bullet
	// 游戏标识
	GameMarks map[obj.MarkID]*obj.Mark
}

// NewMissionState ...
func NewMissionState(mission string) *MissionState {
	missionMD := md.Get(mission)
	misLayout := layout.NewScreenLayout()
	// 初始化战舰 Uid 生成器
	shipUidGenerators := map[faction.Player]*obj.ShipUidGenerator{
		faction.HumanAlpha:    obj.NewShipUidGenerator(faction.HumanAlpha),
		faction.ComputerAlpha: obj.NewShipUidGenerator(faction.ComputerAlpha),
	}
	// 初始化战舰
	ships := map[string]*obj.BattleShip{}
	for _, shipMD := range missionMD.InitShips {
		ship := obj.NewShip(
			shipUidGenerators[shipMD.BelongPlayer],
			shipMD.ShipName,
			shipMD.Pos,
			shipMD.Rotation,
			shipMD.BelongPlayer,
		)
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
			Width:  misLayout.Width/constants.MapBlockSize + 1,
			Height: misLayout.Height/constants.MapBlockSize + 1,
		},
		CurPlayer: faction.HumanAlpha,
		GameOpts: GameOptions{
			// 默认展示游戏单位的状态
			ForceDisplayState: true,
			// TODO 后续允许设置开启友军伤害，游戏性 up！但是如何解决敌人打死自己人？
			FriendlyFire: false,
			// 默认展示伤害数值
			DisplayDamageNumber: true,
			// 默认不展示地图
			MapDisplayMode: MapDisplayModeNone,
		},
		IsAreaSelecting:   false,
		IsGrouping:        false,
		ShipUidGenerators: shipUidGenerators,
		Ships:             ships,
		SelectedShips:     []string{},
		SelectedGroupID:   obj.GroupIDNone,
		DestroyedShips:    []*obj.BattleShip{},
		Trails:            []*obj.Trail{},
		ForwardingBullets: []*obj.Bullet{},
		GameMarks:         map[obj.MarkID]*obj.Mark{},
	}
}
