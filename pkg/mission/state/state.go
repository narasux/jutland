package state

import (
	"sort"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/utils/layout"
)

type MissionStatus string

const (
	// MissionRunning 任务进行中
	MissionRunning MissionStatus = "running"
	// MissionSuccess 任务成功
	MissionSuccess MissionStatus = "success"
	// MissionFailed 任务失败
	MissionFailed MissionStatus = "failed"
	// MissionPaused 任务暂停
	MissionPaused MissionStatus = "paused"
	// MissionInMap 任务地图
	MissionInMap MissionStatus = "inMap"
	// MissionInTerminal 任务终端
	MissionInTerminal MissionStatus = "inTerminal"
	// MissionInBuilding 任务建筑（增援点等）
	MissionInBuilding MissionStatus = "inBuilding"
)

// MissionCoreState 任务核心状态
type MissionCoreState struct {
	Mission string
	// 任务关卡状态
	MissionStatus MissionStatus
	// 是否正在确认放弃任务
	ConfirmQuitMission bool
	// 任务关卡元数据
	MissionMD metadata.MissionMetadata
}

// MissionViewState 任务视图状态
type MissionViewState struct {
	// 屏幕布局
	Layout layout.ScreenLayout
	// 相机
	Camera Camera
}

// MissionPlayerState 任务玩家状态
type MissionPlayerState struct {
	// 当前玩家
	CurPlayer faction.Player
	// 当前资金
	CurFunds int64
	// 当前敌人
	// TODO 支持多个敌对势力
	CurEnemy faction.Player
}

// MissionInteractionState 任务交互状态
type MissionInteractionState struct {
	// 是否正在选择区域
	IsAreaSelecting bool
	// 是否正在编组
	IsGrouping bool

	// 被选中的增援点
	SelectedReinforcePointUid string
	// 被选中的增援战舰名称
	SelectedSummonShipName string
	// 被选中的战舰信息（Uid）
	SelectedShips []string
	// 当前被选中的编组
	SelectedGroupID object.GroupID
}

// MissionArenaState 任务战场对象状态
type MissionArenaState struct {
	// 增援点信息
	ReinforcePoints map[string]*objBuilding.ReinforcePoint
	// 油井信息
	OilPlatforms map[string]*objBuilding.OilPlatform
	// 战舰信息（Key: Uid）
	Ships map[string]*objUnit.BattleShip
	// 战舰 Uid 生成器
	ShipUidGenerators map[faction.Player]*objUnit.ShipUidGenerator
	// 被摧毁的战舰
	DestroyedShips []*objUnit.BattleShip
	// 被摧毁的战机
	DestroyedPlanes []*objUnit.Plane
	// 战舰尾流
	Trails []*objTrail.Trail
	// 飞机
	Planes map[string]*objUnit.Plane
	// 正在前进的弹药信息（炮弹 / 鱼雷）
	ForwardingBullets []*objBullet.Bullet
}

// MissionUIState 任务界面状态
type MissionUIState struct {
	// 游戏标识
	GameMarks map[objMark.ID]*objMark.Mark
	// 右侧任务侧栏是否展开
	SidebarExpanded bool
	// 右侧任务侧栏本帧是否占用鼠标输入
	SidebarConsumesCursor bool
	// 显示集结线的增援点 Uid（游戏主地图中点击增援点时设置）
	ShowRallyLinePointUid string
	// 集结点设置失败计数器（用于短暂闪烁提示）
	RallySetFailedTick int
	// 游戏选项
	GameOpts GameOptions
	// DebugFlags 调试标识
	DebugFlags DebugFlags
}

// MissionState 任务状态（包含地图，资源，进度，对象等）
type MissionState struct {
	Core        MissionCoreState
	View        MissionViewState
	Player      MissionPlayerState
	Interaction MissionInteractionState
	Arena       MissionArenaState
	UI          MissionUIState
}

// CameraPosBorder 获取相机视野边界
func (s *MissionState) CameraPosBorder() (w float64, h float64) {
	w = float64(s.Core.MissionMD.MapCfg.Width - s.View.Camera.Width - 1)
	h = float64(s.Core.MissionMD.MapCfg.Height - s.View.Camera.Height - 1)
	return w, h
}

