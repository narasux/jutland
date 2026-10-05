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
	// patrolRadius 战斗巡逻绕基地 / 航母盘旋的半径（地图格）。
	patrolRadius = 2.5
	// patrolLookahead 盘旋瞄准点在切向前方的距离（地图格）。
	patrolLookahead = 1.0
	// patrolRadialGain 把盘旋半径误差折算成瞄准点的外向 / 内向偏移比例，
	// 与切向分量合成一条收敛到 patrolRadius 的圆航线。
	patrolRadialGain = 0.8
)

// PlanePatrol 战斗机没有可打目标时的战斗巡逻（CAP）：绕所属基地 / 航母盘旋
// 等下一个目标，而不是直接降落。燃油耗尽或基地消失时结束，由
// updatePlaneAttackOrReturn 换成返航指令。
type PlanePatrol struct {
	planeUid string
	status   InstrStatus
}

// NewPlanePatrol 创建战斗巡逻指令。
func NewPlanePatrol(planeUid string) *PlanePatrol {
	return &PlanePatrol{planeUid: planeUid, status: Ready}
}

var _ Instruction = (*PlanePatrol)(nil)

// Exec 执行战斗巡逻：起飞阶段先飞完滑跑，巡航阶段绕基地盘旋。
func (i *PlanePatrol) Exec(ms *state.MissionState) error {
	plane, ok := ms.Arena.Planes[i.planeUid]
	// 飞机已经没了或被打掉，这条巡逻结束。
	if !ok || plane.CurHP <= 0 {
		i.status = Executed
		return nil
	}
	base, ok := ms.FindAircraftBase(plane.BelongShip)
	if !ok {
		// 母舰已经沉没：交给返航流程按坠毁处理（与 PlaneReturn 的现状一致）。
		plane.ForceReturn = true
		i.status = Executed
		return nil
	}
	mapCfg := ms.Core.MissionMD.MapCfg
	if plane.FlightPhase == objUnit.PlaneFlightPhaseTakingOff {
		plane.UpdateTakeoff(mapCfg, base)
		return nil
	}
	if !plane.IsCruising() || plane.MustReturn() {
		i.status = Executed
		return nil
	}
	i.orbit(plane, mapCfg, base.BasePos())
	return nil
}

// orbit 让飞机绕着基地盘旋。瞄准点由顺时针切向分量与半径误差的径向修正合成：
// 半径偏小时朝外偏、偏大时朝内偏，飞机按自身转向与速度自然收敛到一条圆航线。
func (i *PlanePatrol) orbit(plane *objUnit.Plane, mapCfg *mapcfg.MapCfg, center objPos.MapPos) {
	dx, dy := plane.CurPos.RX-center.RX, plane.CurPos.RY-center.RY
	distance := math.Hypot(dx, dy)
	if distance < 1e-6 {
		// 正好压在圆心上时方位角没有意义，先朝正北飞出去，下一拍自然进入圆周。
		dx, dy, distance = 0, -1, 1
	}
	// 单位径向（圆心 -> 飞机）与顺时针切向（地图坐标 y 向下）。
	radialX, radialY := dx/distance, dy/distance
	tangentX, tangentY := -radialY, radialX
	radialCorrection := patrolRadialGain * (patrolRadius - distance)
	// 机身当前位置当作"敌机位置"传给战斗机追踪逻辑，盘旋时用失速速度慢速绕圈，
	// 全速绕圈会让 CAP 几分钟就把燃油烧光。
	plane.MoveTo(
		mapCfg,
		objPos.NewR(
			plane.CurPos.RX+patrolLookahead*(tangentX+radialCorrection*radialX),
			plane.CurPos.RY+patrolLookahead*(tangentY+radialCorrection*radialY),
		),
		plane.CurPos,
		0,
	)
}

// Executed 返回巡逻指令是否已经执行完成。
func (i *PlanePatrol) Executed() bool { return i.status == Executed }

func (i *PlanePatrol) Uid() string {
	return GenInstrUid(NamePlanePatrol, i.planeUid)
}

func (i *PlanePatrol) String() string {
	return fmt.Sprintf("Plane %s patrol", i.planeUid)
}
