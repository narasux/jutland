package state

import (
	"sort"

	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/object"
	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objBullet "github.com/narasux/jutland/pkg/mission/object/bullet"
	objExplosion "github.com/narasux/jutland/pkg/mission/object/explosion"
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
	// 当前游戏拍。武器装填和电脑决策读这一拍，不读墙钟。
	SimTick int64
	// 本局是否启用迷雾，开局从游戏设置拷贝。关闭时不建视野图。
	FogOfWar bool
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
	// 当前玩家本局忽略迷雾。只由 black sheep wall 置位，不改游戏设置。
	IgnoreFog bool
	// 各玩家的已探索和可见。迷雾关闭时保持 nil。
	Visions map[faction.Player]*FactionVision
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
	// 被选中的陆地机场（Uid）：选中后在主地图展示跑道方位线（调试秘籍）
	SelectedAirfieldUid string
	// 被选中的战舰信息（Uid）
	SelectedShips []string
	// 当前在单位面板中聚焦的战舰 Uid；多选时与 SelectedShips 分离维护。
	FocusedShipUid string
	// 当前被选中的编组
	SelectedGroupID object.GroupID
}

// MissionArenaState 任务战场对象状态
type MissionArenaState struct {
	// 增援点信息
	ReinforcePoints map[string]*objBuilding.ReinforcePoint
	// 油井信息
	OilPlatforms map[string]*objBuilding.OilPlatform
	// 陆地机场信息
	Airfields map[string]*objBuilding.Airfield
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
	// 火箭弹等局部爆炸效果
	Explosions []*objExplosion.Explosion
	// 飞机
	Planes map[string]*objUnit.Plane
	// 按 UID 排好的绘制名单。只在增删时更新。
	orderedShips  []*objUnit.BattleShip
	orderedPlanes []*objUnit.Plane
	// 正在前进的弹药信息（炮弹 / 鱼雷）
	ForwardingBullets []*objBullet.Bullet
}