// CountShips 对同类战舰进行计数
func (s *MissionState) Fleet(player faction.Player) Fleet {
	ships := lo.Filter(lo.Values(s.Arena.Ships), func(ship *objUnit.BattleShip, _ int) bool {
		return ship.BelongPlayer == player
	})

	classMap := map[string]ShipClass{}
	for _, ship := range ships {
		if cls, ok := classMap[ship.Name]; ok {
			cls.Total++
			classMap[ship.Name] = cls
		} else {
			classMap[ship.Name] = ShipClass{Total: 1, Kind: ship}
		}
	}

	classes := lo.Values(classMap)
	// 按照吨位从大到小排列
	sort.Slice(classes, func(i, j int) bool {
		return classes[i].Kind.Tonnage > classes[j].Kind.Tonnage
	})
	return Fleet{Player: player, Total: len(ships), Classes: classes}
}

// NewMissionState ...
func NewMissionState(mission string) *MissionState {
	missionMD := metadata.Get(mission)
	misLayout := layout.NewScreenLayout()
	// 初始化战舰 Uid 生成器
	shipUidGenerators := map[faction.Player]*objUnit.ShipUidGenerator{
		faction.HumanAlpha:    objUnit.NewShipUidGenerator(faction.HumanAlpha),
		faction.ComputerAlpha: objUnit.NewShipUidGenerator(faction.ComputerAlpha),
	}
	// 初始化战舰
	ships := map[string]*objUnit.BattleShip{}
	for _, md := range missionMD.InitShips {
		ship := objUnit.NewShip(
			shipUidGenerators[md.BelongPlayer],
			md.ShipName,
			md.Pos,
			md.Rotation,
			md.BelongPlayer,
		)
		ships[ship.Uid] = ship
	}
	// 初始化增援点
	selectedReinforcePointUid := ""
	reinforcePoints := map[string]*objBuilding.ReinforcePoint{}
	for _, md := range missionMD.InitReinforcePoints {
		rp := objBuilding.NewReinforcePoint(
			md.Pos,
			md.Rotation,
			md.RallyPos,
			md.BelongPlayer,
			md.MaxOncomingShip,
			md.ProvidedShipNames,
		)
		reinforcePoints[rp.Uid] = rp
		if rp.BelongPlayer == faction.HumanAlpha {
			selectedReinforcePointUid = rp.Uid
		}
	}
	// 初始化油井
	oilPlatforms := map[string]*objBuilding.OilPlatform{}
	for _, md := range missionMD.InitOilPlatforms {
		op := objBuilding.NewOilPlatform(md.Pos, md.Radius, md.Yield)
		oilPlatforms[op.Uid] = op
	}

	ms := &MissionState{
		Core: MissionCoreState{
			Mission:            mission,
			MissionStatus:      MissionRunning,
			ConfirmQuitMission: false,
			MissionMD:          missionMD,
		},
		View: MissionViewState{
			Layout: misLayout,
			Camera: Camera{
				Pos: missionMD.InitCameraPos,
				// 地图资源，多展示一行 & 列，避免出现黑边
				Width:  misLayout.Width/constants.MapBlockSize + 1,
				Height: misLayout.Height/constants.MapBlockSize + 1,
				// 默认移动速度
				BaseMoveSpeed: 0.25,
			},
		},
		Player: MissionPlayerState{
			CurPlayer: faction.HumanAlpha,
			CurFunds:  missionMD.InitFunds,
			CurEnemy:  faction.ComputerAlpha,
		},
		Interaction: MissionInteractionState{
			IsAreaSelecting:           false,
			IsGrouping:                false,
			SelectedReinforcePointUid: selectedReinforcePointUid,
			SelectedSummonShipName:    "",
			SelectedShips:             []string{},
			SelectedGroupID:           object.GroupIDNone,
		},
		Arena: MissionArenaState{
			ReinforcePoints:   reinforcePoints,
			OilPlatforms:      oilPlatforms,
			ShipUidGenerators: shipUidGenerators,
			Ships:             ships,
			DestroyedShips:    []*objUnit.BattleShip{},
			DestroyedPlanes:   []*objUnit.Plane{},
			Trails:            []*objTrail.Trail{},
			ForwardingBullets: []*objBullet.Bullet{},
			Planes:            map[string]*objUnit.Plane{},
		},
		UI: MissionUIState{
			GameMarks:             map[objMark.ID]*objMark.Mark{},
			SidebarExpanded:       false,
			SidebarConsumesCursor: false,
			ShowRallyLinePointUid: "",
			RallySetFailedTick:    0,
			GameOpts: GameOptions{
				// 默认展示游戏单位的状态
				ForceDisplayState: true,
				// TODO 后续允许设置开启友军伤害，游戏性 up！但是如何解决敌人打死自己人？
				FriendlyFire: false,
				// 默认展示伤害数值
				DisplayDamageNumber: true,
				// 默认缩放 1 倍
				Zoom: DefaultZoom(),
			},
		},
	}
	ms.RefreshCameraSize()
	return ms
}
