package state

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	"github.com/narasux/jutland/pkg/mission/action"
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

	// 战舰信息
	Ships map[string]*object.BattleShip
	// 已发射的弹丸信息
	Bullets map[string]*object.Bullet

	// 任务日志 TODO 考虑直接用 chan ?
	Logs []string
}

// NewMissionState ...
func NewMissionState(mission md.Mission, initPos object.MapPos) *MissionState {
	misLayout := layout.NewScreenLayout()
	missionMD := md.Get(mission)
	// 初始化战舰
	ships := map[string]*object.BattleShip{}
	for _, shipMD := range missionMD.InitShips {
		ship := object.NewShip(shipMD.ShipName, shipMD.Pos, shipMD.Rotate)
		ships[ship.Uid] = ship
	}
	return &MissionState{
		Mission:       mission,
		MissionStatus: MissionRunning,
		MissionMD:     missionMD,
		Layout:        misLayout,
		Camera: Camera{
			Pos: initPos,
			// 地图资源，多展示一行 & 列，避免出现黑边
			Width:  misLayout.Width/mapblock.BlockSize + 1,
			Height: misLayout.Height/mapblock.BlockSize + 1,
		},
		Ships:   ships,
		Bullets: map[string]*object.Bullet{},
	}
}

// Next 计算下一帧任务状态
func (s *MissionState) Next() error {
	s.updateNextBullets()
	s.updateNextShips()
	s.updateNextCameraPosition()
	s.updateNextMissionStatus()
	return nil
}

// TODO 计算下一帧战舰状态
func (s *MissionState) updateNextShips() {
}

// TODO 计算下一帧弹丸状态
func (s *MissionState) updateNextBullets() {
}

// 计算下一帧相机位置
func (s *MissionState) updateNextCameraPosition() {
	switch action.DetectCursorHoverOnGameMap(s.Layout) {
	case action.HoverScreenLeft:
		s.Camera.Pos.MX -= 1
	case action.HoverScreenRight:
		s.Camera.Pos.MX += 1
	case action.HoverScreenTop:
		s.Camera.Pos.MY -= 1
	case action.HoverScreenBottom:
		s.Camera.Pos.MY += 1
	case action.HoverScreenTopLeft:
		s.Camera.Pos.MX -= 1
		s.Camera.Pos.MY -= 1
	case action.HoverScreenTopRight:
		s.Camera.Pos.MX += 1
		s.Camera.Pos.MY -= 1
	case action.HoverScreenBottomLeft:
		s.Camera.Pos.MX -= 1
		s.Camera.Pos.MY += 1
	case action.HoverScreenBottomRight:
		s.Camera.Pos.MX += 1
		s.Camera.Pos.MY += 1
	default:
		// DoNothing
	}

	// 防止超出边界
	s.Camera.Pos.MX = lo.Max([]int{s.Camera.Pos.MX, 0})
	s.Camera.Pos.MY = lo.Max([]int{s.Camera.Pos.MY, 0})
	s.Camera.Pos.MX = lo.Min([]int{s.Camera.Pos.MX, s.MissionMD.MapCfg.Width - s.Camera.Width - 1})
	s.Camera.Pos.MY = lo.Min([]int{s.Camera.Pos.MY, s.MissionMD.MapCfg.Height - s.Camera.Height - 1})
}

// TODO 计算下一帧任务状态
func (s *MissionState) updateNextMissionStatus() {
	switch s.MissionStatus {
	case MissionRunning:
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			s.MissionStatus = MissionPaused
			return
		}
	case MissionPaused:
		if ebiten.IsKeyPressed(ebiten.KeyQ) {
			s.MissionStatus = MissionFailed
			return
		} else if ebiten.IsKeyPressed(ebiten.KeyEnter) {
			s.MissionStatus = MissionRunning
			return
		}
	default:
		s.MissionStatus = MissionRunning
	}
}
