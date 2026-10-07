package instruction

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/narasux/jutland/pkg/mission/object"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// PlaneAttack 攻击
type PlaneAttack struct {
	planeUid      string
	targetObjType object.Type
	targetUid     string
	status        InstrStatus
	// 投放-脱离（Hit-and-Run）相关字段：
	// 飞机攻击战舰时，炸弹要整舱投完才脱离（一次通场把载弹全部投给同一目标），
	// 鱼雷保持一次通场投一雷即脱离；脱离后由 daemon 分配下一个目标或安排返航。
	// 通过快照比对方式检测武器释放进度。
	releaserSnapshot []bool // 指令创建时各释放器（炸弹+鱼雷）的 Released 状态快照
	snapshotTaken    bool   // 标记是否已拍摄快照，避免重复拍摄覆盖初始状态
}

// NewPlaneAttack 创建飞机攻击指令。
func NewPlaneAttack(planeUid string, targetObjType object.Type, targetUid string) *PlaneAttack {
	return &PlaneAttack{
		planeUid:      planeUid,
		targetObjType: targetObjType,
		targetUid:     targetUid,
		status:        Ready,
	}
}

var _ Instruction = (*PlaneAttack)(nil)

// takeReleaserSnapshot 拍摄释放器状态快照
// 按 Bombs → Torpedoes 的顺序，记录每个释放器当前的 Released 状态，
// 后续通过 shouldDisengage 比对，判断是否已完成投放、应该脱离
func (i *PlaneAttack) takeReleaserSnapshot(plane *objUnit.Plane) {
	if i.snapshotTaken {
		return
	}
	i.snapshotTaken = true
	i.releaserSnapshot = make([]bool, len(plane.Weapon.Bombs)+len(plane.Weapon.Torpedoes))
	idx := 0
	for _, b := range plane.Weapon.Bombs {
		i.releaserSnapshot[idx] = b.Released
		idx++
	}
	for _, t := range plane.Weapon.Torpedoes {
		i.releaserSnapshot[idx] = t.Released
		idx++
	}
}

// shouldDisengage 判断本次攻击是否已完成投放、应该脱离：
//   - 炸弹：指令开始时还有未投的炸弹，且现在全部投完，才算完成一次完整投弹，
//     中途不脱离，保证整舱投给同一个目标（中途脱离会让飞机每次通场只丢一枚）；
//   - 鱼雷：任一鱼雷新释放即脱离，鱼雷机一次通场只投一雷。
//
// 指令开始时炸弹就已投完（如纯鱼雷机后续通场），不走炸弹脱离，避免刚接敌就掉头。
func (i *PlaneAttack) shouldDisengage(plane *objUnit.Plane) bool {
	if !i.snapshotTaken {
		return false
	}
	idx := 0
	bombsPendingAtStart, bombsAllReleased := false, true
	for _, b := range plane.Weapon.Bombs {
		if idx < len(i.releaserSnapshot) && !i.releaserSnapshot[idx] {
			bombsPendingAtStart = true
		}
		if !b.Released {
			bombsAllReleased = false
		}
		idx++
	}
	if bombsPendingAtStart && bombsAllReleased {
		return true
	}
	for _, t := range plane.Weapon.Torpedoes {
		if idx < len(i.releaserSnapshot) && !i.releaserSnapshot[idx] && t.Released {
			return true
		}
		idx++
	}
	return false
}

// Exec 执行飞机攻击指令。
// 起飞阶段只推进滑跑；巡航阶段持续追踪目标，整舱炸弹投完（或投出一雷）后
// 结束当前指令，由 MissionManager 在下一帧重新分配目标或安排返航。
func (i *PlaneAttack) Exec(missionState *state.MissionState) error {
	// 获取攻击方飞机
	attacker, ok := missionState.Arena.Planes[i.planeUid]
	// 攻击方已经不存在，判定已经完成
	if !ok {
		i.status = Executed
		return nil
	}
	// 所属基地可以是航母或陆地机场；基地缺失（如航母被击沉）时退化为自由飞行
	base, _ := missionState.FindAircraftBase(attacker.BelongShip)
	mapCfg := missionState.Core.MissionMD.MapCfg
	// 起飞阶段沿跑道 / 甲板弹射线滑跑，不立即转向接敌。
	if attacker.FlightPhase == objUnit.PlaneFlightPhaseTakingOff {
		// 滑跑阶段也要先认领目标，让空中追击名额统计得到这架飞机，
		// 否则同一架敌机会在滑跑的几秒里被反复超额分配。
		attacker.CurAttackTarget = i.targetUid
		attacker.UpdateTakeoff(mapCfg, base)
		return nil
	}
	if !attacker.IsCruising() {
		i.status = Executed
		return nil
	}
	// 如果必须返航，则判定已经完成
	if attacker.MustReturn() {
		i.status = Executed
		return nil
	}
	// 设置攻击目标
	attacker.CurAttackTarget = i.targetUid

	var enemy objUnit.Hurtable
	var enemyExists bool
	// 获取打击目标
	switch i.targetObjType {
	case object.TypeShip:
		enemy, enemyExists = missionState.Arena.Ships[i.targetUid]
	case object.TypePlane:
		enemy, enemyExists = missionState.Arena.Planes[i.targetUid]
	default:
		return errors.Errorf("invalid target obj type: %v", i.targetObjType)
	}
	// 目标不存在，判定已经完成
	if !enemyExists {
		i.status = Executed
		return nil
	}

	// 对舰机型（轰炸机/鱼雷机）只能攻击地面飞机（炸弹/鱼雷的投放门槛）；
	// 锁定的地面目标一旦升空，继续追踪只会绕着机场空转，这里直接终止
	// 指令，交由调度进程重新分配目标或触发返航。
	if i.targetObjType == object.TypePlane && attacker.AttackObjType() == object.TypeShip {
		if target, ok := enemy.(*objUnit.Plane); ok && !target.IsOnGround() {
			i.status = Executed
			return nil
		}
	}

	// 投放-脱离逻辑（仅对战舰目标生效，对飞机目标保持持续追踪直到击落）：
	// 飞机对战舰的攻击是把整舱炸弹投给同一个目标（鱼雷则一次通场一雷），
	// 投完即脱离，不需要持续追踪同一目标直到其被击沉。
	// 脱离后 daemon 进程（updatePlaneAttackOrReturn）会在下一帧检测到该飞机
	// 无攻击指令，并为其重新分配目标或触发返航。
	if i.targetObjType == object.TypeShip {
		// 首次执行时拍摄快照，记录此刻各释放器的状态作为基准线
		i.takeReleaserSnapshot(attacker)
		// 炸弹投完 / 鱼雷新释放时脱离目标
		if i.shouldDisengage(attacker) {
			attacker.CurAttackTarget = ""
			i.status = Executed
			return nil
		}
	}

	// 如果目标存在，则战机应该冲上去贴贴
	eState := enemy.MovementState()
	// 用实际机载武器弹速计算提前点；追击航向与开火判定共用同一落点。
	leadSpeed := attacker.AttackLeadSpeed(i.targetObjType)
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		attacker.CurPos.RX, attacker.CurPos.RY, leadSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)
	// 传递目标位置、敌人当前位置和目标速度，用于战斗机追踪时调整速度
	attacker.MoveTo(missionState.Core.MissionMD.MapCfg, targetPos, eState.CurPos, eState.CurSpeed)
	return nil
}

