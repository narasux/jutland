package mission

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/drawer"
	md "github.com/narasux/jutland/pkg/mission/metadata"
	"github.com/narasux/jutland/pkg/mission/state"
)

// MissionManager 任务管理器
type MissionManager struct {
	state  *state.MissionState
	drawer *drawer.Drawer
}

// NewManager ...
func NewManager(mission md.Mission) *MissionManager {
	return &MissionManager{
		state: state.NewMissionState(
			mission, md.Get(mission).InitCameraPos,
		),
		drawer: drawer.NewDrawer(),
	}
}

// Draw 绘制任务图像
func (m *MissionManager) Draw(screen *ebiten.Image) {
	m.drawer.Draw(screen, m.state)
}

func (m *MissionManager) Update() (state.MissionStatus, error) {
	if err := m.state.Next(); err != nil {
		return state.MissionError, err
	}
	return m.state.MissionStatus, nil
}
