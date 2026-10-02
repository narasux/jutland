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
	ship, ok := s.Arena.Ships[i.shipUid]
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
	ship, ok := s.Arena.Ships[i.shipUid]
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

// EnableAircraft 允许指定战舰继续起飞舰载机。
type EnableAircraft struct {
	shipUid string
	status  InstrStatus
}

// NewEnableAircraft 创建允许舰载机起飞的即时指令。
func NewEnableAircraft(shipUid string) *EnableAircraft {
	return &EnableAircraft{shipUid: shipUid, status: Ready}
}

var _ Instruction = (*EnableAircraft)(nil)

// Exec 执行允许舰载机起飞操作。
func (i *EnableAircraft) Exec(s *state.MissionState) error {
	i.status = Executed
	if ship := s.Arena.Ships[i.shipUid]; ship != nil {
		ship.Aircraft.Disable = false
	}
	return nil
}

// Executed 返回指令是否已经执行。
func (i *EnableAircraft) Executed() bool { return i.status == Executed }

// Uid 返回该舰唯一的舰载机启用指令标识。
func (i *EnableAircraft) Uid() string { return GenInstrUid(NameEnableAircraft, i.shipUid) }

// String 返回便于日志记录的指令说明。
func (i *EnableAircraft) String() string {
	return fmt.Sprintf("Enable ship %s aircraft", i.shipUid)
}

// DisableAircraft 禁止指定战舰继续起飞舰载机。
type DisableAircraft struct {
	shipUid string
	status  InstrStatus
}

// NewDisableAircraft 创建禁止舰载机起飞的即时指令。
func NewDisableAircraft(shipUid string) *DisableAircraft {
	return &DisableAircraft{shipUid: shipUid, status: Ready}
}

var _ Instruction = (*DisableAircraft)(nil)

// Exec 执行禁止舰载机起飞操作。
func (i *DisableAircraft) Exec(s *state.MissionState) error {
	i.status = Executed
	if ship := s.Arena.Ships[i.shipUid]; ship != nil {
		ship.Aircraft.Disable = true
	}
	return nil
}

// Executed 返回指令是否已经执行。
func (i *DisableAircraft) Executed() bool { return i.status == Executed }

// Uid 返回该舰唯一的舰载机禁用指令标识。
func (i *DisableAircraft) Uid() string { return GenInstrUid(NameDisableAircraft, i.shipUid) }

