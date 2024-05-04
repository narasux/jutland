package mission

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/samber/lo"

	obj "github.com/narasux/jutland/pkg/object"
	"github.com/narasux/jutland/pkg/resources/images/mapblock"
)

// Camera 相机（当前视野）
type Camera struct {
	// 相机左上角位置
	Pos    obj.MapPos
	Width  int
	Height int
}

// MissionState 任务状态（包含地图，资源，进度，对象等）
type MissionState struct {
	Mission Mission
	// 任务关卡状态
	MissionStatus MissionStatus
	// 任务关卡元数据
	MissionMD MissionMetadata
	// 屏幕布局
	Layout ScreenLayout
	// 相机
	Camera Camera

	// 战舰信息
	Ships []*obj.BattleShip
	// 已发射的弹丸信息
	Bullets []*obj.Bullet
}

// NewMissionState ...
func NewMissionState(mission Mission, initPos obj.MapPos) *MissionState {
	layout := NewScreenLayout()
	missionMD := missionMetadata[mission]
	// 初始化战舰
	ships := []*obj.BattleShip{}
	for _, shipMD := range missionMD.InitShips {
		ships = append(ships, obj.NewShip(shipMD.ShipName, shipMD.Pos, shipMD.Rotate))
	}
	return &MissionState{
		Mission:       mission,
		MissionStatus: MissionRunning,
		MissionMD:     missionMD,
		Layout:        layout,
		Camera: Camera{
			Pos: initPos,
			// 地图资源，多展示一行 & 列，避免出现黑边
			Width:  layout.Width/mapblock.BlockSize + 1,
			Height: layout.Height/mapblock.BlockSize + 1,
		},
		Ships:   ships,
		Bullets: []*obj.Bullet{},
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
	switch detectCursorHoverOnGameMap(s.Layout) {
	case HoverScreenLeft:
		s.Camera.Pos.MX -= 1
	case HoverScreenRight:
		s.Camera.Pos.MX += 1
	case HoverScreenTop:
		s.Camera.Pos.MY -= 1
	case HoverScreenBottom:
		s.Camera.Pos.MY += 1
	case HoverScreenTopLeft:
		s.Camera.Pos.MX -= 1
		s.Camera.Pos.MY -= 1
	case HoverScreenTopRight:
		s.Camera.Pos.MX += 1
		s.Camera.Pos.MY -= 1
	case HoverScreenBottomLeft:
		s.Camera.Pos.MX -= 1
		s.Camera.Pos.MY += 1
	case HoverScreenBottomRight:
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
