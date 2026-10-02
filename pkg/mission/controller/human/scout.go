package human

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/action"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
)

// handleScout 处理 Shift + 右键的侦察指令。
// 第二个返回值表示这次点击是否已经被侦察指令消费：只要选中列表里有航空基地，
// 即使这一拍派不出飞机也要吞掉点击，不能退化成让航母自己冲向目标点。
func (h *HumanInputHandler) handleScout(ms *state.MissionState) (map[string]instr.Instruction, bool) {
	// 没按 Shift，或点在界面上，就交给原来的右键移动。
	if !ebiten.IsKeyPressed(ebiten.KeyShift) || ms.UI.UIConsumesCursor {
		return nil, false
	}
	pos := action.DetectMouseButtonClickOnMap(ms, ebiten.MouseButtonRight)
	if pos == nil {
		return nil, false
	}
	return scoutAt(ms, *pos)
}

// scoutAt 是侦察指令的落点逻辑，与输入设备解耦，便于测试。
func scoutAt(ms *state.MissionState, target objPos.MapPos) (map[string]instr.Instruction, bool) {
	base, aircraft := nearestSelectedScoutBase(ms, target)
	if aircraft == nil {
		return nil, hasSelectedAirBase(ms)
	}
	// 侦察目标点打个望远镜标记，让玩家知道 Shift + 右键是侦察而不是移动。
	ms.UI.GameMarks[objMark.IDScout] = objMark.NewImg(
		objMark.IDScout, target, textureImg.ScoutTarget, 20,
	)
	out := map[string]instr.Instruction{}
	// 这艘基地已经有一架手动侦察在飞，就只改它的目标点，不再起飞第二架。
	if existing := findManualScout(ms, base.BaseUid()); existing != nil {
		order := instr.NewPlaneScout(existing.Uid, target, true)
		out[order.Uid()] = order
		return out, true
	}
	// 新起飞的这一架标成手动，然后飞向点击点。
	plane := aircraft.TakeOffManualScout(base)
	if plane == nil {
		return nil, true
	}
	plane.ScoutManual = true
	ms.Arena.PutPlane(plane)
	order := instr.NewPlaneScout(plane.Uid, target, true)
	out[order.Uid()] = order
	return out, true
}

// hasSelectedAirBase 当前选中列表里有没有航空基地（航母或机场）。
func hasSelectedAirBase(ms *state.MissionState) bool {
	if ms.Arena.Airfields[ms.Interaction.SelectedAirfieldUid] != nil {
		return true
	}
	for _, uid := range ms.Interaction.SelectedShips {
		if ship := ms.Arena.Ships[uid]; ship != nil && ship.Aircraft.HasPlane {
			return true
		}
	}
	return false
}

func nearestSelectedScoutBase(ms *state.MissionState, target objPos.MapPos) (objUnit.AircraftBase, *objUnit.ShipAircraft) {
	var best objUnit.AircraftBase
	var bestAircraft *objUnit.ShipAircraft
	bestDist := math.MaxFloat64
	consider := func(base objUnit.AircraftBase, aircraft *objUnit.ShipAircraft) {
		if aircraft == nil || (!aircraft.CanManualScout() && findManualScout(ms, base.BaseUid()) == nil) {
			return
		}
		pos := base.BasePos()
		dist := pos.Distance(target)
		if dist < bestDist {
			bestDist = dist
			best = base
			bestAircraft = aircraft
		}
	}
	for _, uid := range ms.Interaction.SelectedShips {
		ship := ms.Arena.Ships[uid]
		if ship == nil {
			continue
		}
		consider(ship, &ship.Aircraft)
	}
	if field := ms.Arena.Airfields[ms.Interaction.SelectedAirfieldUid]; field != nil {
		consider(field, &field.Aircraft)
	}
	return best, bestAircraft
}

// findManualScout 找这个基地还在执行手动侦察、可以改点的飞机。
// 侦察任务已经结束（盘旋完毕或油尽返航）的飞机不再占用改点名额；
// 打击机顶班的侦察机同样算数，否则每次点击都会多派一架。
func findManualScout(ms *state.MissionState, baseUid string) *objUnit.Plane {
	for _, plane := range ms.Arena.Planes {
		if plane.BelongShip != baseUid || !plane.ScoutManual || plane.CurHP <= 0 || plane.MustReturn() {
			continue
		}
		return plane
	}
	return nil
}