// Executed 返回攻击指令是否已经执行完成。
func (i *PlaneAttack) Executed() bool {
	return i.status == Executed
}

// Uid 返回攻击指令的唯一 ID。
func (i *PlaneAttack) Uid() string {
	return GenInstrUid(NamePlaneAttack, i.planeUid)
}

// String 返回攻击指令的可读描述。
func (i *PlaneAttack) String() string {
	return fmt.Sprintf("Plane %s attack %s", i.planeUid, i.targetUid)
}

// PlaneReturn 返航
type PlaneReturn struct {
	planeUid string
	status   InstrStatus
}

// NewPlaneReturn 创建飞机返航指令。
func NewPlaneReturn(planeUid string) *PlaneReturn {
	return &PlaneReturn{planeUid: planeUid}
}

var _ Instruction = (*PlaneReturn)(nil)

// Exec 执行飞机返航指令。
// 根据当前飞行阶段推进待场、进近和着舰流程，最终从任务状态中移除并回收飞机。
func (i *PlaneReturn) Exec(missionState *state.MissionState) error {
	// 获取飞机
	plane, ok := missionState.Arena.Planes[i.planeUid]
	// 飞机已经不存在，判定已经完成
	if !ok {
		i.status = Executed
		return nil
	}

	base, ok := missionState.FindAircraftBase(plane.BelongShip)
	if !ok {
		// FIXME 目前载具如果沉没，则飞机也直接坠毁，后续考虑备降到其他地方
		plane.CurHP = 0
		i.status = Executed
		return nil
	}

	mapCfg := missionState.Core.MissionMD.MapCfg
	switch plane.FlightPhase {
	case objUnit.PlaneFlightPhaseTakingOff:
		plane.UpdateTakeoff(mapCfg, base)
	case "", objUnit.PlaneFlightPhaseCruising:
		slot := base.BaseAircraft().RequestLanding(plane.Uid)
		plane.StartLandingStaging(mapCfg, base, slot)
	case objUnit.PlaneFlightPhaseLandingStaging:
		if plane.UpdateLandingStaging(mapCfg, base) {
			plane.StartLandingApproach(base)
		}
	case objUnit.PlaneFlightPhaseLandingApproach:
		if plane.UpdateLandingApproach(base) {
			plane.StartLandingDeck(base)
		}
	case objUnit.PlaneFlightPhaseLandingDeck:
		if !plane.UpdateLandingDeck(base) {
			return nil
		}
		i.recoverPlane(missionState, base, plane)
	}
	return nil
}

// recoverPlane 将返航飞机归还基地库存，并从在场飞机集合中移除。
func (i *PlaneReturn) recoverPlane(
	missionState *state.MissionState,
	base objUnit.AircraftBase,
	plane *objUnit.Plane,
) {
	// 回收（航母与机场一致）：入库后移除活动实体；飞机无回收价值时按坠毁处理
	base.BaseAircraft().Recovery(plane)
	missionState.Arena.RemovePlane(i.planeUid)
	i.status = Executed
}

// Executed 返回返航指令是否已经执行完成。
func (i *PlaneReturn) Executed() bool {
	return i.status == Executed
}

// Uid 返回返航指令的唯一 ID。
func (i *PlaneReturn) Uid() string {
	return GenInstrUid(NamePlaneReturn, i.planeUid)
}

// String 返回返航指令的可读描述。
func (i *PlaneReturn) String() string {
	return fmt.Sprintf("Plane %s return", i.planeUid)
}
