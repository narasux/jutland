package manager

import "github.com/narasux/jutland/pkg/config"

func (m *MissionManager) updateShipAnimations() {
	for _, ship := range m.state.Arena.Ships {
		ship.AdvanceAnimation(config.G.SpeedMultiplier)
	}
}
