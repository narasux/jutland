package mission

import (
	"github.com/samber/lo"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/resources/mapblock"
)

// MissionState 任务状态（包含地图，资源，进度，对象等）
type MissionState struct {
	Mission       Mission
	MissionStatus MissionStatus
	MissionMD     MissionMetadata
	// 任务暂停
	IsPause bool
	// 相机位置
	CameraPos Position
	// 战舰信息
	Ships []*BattleShip
	// 已发射的弹丸信息
	Bullets []*Bullet
}

// NewMissionState ...
func NewMissionState(mission Mission, initPos Position) *MissionState {
	return &MissionState{
		Mission:       mission,
		MissionStatus: MissionRunning,
		MissionMD:     missionMetadata[mission],
		CameraPos:     initPos,
		Ships:         []*BattleShip{},
		Bullets:       []*Bullet{},
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
	// TODO 限制速度，限制的镜头移动太快了
	hover := NewActionDetector().DetectCursorHover()
	switch hover {
	case HoverScreenLeft:
		s.CameraPos.X -= 1
	case HoverScreenRight:
		s.CameraPos.X += 1
	case HoverScreenTop:
		s.CameraPos.Y -= 1
	case HoverScreenBottom:
		s.CameraPos.Y += 1
	case HoverScreenTopLeft:
		s.CameraPos.X -= 1
		s.CameraPos.Y -= 1
	case HoverScreenTopRight:
		s.CameraPos.X += 1
		s.CameraPos.Y -= 1
	case HoverScreenBottomLeft:
		s.CameraPos.X -= 1
		s.CameraPos.Y += 1
	case HoverScreenBottomRight:
		s.CameraPos.X += 1
		s.CameraPos.Y += 1
	default:
		// DoNothing
	}

	// 防止超出边界
	screenWidth, screenHeight := ebiten.Monitor().Size()
	s.CameraPos.X = lo.Max([]int{s.CameraPos.X, 0})
	s.CameraPos.Y = lo.Max([]int{s.CameraPos.Y, 0})
	s.CameraPos.X = lo.Min([]int{s.CameraPos.X, s.MissionMD.MapCfg.Width - screenWidth/mapblock.BlockSize - 1})
	s.CameraPos.Y = lo.Min([]int{s.CameraPos.Y, s.MissionMD.MapCfg.Height - screenHeight/mapblock.BlockSize - 1})
}

// TODO 计算下一帧任务状态
func (s *MissionState) updateNextMissionStatus() {
	s.MissionStatus = MissionRunning
}
