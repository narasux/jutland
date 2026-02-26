package manager

import (
	"github.com/narasux/jutland/pkg/mission/warfog"
)

// updateFogOfWar 更新战争迷雾视野
func (m *MissionManager) updateFogOfWar() {
	fog := m.state.FogOfWar
	if fog == nil {
		return
	}

	// 标记脏位，确保视野在每帧都更新（因为战舰可能在移动）
	// TODO: 后续可以优化为只在战舰移动时标记脏位
	fog.MarkDirty()

	// 只有脏标记为 true 时才重新计算
	if !fog.IsDirty() {
		return
	}

	// 使用视野计算器更新视野
	calc := warfog.NewVisibilityCalculator()
	calc.UpdateVisibility(fog, m.state.Ships, m.state.CurPlayer)
}
