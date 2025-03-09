package instruction

import (
	"fmt"

	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/grid"
)

// PlaneMove 移动
type PlaneMove struct {
	planeUid  string
	targetPos obj.MapPos
	status    InstrStatus
}

// NewPlaneMove ...
func NewPlaneMove(planeUid string, targetPos obj.MapPos) *PlaneMove {
	return &PlaneMove{planeUid: planeUid, targetPos: targetPos, status: Ready}
}

var _ Instruction = (*PlaneMove)(nil)

func (i *PlaneMove) Exec(s *state.MissionState) error {
	// 战舰如果不存在（被摧毁），直接修改指令为已完成
	plane, ok := s.Planes[i.planeUid]
	if !ok {
		i.status = Executed
		return nil
	}

	if plane.MoveTo(s.MissionMD.MapCfg, i.targetPos) {
		i.status = Executed
	}
	return nil
}

func (i *PlaneMove) Executed() bool {
	return i.status == Executed
}

func (i *PlaneMove) Uid() string {
	return GenInstrUid(NamePlaneMove, i.planeUid)
}

func (i *PlaneMove) String() string {
	return fmt.Sprintf("Plane %s move to %s", i.planeUid, i.targetPos.String())
}

// PlaneMovePath 按照指定路径移动
type PlaneMovePath struct {
	planeUid  string
	curPos    obj.MapPos
	targetPos obj.MapPos
	path      []obj.MapPos
	curIdx    int
	status    InstrStatus
}

// NewPlaneMovePath ...
func NewPlaneMovePath(planeUid string, curPos, targetPos obj.MapPos) *PlaneMovePath {
	return &PlaneMovePath{planeUid: planeUid, curPos: curPos, targetPos: targetPos, status: Pending}
}

var _ Instruction = (*PlaneMovePath)(nil)

func (i *PlaneMovePath) Exec(s *state.MissionState) error {
	if i.status != Ready {
		if i.status != Preparing {
			i.status = Preparing
			go i.genPath(s)
		}
		return nil
	}

	if i.curIdx >= len(i.path) {
		i.status = Executed
		return nil
	}

	// 战舰如果不存在（被摧毁），直接修改指令为已完成
	ship, ok := s.Ships[i.planeUid]
	if !ok {
		i.status = Executed
		return nil
	}
	if ship.MoveTo(
		s.MissionMD.MapCfg,
		i.path[i.curIdx],
		i.curIdx == len(i.path)-1,
	) {
		i.curIdx++
	}
	return nil
}

func (i *PlaneMovePath) genPath(misState *state.MissionState) {
	points := misState.MissionMD.MapCfg.GenPath(
		grid.Point{i.curPos.MX, i.curPos.MY},
		grid.Point{i.targetPos.MX, i.targetPos.MY},
	)
	// 无需执行的路径（如寻路失败，直接算指令执行成功）
	if len(points) < 2 {
		i.status = Executed
		return
	}
	i.path = []obj.MapPos{i.curPos}
	for _, p := range points[1 : len(points)-1] {
		i.path = append(i.path, obj.NewMapPos(p.X, p.Y))
	}
	i.path = append(i.path, i.targetPos)
	i.status = Ready
}

func (i *PlaneMovePath) Executed() bool {
	return i.status == Executed
}

func (i *PlaneMovePath) Uid() string {
	return GenInstrUid(NamePlaneMovePath, i.planeUid)
}

func (i *PlaneMovePath) String() string {
	return fmt.Sprintf("Plane %s move with path %v", i.planeUid, i.path)
}
