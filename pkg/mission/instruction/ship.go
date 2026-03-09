package instruction

import (
	"fmt"

	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objWeapon "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/grid"
)

// EnableWeapon 启用武器
type EnableWeapon struct {
	shipUid    string
	weaponType objWeapon.WeaponType
	status     InstrStatus
}

// NewEnableWeapon ...
func NewEnableWeapon(shipUid string, weaponType objWeapon.WeaponType) *EnableWeapon {
	return &EnableWeapon{shipUid: shipUid, weaponType: weaponType, status: Ready}
}

var _ Instruction = (*EnableWeapon)(nil)

// Exec ...
func (i *EnableWeapon) Exec(s *state.MissionState) error {
	i.status = Executed
	// 战舰如果不存在（被摧毁），直跳过
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		return nil
	}

	ship.EnableWeapon(i.weaponType)
	return nil
}

// Executed ...
func (i *EnableWeapon) Executed() bool {
	return i.status == Executed
}

// Uid ...
func (i *EnableWeapon) Uid() string {
	return GenInstrUid(NameEnableWeapon, i.shipUid)
}

// String ...
func (i *EnableWeapon) String() string {
	return fmt.Sprintf("Enable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

// DisableWeapon 禁用武器
type DisableWeapon struct {
	shipUid    string
	weaponType objWeapon.WeaponType
	status     InstrStatus
}

// NewDisableWeapon ...
func NewDisableWeapon(shipUid string, weaponType objWeapon.WeaponType) *DisableWeapon {
	return &DisableWeapon{shipUid: shipUid, weaponType: weaponType, status: Ready}
}

var _ Instruction = (*DisableWeapon)(nil)

// Exec ...
func (i *DisableWeapon) Exec(s *state.MissionState) error {
	i.status = Executed
	// 战舰如果不存在（被摧毁），直跳过
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		return nil
	}

	ship.DisableWeapon(i.weaponType)
	return nil
}

// Executed ...
func (i *DisableWeapon) Executed() bool {
	return i.status == Executed
}

// Uid ...
func (i *DisableWeapon) Uid() string {
	return GenInstrUid(NameDisableWeapon, i.shipUid)
}

// String ...
func (i *DisableWeapon) String() string {
	return fmt.Sprintf("Disable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

// ShipMove 移动
type ShipMove struct {
	shipUid   string
	targetPos objPos.MapPos
	status    InstrStatus
}

// NewShipMove ...
func NewShipMove(shipUid string, targetPos objPos.MapPos) *ShipMove {
	return &ShipMove{shipUid: shipUid, targetPos: targetPos, status: Ready}
}

var _ Instruction = (*ShipMove)(nil)

// Exec ...
func (i *ShipMove) Exec(s *state.MissionState) error {
	// 战舰如果不存在（被摧毁），直接修改指令为已完成
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		i.status = Executed
		return nil
	}

	if ship.MoveTo(s.MissionMD.MapCfg, i.targetPos, true) {
		i.status = Executed
	}
	return nil
}

// Executed ...
func (i *ShipMove) Executed() bool {
	return i.status == Executed
}

// Uid ...
func (i *ShipMove) Uid() string {
	return GenInstrUid(NameShipMove, i.shipUid)
}

// String ...
func (i *ShipMove) String() string {
	return fmt.Sprintf("Ship %s move to %s", i.shipUid, i.targetPos.String())
}

// ShipMovePath 按照指定路径移动
type ShipMovePath struct {
	shipUid   string
	curPos    objPos.MapPos
	targetPos objPos.MapPos
	path      []objPos.MapPos
	curIdx    int
	status    InstrStatus
	// 创建指令时战舰的当前速度，用于路径就绪后恢复速度
	initSpeed float64
}

// NewShipMovePath ...
func NewShipMovePath(shipUid string, curPos, targetPos objPos.MapPos, curSpeed float64) *ShipMovePath {
	return &ShipMovePath{shipUid: shipUid, curPos: curPos, targetPos: targetPos, status: Pending, initSpeed: curSpeed}
}

var _ Instruction = (*ShipMovePath)(nil)

// Exec ...
func (i *ShipMovePath) Exec(s *state.MissionState) error {
	// 寻路失败（genPath 异步标记为 Executing），主线程中重置速度后标记完成
	if i.status == Executing {
		if ship, ok := s.Ships[i.shipUid]; ok {
			ship.CurSpeed = 0
		}
		i.status = Executed
		return nil
	}
	if i.status == Executed {
		return nil
	}

	if i.status != Ready {
		if i.status != Preparing {
			i.status = Preparing
			go i.genPath(s)
		}
		// Preparing 状态下，让战舰继续朝目标方向直线移动作为过渡
		if ship, ok := s.Ships[i.shipUid]; ok && ship.CurSpeed > 0 {
			ship.MoveTo(s.MissionMD.MapCfg, i.targetPos, false)
		}
		return nil
	}

	if i.curIdx >= len(i.path) {
		i.status = Executed
		if ship, ok := s.Ships[i.shipUid]; ok {
			ship.CurSpeed = 0
		}
		return nil
	}

	// 战舰如果不存在（被摧毁），直接修改指令为已完成
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		i.status = Executed
		return nil
	}

	// 路径就绪后的首帧处理（initSpeed >= 0 表示尚未处理过）
	if i.initSpeed >= 0 {
		// 恢复创建指令时的速度，确保不因路径切换而归零
		if i.initSpeed > 0 {
			ship.CurSpeed = min(i.initSpeed, ship.MaxSpeed)
		}
		// 标记首帧处理已完成，避免后续帧重复执行
		i.initSpeed = -1
		// 找到路径中离战舰当前位置最近的有效路径点，跳过已经过的点
		// 避免战舰在过渡移动后"回退"到已经过的路径点
		minDist := ship.CurPos.Distance(i.path[0])
		bestIdx := 0
		for idx := 1; idx < len(i.path); idx++ {
			dist := ship.CurPos.Distance(i.path[idx])
			if dist < minDist {
				minDist = dist
				bestIdx = idx
			} else {
				// 路径点距离开始增大，说明已经过了最近点，停止搜索
				break
			}
		}
		i.curIdx = bestIdx
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

// genPath 生成战舰移动的路径
func (i *ShipMovePath) genPath(misState *state.MissionState) {
	points := misState.MissionMD.MapCfg.GenPath(
		grid.Point{i.curPos.MX, i.curPos.MY},
		grid.Point{i.targetPos.MX, i.targetPos.MY},
	)
	// 寻路失败，标记为 Executing 让主线程的 Exec 处理速度重置
	// 不能直接标记 Executed，否则会被 RemoveExecuted 在 Exec 之前清除，导致速度无法重置
	if len(points) < 2 {
		i.status = Executing
		return
	}
	i.path = []objPos.MapPos{i.curPos}
	for _, p := range points[1 : len(points)-1] {
		i.path = append(i.path, objPos.New(p.X, p.Y))
	}
	i.path = append(i.path, i.targetPos)

	// 如果战舰仍然存在，检查路径起始点是否与战舰当前实际位置重合或过近
	// 如果是则跳过起始点，避免 MoveTo 因"已到达"而将速度归零
	if ship, ok := misState.Ships[i.shipUid]; ok {
		if ship.CurPos.Near(i.path[0], 0.6) && len(i.path) > 1 {
			i.path = i.path[1:]
		}
	}

	i.status = Ready
}

// Executed ...
func (i *ShipMovePath) Executed() bool {
	return i.status == Executed
}

// Uid ...
func (i *ShipMovePath) Uid() string {
	return GenInstrUid(NameShipMovePath, i.shipUid)
}

// String ...
func (i *ShipMovePath) String() string {
	return fmt.Sprintf("Ship %s move with path %v", i.shipUid, i.path)
}
