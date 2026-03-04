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
	// 一击脱离（Hit-and-Run）相关字段：
	// 飞机攻击战舰时，释放鱼雷/炸弹后应立即脱离，由 daemon 分配下一个目标，
	// 而非持续追踪同一目标直到其被击沉。通过快照比对方式检测武器释放事件。
	releaserSnapshot []bool // 指令创建时各释放器（炸弹+鱼雷）的 Released 状态快照
	snapshotTaken    bool   // 标记是否已拍摄快照，避免重复拍摄覆盖初始状态
}

// NewPlaneAttack ...
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
// 后续通过 hasNewRelease 比对，发现从 false → true 的变化即视为完成了一次攻击。
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

// hasNewRelease 检测是否有新的释放（鱼雷/炸弹）
// 将当前释放器状态与快照逐一比对，任一释放器从 false（快照时未释放）
// 变为 true（当前已释放），即认为飞机已完成一次攻击投弹，应脱离目标。
func (i *PlaneAttack) hasNewRelease(plane *objUnit.Plane) bool {
	if !i.snapshotTaken {
		return false
	}
	idx := 0
	for _, b := range plane.Weapon.Bombs {
		if idx < len(i.releaserSnapshot) && !i.releaserSnapshot[idx] && b.Released {
			return true
		}
		idx++
	}
	for _, t := range plane.Weapon.Torpedoes {
		if idx < len(i.releaserSnapshot) && !i.releaserSnapshot[idx] && t.Released {
			return true
		}
		idx++
	}
	return false
}

// Exec 执行指令
func (i *PlaneAttack) Exec(missionState *state.MissionState) error {
	// 获取攻击方飞机
	attacker, ok := missionState.Planes[i.planeUid]
	// 攻击方已经不存在，判定已经完成
	if !ok {
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
		enemy, enemyExists = missionState.Ships[i.targetUid]
	case object.TypePlane:
		enemy, enemyExists = missionState.Planes[i.targetUid]
	default:
		return errors.Errorf("invalid target obj type: %s", i.targetObjType)
	}
	// 目标不存在，判定已经完成
	if !enemyExists {
		i.status = Executed
		return nil
	}

	// 一击脱离逻辑（仅对战舰目标生效，对飞机目标保持持续追踪直到击落）：
	// 飞机对战舰的攻击本质上是投弹/投雷后即脱离，不需要像空战那样持续缠斗。
	// 脱离后 daemon 进程（updatePlaneAttackOrReturn）会在下一帧检测到该飞机
	// 无攻击指令，并为其重新分配目标或触发返航。
	if i.targetObjType == object.TypeShip {
		// 首次执行时拍摄快照，记录此刻各释放器的状态作为基准线
		i.takeReleaserSnapshot(attacker)
		// 检测是否有新的鱼雷/炸弹释放，如果有则一击脱离
		if i.hasNewRelease(attacker) {
			attacker.CurAttackTarget = ""
			i.status = Executed
			return nil
		}
	}

	// 如果目标存在，则战机应该冲上去贴贴
	eState := enemy.MovementState()
	// 考虑提前量（依赖敌舰 / 敌机速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		attacker.CurPos.RX, attacker.CurPos.RY, attacker.CurSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)
	// 传递目标位置、敌人当前位置和目标速度，用于战斗机追踪时调整速度
	attacker.MoveTo(missionState.MissionMD.MapCfg, targetPos, eState.CurPos, eState.CurSpeed)
	return nil
}

// Executed 返回指令是否已经执行
func (i *PlaneAttack) Executed() bool {
	return i.status == Executed
}

// Uid 返回指令唯一ID
func (i *PlaneAttack) Uid() string {
	return GenInstrUid(NamePlaneAttack, i.planeUid)
}

// String 返回指令的描述
func (i *PlaneAttack) String() string {
	return fmt.Sprintf("Plane %s attack %s", i.planeUid, i.targetUid)
}

// PlaneReturn 返航
type PlaneReturn struct {
	planeUid string
	status   InstrStatus
}

// NewPlaneReturn ...
func NewPlaneReturn(planeUid string) *PlaneReturn {
	return &PlaneReturn{planeUid: planeUid}
}

var _ Instruction = (*PlaneReturn)(nil)

// Exec 执行指令
func (i *PlaneReturn) Exec(missionState *state.MissionState) error {
	// 获取飞机
	plane, ok := missionState.Planes[i.planeUid]
	// 飞机已经不存在，判定已经完成
	if !ok {
		i.status = Executed
		return nil
	}

	ship, ok := missionState.Ships[plane.BelongShip]
	if !ok {
		// FIXME 目前载舰如果沉没，则飞机也直接坠毁，后续考虑备降到其他地方
		plane.CurHP = 0
		i.status = Executed
		return nil
	}

	// 考虑提前量（依赖母舰速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		plane.CurPos.RX, plane.CurPos.RY, plane.CurSpeed,
		ship.CurPos.RX, ship.CurPos.RY, ship.CurSpeed, ship.CurRotation,
	)
	targetPos := objPos.NewR(targetRx, targetRY)
	// 返航时不需要调整速度（敌人位置和目标速度都传空值/0）
	plane.MoveTo(missionState.MissionMD.MapCfg, targetPos, objPos.New(0, 0), 0)
	// 飞机到达载舰附近，判定已经完成
	if plane.CurPos.Near(ship.CurPos, 1) {
		// 尝试回收飞机
		ship.Aircraft.Recovery(plane)
		// 删除该飞机（视同着陆）
		delete(missionState.Planes, i.planeUid)
		// 修改状态为已经完成
		i.status = Executed
	}
	return nil
}

// Executed 返回指令是否已经执行
func (i *PlaneReturn) Executed() bool {
	return i.status == Executed
}

// Uid 返回指令唯一ID
func (i *PlaneReturn) Uid() string {
	return GenInstrUid(NamePlaneReturn, i.planeUid)
}

// String 返回指令的描述
func (i *PlaneReturn) String() string {
	return fmt.Sprintf("Plane %s return", i.planeUid)
}