// String 返回便于日志记录的指令说明。
func (i *DisableAircraft) String() string {
	return fmt.Sprintf("Disable ship %s aircraft", i.shipUid)
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
	ship, ok := s.Arena.Ships[i.shipUid]
	if !ok {
		i.status = Executed
		return nil
	}

	// 直线移动指令（方向键/散开/可上陆的特殊船）没有重寻路的概念：
	// 到达即完成；被地形拦停也视为完成，避免指令永久滞留。
	arrive, blocked := ship.MoveTo(s.Core.MissionMD.MapCfg, i.targetPos, true)
	if arrive || blocked {
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

// 撞岸受阻后的重新规划节奏：每 blockedRepathInterval 拍重寻一次，
// 连续 blockedRepathMaxTries 次仍然受阻就放弃本次机动。
const (
	blockedRepathInterval = 20
	blockedRepathMaxTries = 3
)

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
	result    chan []grid.Point
	// 撞岸受阻计数（拍）与已重试次数，用于限频重寻路和最终放弃
	blockedTicks int
	repathTries  int
}

// NewShipMovePath ...
func NewShipMovePath(shipUid string, curPos, targetPos objPos.MapPos, curSpeed float64) *ShipMovePath {
	return &ShipMovePath{shipUid: shipUid, curPos: curPos, targetPos: targetPos, status: Pending, initSpeed: curSpeed}
}

var _ Instruction = (*ShipMovePath)(nil)

// Exec ...
func (i *ShipMovePath) Exec(s *state.MissionState) error {
	i.consumePathResult()
	// 寻路失败（异步标记为 Executing），主线程中重置速度后标记完成
	if i.status == Executing {
		if ship, ok := s.Arena.Ships[i.shipUid]; ok {
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
			i.result = make(chan []grid.Point, 1)
			pathSearches.submit(pathSearchJob{
				grid:  s.Core.MissionMD.MapCfg.PreparedGrid(),
				start: grid.Point{i.curPos.MX, i.curPos.MY},
				goal:  grid.Point{i.targetPos.MX, i.targetPos.MY},
				reply: i.result,
			})
		}
		// Preparing 状态下，让战舰继续朝目标方向直线移动作为过渡
		if ship, ok := s.Arena.Ships[i.shipUid]; ok && ship.CurSpeed > 0 {
			ship.MoveTo(s.Core.MissionMD.MapCfg, i.targetPos, false)
		}
		return nil
	}

	if i.curIdx >= len(i.path) {
		i.status = Executed
		if ship, ok := s.Arena.Ships[i.shipUid]; ok {
			ship.CurSpeed = 0
		}
		return nil
	}

	// 战舰如果不存在（被摧毁），直接修改指令为已完成
	ship, ok := s.Arena.Ships[i.shipUid]
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

	arrive, blocked := ship.MoveTo(
		s.Core.MissionMD.MapCfg,
		i.path[i.curIdx],
		i.curIdx == len(i.path)-1,
	)
	if blocked {
		i.blockedTicks++
		// 撞岸被地形拦停：不推进航点，限频地按当前位置重新规划；
		// 反复受阻则放弃本次机动。
		if i.blockedTicks >= blockedRepathInterval {
			i.blockedTicks = 0
			i.repathTries++
			if i.repathTries > blockedRepathMaxTries {
				ship.CurSpeed = 0
				i.status = Executed
				return nil
			}
			i.repathFrom(s, ship.CurPos)
		}
		return nil
	}
	if arrive {
		i.curIdx++
	}
	return nil
}

// repathFrom 撞岸恢复：以战舰当前位置为起点重新提交寻路。
// 起点吸附（GenPath.snapToSea）保证中心点已压进海岸格的搁浅舰也能退出。
func (i *ShipMovePath) repathFrom(s *state.MissionState, curPos objPos.MapPos) {
	i.curPos = curPos
	i.status = Preparing
	i.result = make(chan []grid.Point, 1)
	pathSearches.submit(pathSearchJob{
		grid:  s.Core.MissionMD.MapCfg.PreparedGrid(),
		start: grid.Point{curPos.MX, curPos.MY},
		goal:  grid.Point{i.targetPos.MX, i.targetPos.MY},
		reply: i.result,
	})
}

// consumePathResult 在主线程取回已经算完的航线。
func (i *ShipMovePath) consumePathResult() {
	if i.result == nil {
		return
	}
	select {
	case points := <-i.result:
		i.applyPoints(points)
		i.result = nil
	default:
	}
}

// genPath 生成战舰移动的路径。测试直接调用；对局里由寻路线程算完后在主线程套用。
func (i *ShipMovePath) genPath(misState *state.MissionState) {
	i.applyPoints(misState.Core.MissionMD.MapCfg.GenPath(
		grid.Point{i.curPos.MX, i.curPos.MY},
		grid.Point{i.targetPos.MX, i.targetPos.MY},
	))
}

func (i *ShipMovePath) applyPoints(points []grid.Point) {
	// 起点与终点在同一格（或吸附到同一格）：直接对精确目标点做最后一段直线逼近。
	if len(points) == 1 {
		i.path = []objPos.MapPos{i.targetPos}
		i.curIdx = 0
		i.status = Ready
		return
	}
	// 寻路失败，标记为 Executing 让主线程的 Exec 处理速度重置
	// 不能直接标记 Executed，否则会被 RemoveExecuted 在 Exec 之前清除，导致速度无法重置
	if len(points) < 2 {
		i.status = Executing
		return
	}
	// 寻路期间战舰会继续向新目标转向、移动；命令下达时的位置已经在身后，
	// 不能再作为航点，否则大地图寻路较慢时战舰会折返并在近距离原地掉头。
	// 中间航点取路径格中心而不是格角：舰体从偏离折线的位置"追赶"航点时，
	// 同格内的直线段不会踏进相邻的海岸/陆地格（格角航点会切角）。
	i.path = make([]objPos.MapPos, 0, len(points)-1)
	for _, p := range points[1 : len(points)-1] {
		i.path = append(i.path, objPos.NewR(float64(p.X)+0.5, float64(p.Y)+0.5))
	}
	i.path = append(i.path, i.targetPos)
	// 重寻路（撞岸恢复）会套用新路径，航点游标必须归零；
	// 首次套用的"跳过已经过的点"由 Exec 的 initSpeed 首帧逻辑负责。
	i.curIdx = 0

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
