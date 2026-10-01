package human

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/narasux/jutland/pkg/mission/action"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func (h *HumanInputHandler) handleScout(ms *state.MissionState) map[string]instr.Instruction {
	// 没按 Shift，或点在界面上，就交给原来的右键移动。
	if !ebiten.IsKeyPressed(ebiten.KeyShift) || ms.UI.UIConsumesCursor {
		return nil
	}
	pos := action.DetectMouseButtonClickOnMap(ms, ebiten.MouseButtonRight)
	if pos == nil {
		return nil
	}
	// 在选中的航母和机场里，挑离点击点最近、而且还有侦察机的那个。
	base, aircraft := nearestSelectedScoutBase(ms, *pos)
	if aircraft == nil {
		return nil
	}
	out := map[string]instr.Instruction{}
	// 这艘基地已经有一架手动侦察在飞，就只改它的目标点，不再起飞第二架。
	if existing := findManualScout(ms, base.BaseUid()); existing != nil {
		order := instr.NewPlaneScout(existing.Uid, *pos, true)
		out[order.Uid()] = order
		return out
	}
	if aircraft.ScoutStock() < 1 {
		return nil
	}
	// 新起飞的这一架标成手动，航程在起飞时翻倍，然后飞向点击点。
	plane := aircraft.TakeOffScout(base)
	if plane == nil {
		return nil
	}
	plane.ScoutManual = true
	ms.Arena.PutPlane(plane)
	order := instr.NewPlaneScout(plane.Uid, *pos, true)
	out[order.Uid()] = order
	return out
}

func nearestSelectedScoutBase(ms *state.MissionState, target objPos.MapPos) (objUnit.AircraftBase, *objUnit.ShipAircraft) {
	var best objUnit.AircraftBase
	var bestAircraft *objUnit.ShipAircraft
	bestDist := math.MaxFloat64
	consider := func(base objUnit.AircraftBase, aircraft *objUnit.ShipAircraft) {
		if aircraft == nil || (aircraft.ScoutStock() < 1 && findManualScout(ms, base.BaseUid()) == nil) {
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

func findManualScout(ms *state.MissionState, baseUid string) *objUnit.Plane {
	for _, plane := range ms.Arena.Planes {
		if plane.BelongShip == baseUid && plane.Type == objUnit.PlaneTypeScout && plane.ScoutManual && plane.CurHP > 0 {
			return plane
		}
	}
	return nil
}
