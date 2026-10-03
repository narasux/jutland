package instruction

import (
	"fmt"
	"math"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

const (
	// scoutThreatRange 敌机靠近到这个距离就转入规避机动，取侦察视距的一半。
	scoutThreatRange = objUnit.SightRangeScout / 2
	// scoutSafeRange 拉开到这个距离才恢复侦察航线，取四分之三视距，
	// 比战斗机视距更远，避免刚回到航线又被咬上。
	scoutSafeRange = objUnit.SightRangeScout * 3 / 4
	// scoutEvadeStep 规避目标点取在背离敌机方向的这个距离上。
	scoutEvadeStep = 6.0
	// scoutArriveRadius 是判定「已到达侦察点」的距离（地图格）。
	scoutArriveRadius = 1.5
	// scoutLoiterRadius 是到达侦察点后的盘旋半径（地图格）。必须明显小于
	// scoutArriveRadius，否则飞机绕到远端会被重新判成「还没到达」而直飞圆心。
	scoutLoiterRadius = 1.0
	// scoutLoiterLookahead 是盘旋瞄准点在切向前方的距离（地图格）。
	scoutLoiterLookahead = 1.0
	// scoutLoiterRadialGain 把盘旋半径误差折算成瞄准点的外向/内向偏移比例，
	// 与切向分量合成一条收敛到 scoutLoiterRadius 的盘旋航线。
	scoutLoiterRadialGain = 0.8
)

// PlaneScout 侦察机飞向一个海面点，并按可见情况跟踪或巡逻。
type PlaneScout struct {
	planeUid        string
	point           objPos.MapPos
	manual          bool
	status          InstrStatus
	loiterUntil     int64
	lastPursuitDist float64
	// evading 正在规避逼近的敌机，期间不飞向侦察点也不返航。
	evading bool
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
	i.evading = false
	i.lastPursuitDist = math.MaxFloat64
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
	// 敌机逼近时先做规避机动继续侦察，不放弃任务直接返航：
	// 提前返航会把侦察机送进慢速进近，反而更容易在母舰附近被咬住击落。
	if i.evade(ms, plane) {
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
	if plane.CurPos.Distance(i.point) > scoutArriveRadius {
		plane.MoveTo(mapCfg, i.point, plane.CurPos, 0)
		return nil
	}
	if i.loiterUntil == 0 {
		i.loiterUntil = ms.Core.SimTick + objUnit.ReloadTicks(30)
	}
	if ms.Core.SimTick < i.loiterUntil {
		// 飞机到了点位也不能停下，绕着侦察点盘旋，保持视野扫过整片区域。
		i.orbit(plane, mapCfg)
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

// evade 遇到敌机时朝背离方向规避，而不是中止侦察返航。
// 返回 true 表示这一拍用规避机动代替了飞向侦察点。
func (i *PlaneScout) evade(ms *state.MissionState, plane *objUnit.Plane) bool {
	threat, dist := nearestEnemyPlane(ms, plane)
	if threat == nil {
		i.evading, i.lastPursuitDist = false, math.MaxFloat64
		return false
	}
	if !i.evading {
		// 还没被咬上时，只有敌机靠得够近而且在接近才动手，避免对着远去的敌机乱转向。
		closing := dist < i.lastPursuitDist
		i.lastPursuitDist = dist
		if dist > scoutThreatRange || !closing {
			return false
		}
		i.evading = true
	} else if dist > scoutSafeRange {
		// 已经拉开到安全距离，交还给下面的侦察航线。
		i.evading, i.lastPursuitDist = false, math.MaxFloat64
		return false
	}
	// 背离最近敌机设一个规避点，每拍按敌机新位置重算，飞出一条连续转弯的脱离航线。
	bearing := threat.CurPos.Angle(plane.CurPos) * math.Pi / 180
	plane.MoveTo(
		ms.Core.MissionMD.MapCfg,
		objPos.NewR(
			plane.CurPos.RX+math.Sin(bearing)*scoutEvadeStep,
			plane.CurPos.RY-math.Cos(bearing)*scoutEvadeStep,
		),
		plane.CurPos,
		0,
	)
	return true
}

// orbit 让飞机绕着侦察点盘旋。瞄准点由顺时针切向分量与半径误差的径向修正合成：
// 半径偏小时朝外偏、偏大时朝内偏，飞机按自身转向与速度自然收敛到一条圆航线，
// 而不是在圆心原地悬停。
func (i *PlaneScout) orbit(plane *objUnit.Plane, mapCfg *mapcfg.MapCfg) {
	dx, dy := plane.CurPos.RX-i.point.RX, plane.CurPos.RY-i.point.RY
	distance := math.Hypot(dx, dy)
	if distance < 1e-6 {
		// 正好压在圆心上时方位角没有意义，先朝正北飞出去，下一拍自然进入圆周。
		dx, dy, distance = 0, -1, 1
	}
	// 单位径向（圆心 -> 飞机）与顺时针切向（地图坐标 y 向下）。
	radialX, radialY := dx/distance, dy/distance
	tangentX, tangentY := -radialY, radialX
	radialCorrection := scoutLoiterRadialGain * (scoutLoiterRadius - distance)
	// 盘旋只是原地绕圈，不额外消耗任务航程：侦察续航沿用到达前的预算。
	// 否则每段 30 秒盘旋要飞掉几十格航程，侦察机会提前返航、开图范围缩水。
	remainRange := plane.RemainRange
	plane.MoveTo(
		mapCfg,
		objPos.NewR(
			plane.CurPos.RX+scoutLoiterLookahead*(tangentX+radialCorrection*radialX),
			plane.CurPos.RY+scoutLoiterLookahead*(tangentY+radialCorrection*radialY),
		),
		plane.CurPos,
		0,
	)
	plane.RemainRange = remainRange
}

// nearestEnemyPlane 返回离这架飞机最近的敌机及距离，没有敌机时返回 nil。
func nearestEnemyPlane(ms *state.MissionState, plane *objUnit.Plane) (*objUnit.Plane, float64) {
	var nearest *objUnit.Plane
	dist := math.MaxFloat64
	for _, other := range ms.Arena.Planes {
		if other.Uid == plane.Uid || other.BelongPlayer == plane.BelongPlayer || other.CurHP <= 0 {
			continue
		}
		if d := plane.CurPos.Distance(other.CurPos); d < dist {
			nearest, dist = other, d
		}
	}
	return nearest, dist
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
