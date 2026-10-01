package instruction

import (
	"fmt"
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// PlaneScout 侦察机飞向一个海面点，并按可见情况跟踪或巡逻。
type PlaneScout struct {
	planeUid        string
	point           objPos.MapPos
	manual          bool
	status          InstrStatus
	loiterUntil     int64
	lastPursuitDist float64
}

// NewPlaneScout 创建侦察指令。manual 为真时再次指定同一架飞机会改飞向。
func NewPlaneScout(planeUid string, point objPos.MapPos, manual bool) *PlaneScout {
	return &PlaneScout{
		planeUid:        planeUid,
		point:           point,
		manual:          manual,
		status:          Ready,
		lastPursuitDist: math.MaxFloat64,
	}
}

// Retarget 玩家再次指定侦察点时改飞向，并重新计算盘旋。
func (i *PlaneScout) Retarget(point objPos.MapPos) {
	i.point = point
	i.loiterUntil = 0
	i.status = Ready
}

func (i *PlaneScout) Exec(ms *state.MissionState) error {
	plane, ok := ms.Arena.Planes[i.planeUid]
	// 飞机已经没了或被打掉，这条侦察结束。
	if !ok || plane.CurHP <= 0 {
		i.status = Executed
		return nil
	}
	base, _ := ms.FindAircraftBase(plane.BelongShip)
	mapCfg := ms.Core.MissionMD.MapCfg
	// 还在滑跑或爬升时先把起飞飞完，不改航向。
	if plane.FlightPhase == objUnit.PlaneFlightPhaseTakingOff {
		plane.UpdateTakeoff(mapCfg, base)
		return nil
	}
	// 油尽后交给现有返航，侦察指令本身结束。
	if plane.MustReturn() {
		i.status = Executed
		return nil
	}
	vision := ms.Player.Visions[plane.BelongPlayer]
	// 敌机进入半个侦察视距并且还在靠近，停止侦察并返航。
	if i.pursued(ms, plane) {
		plane.ForceReturn = true
		i.status = Executed
		return nil
	}
	// 看得见敌舰就改去跟踪舰队，不再飞原来的点。
	// 已经贴到防空圈外沿就停住，避免再靠近挨打。
	if vision != nil {
		if move, hold := trackFleet(ms, plane, vision); move || hold {
			if !hold {
				plane.MoveTo(mapCfg, fleetHoldPoint(ms, plane, vision), plane.CurPos, 0)
			}
			i.loiterUntil = 0
			return nil
		}
	}
	// 还没到目标点就继续飞。到达后的盘旋从这一拍开始计 30 秒任务时间。
	if plane.CurPos.Distance(i.point) > 1.5 {
		plane.MoveTo(mapCfg, i.point, plane.CurPos, 0)
		return nil
	}
	if i.loiterUntil == 0 {
		i.loiterUntil = ms.Core.SimTick + objUnit.ReloadTicks(30)
	}
	if ms.Core.SimTick < i.loiterUntil {
		return nil
	}
	// 自动侦察盘旋结束后换下一段：先揭未探索，没有就去最远的已探索海面。
	// 手动侦察不换点，盘旋完直接返航。
	if !i.manual && vision != nil {
		if next, ok := vision.BestUncover(plane.CurPos, plane.SightRange); ok {
			i.point = next
			i.loiterUntil = 0
			return nil
		}
		if next, ok := vision.FarthestExplored(plane.CurPos, otherScoutPositions(ms, plane)); ok {
			i.point = next
			i.loiterUntil = 0
			return nil
		}
	}
	plane.ForceReturn = true
	i.status = Executed
	return nil
}

func (i *PlaneScout) pursued(ms *state.MissionState, plane *objUnit.Plane) bool {
	nearest := math.MaxFloat64
	for _, other := range ms.Arena.Planes {
		if other.Uid == plane.Uid || other.BelongPlayer == plane.BelongPlayer || other.CurHP <= 0 {
			continue
		}
		if dist := plane.CurPos.Distance(other.CurPos); dist < nearest {
			nearest = dist
		}
	}
	closing := nearest < i.lastPursuitDist
	i.lastPursuitDist = nearest
	return nearest <= objUnit.SightRangeScout/2 && closing
}

func (i *PlaneScout) Executed() bool { return i.status == Executed }

// PlaneUid 这架侦察机的 uid。
func (i *PlaneScout) PlaneUid() string { return i.planeUid }

// Manual 是否为玩家手动派出的侦察。
func (i *PlaneScout) Manual() bool { return i.manual }

func (i *PlaneScout) Uid() string {
	return GenInstrUid(NamePlaneScout, i.planeUid)
}

func (i *PlaneScout) String() string {
	return fmt.Sprintf("Plane %s scout %s manual=%v", i.planeUid, i.point.String(), i.manual)
}

var _ Instruction = (*PlaneScout)(nil)

func trackFleet(ms *state.MissionState, plane *objUnit.Plane, vision *state.FactionVision) (move, hold bool) {
	// 当前可见的敌舰合成一群。没有就回到飞向侦察点的逻辑。
	ships := visibleEnemyShips(ms, plane, vision)
	if len(ships) == 0 {
		return false, false
	}
	// 停在这群舰最远防空射程再加 2 格之外，能看进舰队又打不着侦察机。
	limit := maxAntiAir(ships) + 2
	nearest := math.MaxFloat64
	for _, ship := range ships {
		if dist := plane.CurPos.Distance(ship.CurPos); dist < nearest {
			nearest = dist
		}
	}
	if nearest <= limit {
		return true, true
	}
	return true, false
}

func fleetHoldPoint(ms *state.MissionState, plane *objUnit.Plane, vision *state.FactionVision) objPos.MapPos {
	ships := visibleEnemyShips(ms, plane, vision)
	if len(ships) == 0 {
		return plane.CurPos
	}
	sumX, sumY := 0.0, 0.0
	for _, ship := range ships {
		sumX += ship.CurPos.RX
		sumY += ship.CurPos.RY
	}
	return objPos.NewR(sumX/float64(len(ships)), sumY/float64(len(ships)))
}

func visibleEnemyShips(ms *state.MissionState, plane *objUnit.Plane, vision *state.FactionVision) []*objUnit.BattleShip {
	ships := make([]*objUnit.BattleShip, 0)
	for _, ship := range ms.Arena.Ships {
		if ship.BelongPlayer == plane.BelongPlayer || ship.CurHP <= 0 {
			continue
		}
		if vision.VisibleAt(ship.CurPos.MX, ship.CurPos.MY) {
			ships = append(ships, ship)
		}
	}
	return ships
}

func maxAntiAir(ships []*objUnit.BattleShip) float64 {
	best := 0.0
	for _, ship := range ships {
		if ship.Weapon.MaxToPlaneRange > best {
			best = ship.Weapon.MaxToPlaneRange
		}
	}
	return best
}

func otherScoutPositions(ms *state.MissionState, self *objUnit.Plane) []objPos.MapPos {
	positions := make([]objPos.MapPos, 0)
	for _, plane := range ms.Arena.Planes {
		if plane.Uid == self.Uid || plane.Type != objUnit.PlaneTypeScout || plane.CurHP <= 0 {
			continue
		}
		positions = append(positions, plane.CurPos)
	}
	return positions
}