// MissionUIState 任务界面状态
type MissionUIState struct {
	// 游戏标识
	GameMarks map[objMark.ID]*objMark.Mark
	// 右侧任务侧栏是否展开
	SidebarExpanded bool
	// 任一任务 UI 本帧是否占用鼠标输入
	UIConsumesCursor bool
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

// fogOfWarFromSettings 开局拷贝设置。配置还没加载时视为关闭，避免任务里碰到空指针。
func fogOfWarFromSettings() bool {
	return config.G != nil && config.G.EnableFogOfWar
}

// UsesFog 该玩家这一拍要不要跑迷雾。
// 热路径先调用它：返回 false 时保持全图可见，不分配视野图，也不扫描单位。
func (s *MissionState) UsesFog(player faction.Player) bool {
	if !s.Core.FogOfWar {
		return false
	}
	// 秘籍只揭开当前玩家，电脑仍按迷雾决策。
	if player == s.Player.CurPlayer && s.Player.IgnoreFog {
		return false
	}
	return true
}

// allocateVisions 只在本局开启迷雾时为双方建图。关闭时保持 nil，热路径不再扫描。
func (s *MissionState) allocateVisions(width, height int) {
	if !s.Core.FogOfWar || width <= 0 || height <= 0 {
		return
	}
	s.Player.Visions = map[faction.Player]*FactionVision{
		faction.HumanAlpha:    NewFactionVision(width, height),
		faction.ComputerAlpha: NewFactionVision(width, height),
	}
}

// CameraPosBorder 获取相机视野边界
func (s *MissionState) CameraPosBorder() (w float64, h float64) {
	w = float64(s.Core.MissionMD.MapCfg.Width - s.View.Camera.Width - 1)
	h = float64(s.Core.MissionMD.MapCfg.Height - s.View.Camera.Height - 1)
	return w, h
}

// FindAircraftBase 按 舰船 → 陆地机场 的顺序解析飞机所属基地（BelongShip）。
// 返回的基地可作为起降几何计算的统一入参；基地不存在（航母被击沉）时返回 false。
func (s *MissionState) FindAircraftBase(uid string) (objUnit.AircraftBase, bool) {
	if ship, ok := s.Arena.Ships[uid]; ok {
		return ship, true
	}
	if airfield, ok := s.Arena.Airfields[uid]; ok {
		return airfield, true
	}
	return nil, false
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

// NewMissionState 创建任务状态，并把配置层 P1/P2 映射为运行时 HA/CA。
func NewMissionState(mission string, playerSide faction.Side) *MissionState {
	missionMD := metadata.Get(mission)
	playerSide = missionMD.NormalizePlayerSide(playerSide)
	misLayout := layout.NewScreenLayout()
	// 初始化战舰 Uid 生成器
	shipUidGenerators := map[faction.Player]*objUnit.ShipUidGenerator{
		faction.HumanAlpha:    objUnit.NewShipUidGenerator(faction.HumanAlpha),
		faction.ComputerAlpha: objUnit.NewShipUidGenerator(faction.ComputerAlpha),
	}
	// 初始化战舰
	ships := map[string]*objUnit.BattleShip{}
	for _, md := range missionMD.InitShips {
		runtimePlayer := md.BelongSide.RuntimePlayer(playerSide)
		ship := objUnit.NewShip(
			shipUidGenerators[runtimePlayer],
			md.ShipName,
			md.Pos,
			md.Rotation,
			runtimePlayer,
		)
		ships[ship.Uid] = ship
	}
	// 初始化增援点
	selectedReinforcePointUid := ""
	reinforcePoints := map[string]*objBuilding.ReinforcePoint{}
	for _, md := range missionMD.InitReinforcePoints {
		runtimePlayer := md.BelongSide.RuntimePlayer(playerSide)
		rp := objBuilding.NewReinforcePoint(
			md.Pos,
			md.Rotation,
			md.RallyPos,
			runtimePlayer,
			md.MaxOncomingShip,
			md.ProvidedShipNames,
		)
		reinforcePoints[rp.Uid] = rp
		if md.BelongSide == playerSide {
			selectedReinforcePointUid = rp.Uid
		}
	}
	// 初始化油井
	oilPlatforms := map[string]*objBuilding.OilPlatform{}
	for _, md := range missionMD.InitOilPlatforms {
		op := objBuilding.NewOilPlatform(md.Pos, md.Radius, md.Yield)
		oilPlatforms[op.Uid] = op
	}
	// 初始化陆地机场与初始机库库存（全局功能开关关闭时不生成）
	airfields := map[string]*objBuilding.Airfield{}
	if config.G == nil || config.G.EnableLandAirfield {
		for _, md := range missionMD.InitAirfields {
			runtimePlayer := md.BelongSide.RuntimePlayer(playerSide)
			groups := make([]objUnit.PlaneGroup, 0, len(md.PlaneGroups))
			for _, g := range md.PlaneGroups {
				groups = append(groups, objUnit.PlaneGroup{
					Name:     g.Name,
					MaxCount: g.MaxCount,
					CurCount: 0,
				})
			}
			af := objBuilding.NewAirfield(
				md.Pos,
				md.Rotation,
				md.RunwayLength,
				md.RunwayWidth,
				runtimePlayer,
				md.TakeOffTime,
				groups,
			)
			airfields[af.Uid] = af
			// 按配置重建跑道起飞点数（1 = 单机串行，用于重轰炸机间隔起飞）
			if md.TakeoffPoints > 0 {
				af.SetTakeoffPoints(md.TakeoffPoints)
			}
			// 初始库存（initCount，缺省为 maxCount）：飞机待命于机库，
			// 起飞时直接在跑道起点刷新
			for _, g := range md.PlaneGroups {
				for i := int64(0); i < g.InitCount; i++ {
					af.StockPlane(g.Name)
				}
			}
		}
	}

	ms := &MissionState{
		Core: MissionCoreState{
			Mission:            mission,
			MissionStatus:      MissionRunning,
			ConfirmQuitMission: false,
			MissionMD:          missionMD,
			FogOfWar:           fogOfWarFromSettings(),
		},
		View: MissionViewState{
			Layout: misLayout,
			Camera: Camera{
				Pos: missionMD.CameraPosForPlayerSide(playerSide),
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
			SelectedAirfieldUid:       "",
			SelectedShips:             []string{},
			FocusedShipUid:            "",
			SelectedGroupID:           object.GroupIDNone,
		},
		Arena: MissionArenaState{
			ReinforcePoints:   reinforcePoints,
			OilPlatforms:      oilPlatforms,
			Airfields:         airfields,
			ShipUidGenerators: shipUidGenerators,
			Ships:             ships,
			DestroyedShips:    []*objUnit.BattleShip{},
			DestroyedPlanes:   []*objUnit.Plane{},
			Trails:            []*objTrail.Trail{},
			Explosions:        []*objExplosion.Explosion{},
			ForwardingBullets: []*objBullet.Bullet{},
			Planes:            map[string]*objUnit.Plane{},
		},
		UI: MissionUIState{
			GameMarks:             map[objMark.ID]*objMark.Mark{},
			SidebarExpanded:       false,
			UIConsumesCursor:      false,
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
	if missionMD.MapCfg != nil {
		ms.allocateVisions(missionMD.MapCfg.Width, missionMD.MapCfg.Height)
	}
	return ms
}
