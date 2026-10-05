package human

import (
	"math"
	"sort"

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
	candidates := scoutBasesByPriority(ms, target)
	if len(candidates) == 0 {
		return nil, hasSelectedAirBase(ms)
	}
	// 侦察目标点打个望远镜标记，让玩家知道 Shift + 右键是侦察而不是移动。
	ms.UI.GameMarks[objMark.IDScout] = objMark.NewImg(
		objMark.IDScout, target, textureImg.ScoutTarget, 20,
	)
	out := map[string]instr.Instruction{}
	// 按优先级找第一个真能执行这次指令的基地：起飞点还有空位就派一架新的
	// 手动侦察机；已经占满（或弹射器还在冷却、库存不足）时改现有最近那架的目标点。
	// 排在前面但这一拍什么都做不了的基地（比如弹射器在冷却又没有在飞的侦察机）
	// 会被跳过，交给下一个候选。
	for _, candidate := range candidates {
		if len(candidate.scouts) < manualScoutLimit(candidate.aircraft) {
			if plane := candidate.aircraft.TakeOffManualScout(candidate.base); plane != nil {
				plane.ScoutManual = true
				ms.Arena.PutPlane(plane)
				order := instr.NewPlaneScout(plane.Uid, target, true)
				out[order.Uid()] = order
				return out, true
			}
		}
		if existing := nearestManualScout(candidate.scouts, target); existing != nil {
			order := instr.NewPlaneScout(existing.Uid, target, true)
			out[order.Uid()] = order
			return out, true
		}
	}
	return out, true
}

// manualScoutLimit 同一基地同时可派出的手动侦察机数量：按起飞点（弹射器）数量，
// 没有甲板信息时退化为 1 架。
func manualScoutLimit(aircraft *objUnit.ShipAircraft) int {
	if count := aircraft.TakeoffPointCount(); count > 0 {
		return count
	}
	return 1
}

// scoutCandidate 是一个能响应侦察指令的选中基地，带上它的优先级与到点击点的距离。
type scoutCandidate struct {
	base     objUnit.AircraftBase
	aircraft *objUnit.ShipAircraft
	scouts   []*objUnit.Plane
	priority int
	distance float64
}

// 基地响应侦察指令的优先级，数字越小越先派。
const (
	// scoutPriorityNeverLaunched 还没派过手动侦察机：先让它出第一架。
	// 选中一个编队连续点击时，这样能一艘接一艘轮着派，而不是死盯着最近那艘。
	scoutPriorityNeverLaunched = iota
	// scoutPriorityHasSlot 派过了，但起飞点还有空位，可以补第二架。
	scoutPriorityHasSlot
	// scoutPriorityRetarget 名额已满或库存不足，只能改现有侦察机的目标点。
	scoutPriorityRetarget
)

// scoutBasesByPriority 收集选中的航空基地，按（优先级，到点击点距离）排序。
// 同一档里取最近的，所以编队里想先派哪艘就往哪边点。
func scoutBasesByPriority(ms *state.MissionState, target objPos.MapPos) []scoutCandidate {
	candidates := make([]scoutCandidate, 0, len(ms.Interaction.SelectedShips)+1)
	consider := func(base objUnit.AircraftBase, aircraft *objUnit.ShipAircraft) {
		if aircraft == nil {
			return
		}
		scouts := manualScouts(ms, base.BaseUid())
		priority := scoutPriorityRetarget
		switch {
		case aircraft.CanManualScout() && len(scouts) == 0:
			priority = scoutPriorityNeverLaunched
		case aircraft.CanManualScout() && len(scouts) < manualScoutLimit(aircraft):
			priority = scoutPriorityHasSlot
		case len(scouts) > 0:
			priority = scoutPriorityRetarget
		default:
			// 没库存、天上也没有手动侦察机：这个基地帮不上忙。
			return
		}
		pos := base.BasePos()
		candidates = append(candidates, scoutCandidate{
			base: base, aircraft: aircraft, scouts: scouts,
			priority: priority, distance: pos.Distance(target),
		})
	}
	for _, uid := range ms.Interaction.SelectedShips {
		if ship := ms.Arena.Ships[uid]; ship != nil {
			consider(ship, &ship.Aircraft)
		}
	}
	if field := ms.Arena.Airfields[ms.Interaction.SelectedAirfieldUid]; field != nil {
		consider(field, &field.Aircraft)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].distance < candidates[j].distance
	})
	return candidates
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

// manualScouts 找出这个基地还在执行手动侦察、可以改点的飞机。
// 侦察任务已经结束（盘旋完毕或油尽返航）的飞机不再占用名额；
// 打击机顶班的侦察机同样算数，否则每次点击都会多派一架。
func manualScouts(ms *state.MissionState, baseUid string) []*objUnit.Plane {
	scouts := []*objUnit.Plane{}
	for _, plane := range ms.Arena.Planes {
		if plane.BelongShip != baseUid || !plane.ScoutManual || plane.CurHP <= 0 || plane.MustReturn() {
			continue
		}
		scouts = append(scouts, plane)
	}
	return scouts
}

// nearestManualScout 取离改点目标最近的一架手动侦察机。
// 同一基地允许多架手动侦察机时，改点的应该是离点击点最近的那架。
func nearestManualScout(scouts []*objUnit.Plane, target objPos.MapPos) *objUnit.Plane {
	var best *objUnit.Plane
	bestDist := math.MaxFloat64
	for _, plane := range scouts {
		dist := plane.CurPos.Distance(target)
		if best == nil || dist < bestDist || (dist == bestDist && plane.Uid < best.Uid) {
			bestDist, best = dist, plane
		}
	}
	return best
}
