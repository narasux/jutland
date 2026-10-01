package manager

import (
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
)

// updateAutoScout 电脑基地在迷雾开启时自动派一架侦察机。玩家基地默认不自动派。
func (m *MissionManager) updateAutoScout() {
	if m.state == nil || !m.state.Core.FogOfWar || m.instructionSet == nil {
		return
	}
	for _, ship := range m.state.Arena.Ships {
		m.launchAutoScout(&ship.Aircraft, ship)
	}
	for _, field := range m.state.Arena.Airfields {
		m.launchAutoScout(&field.Aircraft, field)
	}
}

func (m *MissionManager) launchAutoScout(aircraft *objUnit.ShipAircraft, base objUnit.AircraftBase) {
	owner := base.BaseBelongPlayer()
	// 玩家基地默认不自动派。这一侧迷雾被关掉时也不派，因为没有未探索可揭。
	if owner == m.state.Player.CurPlayer || !m.state.UsesFog(owner) {
		return
	}
	// 起飞后甲板上还要留一架。天上已经有一架自动搜索就不再派。
	if !aircraft.CanSearch() || m.countAutoScouts(base.BaseUid()) >= 1 {
		return
	}
	plane := aircraft.TakeOffSearch(base)
	if plane == nil {
		return
	}
	m.state.Arena.PutPlane(plane)
	// 第一段飞向能新揭开格子最多、又比较近的前线点。没有前线就先停在起飞点。
	point := plane.CurPos
	if vision := m.state.Player.Visions[owner]; vision != nil {
		if next, ok := vision.BestUncover(plane.CurPos, plane.SightRange); ok {
			point = next
		}
	}
	m.instructionSet.Add(instr.NewPlaneScout(plane.Uid, point, false))
}

// countAutoScouts 数这个基地已经有几架自动搜索机在天上。
// 没有侦察机的基地会拿打击机顶上，所以按搜索指令数，而不是按机型数。
func (m *MissionManager) countAutoScouts(baseUid string) int {
	count := 0
	for _, ins := range m.instructionSet.Items() {
		scout, ok := ins.(*instr.PlaneScout)
		if !ok || scout.Manual() {
			continue
		}
		plane := m.state.Arena.Planes[scout.PlaneUid()]
		if plane == nil || plane.CurHP <= 0 || plane.BelongShip != baseUid {
			continue
		}
		count++
	}
	return count
}
