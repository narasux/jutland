package manager

import (
	"github.com/narasux/jutland/pkg/mission/faction"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// updateVision 在开火和航空择敌之前刷新可见格子。
// 本局没开迷雾时直接返回，不扫描单位。某一侧被秘籍揭开时跳过该侧。
func (m *MissionManager) updateVision() {
	// 设置关闭时不扫描单位，开火和择敌继续看整张战场。
	if m.state == nil || !m.state.Core.FogOfWar {
		return
	}
	for player, vision := range m.state.Player.Visions {
		// 秘籍揭开的一侧本拍不再盖格，电脑另一侧照常算。
		if !m.state.UsesFog(player) {
			continue
		}
		// 先清掉上一拍的可见和软边亮度，已探索留着。
		vision.BeginFrame()
		// 再用本方活着的舰船、在空飞机和机场盖上这一拍的圆。
		m.stampPlayerVision(player, vision)
		// 可见并进已探索，并生成给绘制用的蒙层。
		vision.FinishFrame()
		// 最后根据这一拍看见的敌舰刷新接触，并清掉过期锁定。
		m.refreshContacts(player, vision)
	}
}

func (m *MissionManager) stampPlayerVision(player faction.Player, vision *state.FactionVision) {
	// 活着的舰船按自己的视距照两层：逻辑格给判定，软边亮度给蒙层。
	for _, ship := range m.state.Arena.Ships {
		if ship.BelongPlayer != player || ship.CurHP <= 0 || ship.SightRange <= 0 {
			continue
		}
		vision.Stamp(ship.CurPos.RX, ship.CurPos.RY, ship.SightRange)
		vision.StampLight(ship.CurPos.RX, ship.CurPos.RY, ship.SightRange)
	}
	// 只算还在空中的飞机。甲板滑跑和着舰回收跟着载舰，不再单独照。
	for _, plane := range m.state.Arena.Planes {
		if plane.BelongPlayer != player || !plane.ProvidesSight() || plane.SightRange <= 0 {
			continue
		}
		vision.Stamp(plane.CurPos.RX, plane.CurPos.RY, plane.SightRange)
		vision.StampLight(plane.CurPos.RX, plane.CurPos.RY, plane.SightRange)
	}
	// 停用只关起飞和生产，机场本身仍照固定半径。
	for _, airfield := range m.state.Arena.Airfields {
		if airfield.BelongPlayer != player {
			continue
		}
		vision.Stamp(airfield.Pos.RX, airfield.Pos.RY, objUnit.SightRangeAirfield)
		vision.StampLight(airfield.Pos.RX, airfield.Pos.RY, objUnit.SightRangeAirfield)
	}
}

func (m *MissionManager) refreshContacts(player faction.Player, vision *state.FactionVision) {
	if vision.Contacts == nil {
		vision.Contacts = map[string]state.Contact{}
	}
	now := m.state.Core.SimTick
	ttl := objUnit.ReloadTicks(30)
	// 这一拍还看得见的敌舰刷新位置，并把记忆再延长 30 秒任务时间。
	// 看得见它沉了就立刻忘掉，不把沉船留在接触里。
	for _, ship := range m.state.Arena.Ships {
		if ship.BelongPlayer == player {
			continue
		}
		if !vision.VisibleAt(ship.CurPos.MX, ship.CurPos.MY) {
			continue
		}
		if ship.CurHP <= 0 {
			delete(vision.Contacts, ship.Uid)
			continue
		}
		vision.Contacts[ship.Uid] = state.Contact{
			RX: ship.CurPos.RX, RY: ship.CurPos.RY, ExpireTick: now + ttl,
		}
	}
	// 没被刷新、并且到点的接触删掉。看不见但还没过期的先留着。
	for uid, contact := range vision.Contacts {
		if contact.ExpireTick <= now {
			delete(vision.Contacts, uid)
		}
	}
	// 锁定跟着接触走。目标还活着并且看得见就保留；接触没了就清掉锁定。
	for _, ship := range m.state.Arena.Ships {
		if ship.BelongPlayer != player || ship.AttackTarget == "" {
			continue
		}
		contact, ok := vision.Contacts[ship.AttackTarget]
		target := m.state.Arena.Ships[ship.AttackTarget]
		if target != nil && target.CurHP > 0 && vision.VisibleAt(target.CurPos.MX, target.CurPos.MY) {
			continue
		}
		if !ok || contact.ExpireTick <= now {
			ship.AttackTarget = ""
		}
	}
}
