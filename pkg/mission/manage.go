package mission

import "github.com/hajimehoshi/ebiten/v2"

// MissionManager 任务管理器
type MissionManager struct {
	state  *MissionState
	drawer *Drawer
}

// NewManager ...
func NewManager(mission Mission) *MissionManager {
	return &MissionManager{
		state: NewMissionState(
			mission,
			missionMetadata[mission].InitPos,
		),
		drawer: NewDrawer(),
	}
}

// Draw 绘制任务图像
func (m *MissionManager) Draw(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state)
}

func (m *MissionManager) Update() (MissionStatus, error) {
	if err := m.state.Next(); err != nil {
		return MissionError, err
	}
	return m.state.MissionStatus, nil
}
