package instruction

import (
	"fmt"
	"math"

	"github.com/narasux/jutland/pkg/common/constants"
	"github.com/narasux/jutland/pkg/i18n"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objWeapon "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
	"github.com/narasux/jutland/pkg/utils/colorx"
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

// 撞岸受阻后的重新规划节奏：连续受阻 blockedRepathInterval 拍，
// 且舰体沿岸滑动超过 blockedRepathMinDrift 格时重寻一次，让航线跟上实际位置。
// 这里不再按受阻拍数放弃机动：掉头期间舰体一直在原地转向，转向本身就是进展，
// 旧实现累计受阻 80 拍就结束指令，而战列舰掉头需要 400 拍以上，玩家必须反复下单。
const (
	blockedRepathInterval = 20
	blockedRepathMinDrift = 0.5
	// blockedRepathMaxStalled 原地顶住时允许的连续重寻次数。
	// 重寻会以舰船当前格为起点，能救回“航线起点还是旧位置、第一段已经走不通”的情况；
	// 连续重寻仍然没有位移就停手，避免在真正走不通的位置反复占用寻路线程。
	blockedRepathMaxStalled = 2
	// blockedNoProgressMaxTicks 连续“既无位移也无转向”的拍数上限。
	// 撞岸掉头不会触发；只有目标真的不可达（例如点在陆地上）才会走到这里。
	blockedNoProgressMaxTicks = 240
	// progressEpsilonDistance / progressEpsilonRotation 判定“有进展”的最小位移（格）与转角（度）。
	progressEpsilonDistance = 0.05
	progressEpsilonRotation = 0.5
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
	// targetPrepared 标记目标点是否已做过“可航行吸附”，只做一次
	targetPrepared bool
	// escapePos 搁浅舰退出搁浅用的第一航点（吸附后的水面格心），非搁浅时为 nil
	escapePos *objPos.MapPos
	// failed 标记本次机动失败（寻路不可达或长时间无进展），供 UI 提示
	failed bool
	// pathGeneration / syncedGeneration 用于在每次套用新路径后重新同步航点游标
	pathGeneration   int
	syncedGeneration int
	// 连续受阻计数（成功移动一帧即清零）与上次重寻时的位置
	blockedTicks int
	repathPos    objPos.MapPos
	// 原地顶住时的连续重寻次数（有位移即清零）
	stalledRepaths int
	// 无进展计数与上一次“有进展”时的位置 / 航向
	noProgressTicks     int
	progressInitialized bool
	lastProgressPos     objPos.MapPos
	lastProgressRot     float64
}

// NewShipMovePath ...
func NewShipMovePath(shipUid string, curPos, targetPos objPos.MapPos, curSpeed float64) *ShipMovePath {
	return &ShipMovePath{
		shipUid:   shipUid,
		curPos:    curPos,
		targetPos: targetPos,
		repathPos: curPos,
		status:    Pending,
		initSpeed: curSpeed,
	}
}

var _ Instruction = (*ShipMovePath)(nil)

// Failed 本次机动是否失败（目标不可达或长时间无进展）。
func (i *ShipMovePath) Failed() bool {
	return i.failed
}

// prepareTarget 把落在陆格上的目标点吸附到与当前位置同一片水域的可航行格心。
// 只做一次：末段航点、到达判定与重寻路都以吸附后的目标为准，避免
// “最后一段主动往陆格开 → 撞停 → 放弃”，以及“终点在封闭水域 → 寻路直接失败”。
func (i *ShipMovePath) prepareTarget(mapCfg *mapcfg.MapCfg) {
	if i.targetPrepared {
		return
	}
	i.targetPrepared = true
	if mapCfg.Map.IsSea(i.targetPos.MX, i.targetPos.MY) {
		return
	}
	_, goal := mapCfg.SnapPathEndpoints(
		grid.Point{X: i.curPos.MX, Y: i.curPos.MY},
		grid.Point{X: i.targetPos.MX, Y: i.targetPos.MY},
	)
	// 只有吸附真的换了一格才改写目标：既保留玩家精确点击的位置，
	// 也避免没有地图数据的手工测试夹具被无条件改写成格心。
	if goal.X != i.targetPos.MX || goal.Y != i.targetPos.MY {
		i.targetPos = objPos.NewR(float64(goal.X)+0.5, float64(goal.Y)+0.5)
	}
}

// submitSearch 在主线程完成起终点吸附后提交异步寻路。
// 吸附必须复用 MapCfg.SnapPathEndpoints（与 GenPath 同一套逻辑），
// 否则中心点压岸的搁浅舰会被 Grid.Search 直接判为无效起终点，指令静默结束。
func (i *ShipMovePath) submitSearch(s *state.MissionState, from objPos.MapPos) {
	mapCfg := s.Core.MissionMD.MapCfg
	i.prepareTarget(mapCfg)
	start, goal := mapCfg.SnapPathEndpoints(
		grid.Point{X: from.MX, Y: from.MY},
		grid.Point{X: i.targetPos.MX, Y: i.targetPos.MY},
	)
	// 中心点压在陆格上的搁浅舰：把吸附后的水面格插为第一航点，
	// 让它就近退出搁浅（MoveTo 会限制搁浅舰只能朝该航点靠近），
	// 而不是朝着最终航向在陆地上横向漂移。
	i.escapePos = nil
	if mapCfg.Map.IsLand(from.MX, from.MY) && (start.X != from.MX || start.Y != from.MY) {
		escapePos := objPos.NewR(float64(start.X)+0.5, float64(start.Y)+0.5)
		i.escapePos = &escapePos
	}
	i.status = Preparing
	i.result = make(chan []grid.Point, 1)
	costScale := 1.0
	if ship, ok := s.Arena.Ships[i.shipUid]; ok {
		costScale = pathClearanceScale(ship)
	}
	pathSearches.submit(pathSearchJob{
		grid:      mapCfg.PreparedGrid(),
		start:     start,
		goal:      goal,
		costScale: costScale,
		reply:     i.result,
	})
}

// pathClearanceScale 按舰体半长缩放贴岸代价：小艇可以贴着岸走，大舰需要留更多余量。
// 基准代价是“离岸 1 格 +2、2 格 +1”，这里按半长（格）在 0.5~2 之间缩放。
func pathClearanceScale(ship *objWeapon.BattleShip) float64 {
	halfLengthCells := ship.Length / constants.MapBlockSize / 2
	return min(2, max(0.5, halfLengthCells))
}

// Exec ...
func (i *ShipMovePath) Exec(s *state.MissionState) error {
	mapCfg := s.Core.MissionMD.MapCfg
	i.prepareTarget(mapCfg)
	i.consumePathResult()
	// 寻路失败（异步标记为 Executing），主线程中重置速度后标记完成
	if i.status == Executing {
		if ship, ok := s.Arena.Ships[i.shipUid]; ok {
			ship.CurSpeed = 0
			i.markUnreachable(s, ship)
		}
		i.failed = true
		i.status = Executed
		return nil
	}
	if i.status == Executed {
		return nil
	}

	if i.status != Ready {
		if i.status != Preparing {
			i.submitSearch(s, i.curPos)
		}
		// Preparing 状态下，让战舰继续朝目标方向直线移动作为过渡
		if ship, ok := s.Arena.Ships[i.shipUid]; ok && ship.CurSpeed > 0 {
			ship.MoveTo(mapCfg, i.targetPos, false)
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
	}
	// 每次套用新路径（首次或撞岸重寻）都把航点游标同步到离舰船最近的航点：
	// 否则重寻后舰船会掉头开回已经过的航点，甚至朝身后的陆地开，卡在岸边。
	if i.syncedGeneration != i.pathGeneration {
		i.syncedGeneration = i.pathGeneration
		i.syncPathCursor(ship)
	}

	arrive, blocked := ship.MoveTo(
		mapCfg,
		i.path[i.curIdx],
		i.curIdx == len(i.path)-1,
	)
	i.trackProgress(ship)
	// 长时间既没有位移也没有转向：目标不可达（例如终点被陆地完全隔开），
	// 结束机动并留下失败标记，交给 UI 提示，而不是让舰船永远磨在岸边。
	if i.noProgressTicks >= blockedNoProgressMaxTicks {
		ship.CurSpeed = 0
		i.markUnreachable(s, ship)
		i.failed = true
		i.status = Executed
		return nil
	}
	if blocked {
		i.blockedTicks++
		if i.blockedTicks >= blockedRepathInterval {
			i.blockedTicks = 0
			i.repathWhileBlocked(s, ship)
		}
		return nil
	}
	i.blockedTicks = 0
	i.stalledRepaths = 0
	if arrive {
		i.curIdx++
	}
	return nil
}

// repathWhileBlocked 撞岸受阻时重新规划：沿岸滑动过就以当前位置重寻，
// 让航线跟上实际位置；原地顶住时也允许少量重寻（起点会换成舰船当前格）。
func (i *ShipMovePath) repathWhileBlocked(s *state.MissionState, ship *objWeapon.BattleShip) {
	if ship.CurPos.Distance(i.repathPos) >= blockedRepathMinDrift {
		i.repathPos = ship.CurPos.Copy()
		i.stalledRepaths = 0
		i.repathFrom(s, ship.CurPos)
		return
	}
	if i.stalledRepaths >= blockedRepathMaxStalled {
		return
	}
	i.stalledRepaths++
	i.repathFrom(s, ship.CurPos)
}

// syncPathCursor 把航点游标移到离舰船当前位置最近的有效航点。
// 路径点距离开始增大即说明已经过了最近点，停止搜索，避免折线路径回折时
// 选到绕远的一侧。
func (i *ShipMovePath) syncPathCursor(ship *objWeapon.BattleShip) {
	if len(i.path) == 0 {
		return
	}
	i.curIdx = 0
	minDist := ship.CurPos.Distance(i.path[0])
	for idx := 1; idx < len(i.path); idx++ {
		dist := ship.CurPos.Distance(i.path[idx])
		if dist >= minDist {
			break
		}
		minDist, i.curIdx = dist, idx
	}
}

// markUnreachable 在舰船位置弹出“无法抵达”提示。
// 机动失败时不再静默结束：玩家点了目标却看不到任何反应，最容易误判成“卡住”。
func (i *ShipMovePath) markUnreachable(s *state.MissionState, ship *objWeapon.BattleShip) {
	if s.UI.GameMarks == nil {
		return
	}
	mark := objMark.NewText(ship.CurPos, i18n.Text(i18n.MsgOrderUnreachable), 20, colorx.Red, 90)
	s.UI.GameMarks[mark.ID] = mark
}

// trackProgress 记录舰船是否仍在移动或转向。撞岸掉头时舰体持续转向，
// 因此不会因为“被地形拦停”而被判定为没有进展。
func (i *ShipMovePath) trackProgress(ship *objWeapon.BattleShip) {
	if !i.progressInitialized {
		i.progressInitialized = true
		i.lastProgressPos = ship.CurPos.Copy()
		i.lastProgressRot = ship.CurRotation
		return
	}
	moved := ship.CurPos.Distance(i.lastProgressPos) >= progressEpsilonDistance
	turned := math.Abs(shortestAngleDiff(ship.CurRotation, i.lastProgressRot)) >= progressEpsilonRotation
	if moved || turned {
		i.lastProgressPos = ship.CurPos.Copy()
		i.lastProgressRot = ship.CurRotation
		i.noProgressTicks = 0
		return
	}
	i.noProgressTicks++
}

// shortestAngleDiff 返回两个航向之间的最短有符号夹角（度）。
func shortestAngleDiff(a, b float64) float64 {
	return math.Mod(a-b+540, 360) - 180
}

// repathFrom 撞岸恢复：以战舰当前位置为起点重新提交寻路。
// 起点吸附（MapCfg.SnapPathEndpoints）保证中心点已压进海岸格的搁浅舰也能退出。
func (i *ShipMovePath) repathFrom(s *state.MissionState, curPos objPos.MapPos) {
	i.curPos = curPos
	i.submitSearch(s, curPos)
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
	mapCfg := misState.Core.MissionMD.MapCfg
	i.prepareTarget(mapCfg)
	start, goal := mapCfg.SnapPathEndpoints(
		grid.Point{X: i.curPos.MX, Y: i.curPos.MY},
		grid.Point{X: i.targetPos.MX, Y: i.targetPos.MY},
	)
	i.applyPoints(mapCfg.PreparedGrid().Search(start, goal))
}

func (i *ShipMovePath) applyPoints(points []grid.Point) {
	// 起点与终点在同一格（或吸附到同一格）：直接对精确目标点做最后一段直线逼近。
	if len(points) == 1 {
		i.path = []objPos.MapPos{i.targetPos}
		i.curIdx = 0
		i.pathGeneration++
		i.status = Ready
		i.prependEscapeWaypoint()
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
	i.pathGeneration++
	i.status = Ready
	i.prependEscapeWaypoint()
}

// prependEscapeWaypoint 给搁浅舰（中心点压在陆格上）插入“先退回水面”的第一航点。
func (i *ShipMovePath) prependEscapeWaypoint() {
	if i.escapePos == nil {
		return
	}
	i.path = append([]objPos.MapPos{*i.escapePos}, i.path...)
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
