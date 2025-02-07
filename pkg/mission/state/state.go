package state

import (
	"sort"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	obj "github.com/narasux/jutland/pkg/mission/object"
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

// MissionState 任务状态（包含地图，资源，进度，对象等）
type MissionState struct {
	Mission string
	// 任务关卡状态
	MissionStatus MissionStatus
	// 任务关卡元数据
	MissionMD metadata.MissionMetadata
	// 屏幕布局
	Layout layout.ScreenLayout
	// 相机
	Camera Camera

	// 当前玩家
	CurPlayer faction.Player
	// 当前资金
	CurFunds int64
	// 当前敌人
	// TODO 支持多个敌对势力
	CurEnemy faction.Player
	// 游戏选项
	GameOpts GameOptions

	// 是否正在选择区域
	IsAreaSelecting bool
	// 是否正在编组
	IsGrouping bool

	// 被选中的增援点
	SelectedReinforcePointUid string
	// 增援点信息
	ReinforcePoints map[string]*obj.ReinforcePoint
	// 被选中的增援战舰名称
	SelectedSummonShipName string
	// 油井信息
	OilPlatforms map[string]*obj.OilPlatform
	// 战舰信息（Key: Uid）
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
	// 飞机
	Planes map[string]*obj.Plane
	// 正在前进的弹药信息（炮弹 / 鱼雷）
	ForwardingBullets []*obj.Bullet
	// 游戏标识
	GameMarks map[obj.MarkID]*obj.Mark
}

// CameraPosBorder 获取相机视野边界
func (s *MissionState) CameraPosBorder() (w float64, h float64) {
	w = float64(s.MissionMD.MapCfg.Width - s.Camera.Width - 1)
	h = float64(s.MissionMD.MapCfg.Height - s.Camera.Height - 1)
	return w, h
}

// CountShips 对同类战舰进行计数
func (s *MissionState) Fleet(player faction.Player) Fleet {
	ships := lo.Filter(lo.Values(s.Ships), func(ship *obj.BattleShip, _ int) bool {
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
	shipUidGenerators := map[faction.Player]*obj.ShipUidGenerator{
		faction.HumanAlpha:    obj.NewShipUidGenerator(faction.HumanAlpha),
		faction.ComputerAlpha: obj.NewShipUidGenerator(faction.ComputerAlpha),
	}
	// 初始化战舰
	ships := map[string]*obj.BattleShip{}
	for _, md := range missionMD.InitShips {
		ship := obj.NewShip(
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
	reinforcePoints := map[string]*obj.ReinforcePoint{}
	for _, md := range missionMD.InitReinforcePoints {
		rp := obj.NewReinforcePoint(
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
	oilPlatforms := map[string]*obj.OilPlatform{}
	for _, md := range missionMD.InitOilPlatforms {
		op := obj.NewOilPlatform(md.Pos, md.Radius, md.Yield)
		oilPlatforms[op.Uid] = op
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
			// 默认移动速度
			BaseMoveSpeed: 0.25,
		},
		CurPlayer: faction.HumanAlpha,
		CurFunds:  missionMD.InitFunds,
		CurEnemy:  faction.ComputerAlpha,
		GameOpts: GameOptions{
			// 默认展示游戏单位的状态
			ForceDisplayState: true,
			// TODO 后续允许设置开启友军伤害，游戏性 up！但是如何解决敌人打死自己人？
			FriendlyFire: false,
			// 默认展示伤害数值
			DisplayDamageNumber: true,
			// 默认缩放 1 倍
			Zoom: 1,
		},
		IsAreaSelecting:           false,
		IsGrouping:                false,
		SelectedReinforcePointUid: selectedReinforcePointUid,
		ReinforcePoints:           reinforcePoints,
		OilPlatforms:              oilPlatforms,
		ShipUidGenerators:         shipUidGenerators,
		Ships:                     ships,
		SelectedShips:             []string{},
		SelectedGroupID:           obj.GroupIDNone,
		DestroyedShips:            []*obj.BattleShip{},
		Trails:                    []*obj.Trail{},
		ForwardingBullets:         []*obj.Bullet{},
		Planes:                    map[string]*obj.Plane{
			//"1": obj.NewPlane("F4F-3", obj.NewMapPos(4, 4), 90),
			//"2": obj.NewPlane("SBD-3", obj.NewMapPos(5, 5), 180),
			//"3": obj.NewPlane("TBD-1", obj.NewMapPos(6, 6), 270),
			//"4": obj.NewPlane("A6M_0", obj.NewMapPos(7, 7), 45),
			//"5": obj.NewPlane("D3A_99", obj.NewMapPos(8, 8), 135),
			//"6": obj.NewPlane("B5N2_97", obj.NewMapPos(9, 9), 225),
		},
		GameMarks: map[obj.MarkID]*obj.Mark{},
	}
}
