package instruction

import (
	"fmt"

	"github.com/pkg/errors"

	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/geometry"
)

// PlaneAttack 攻击
type PlaneAttack struct {
	planeUid      string
	targetObjType obj.ObjectType
	targetUid     string
	status        InstrStatus
}

// NewPlaneAttack ...
func NewPlaneAttack(planeUid string, targetObjType obj.ObjectType, targetUid string) *PlaneAttack {
	return &PlaneAttack{
		planeUid:      planeUid,
		targetObjType: targetObjType,
		targetUid:     targetUid,
		status:        Ready,
	}
}

var _ Instruction = (*PlaneAttack)(nil)

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

	var enemy obj.Hurtable
	var enemyExists bool
	// 获取打击目标
	switch i.targetObjType {
	case obj.ObjectTypeShip:
		enemy, enemyExists = missionState.Ships[i.targetUid]
	case obj.ObjectTypePlane:
		enemy, enemyExists = missionState.Planes[i.targetUid]
	default:
		return errors.Errorf("invalid target obj type: %s", i.targetObjType)
	}
	// 目标不存在，判定已经完成
	if !enemyExists {
		i.status = Executed
		return nil
	}

	// 如果目标存在，则战机应该冲上去贴贴
	eState := enemy.MovementState()
	// 考虑提前量（依赖敌舰 / 敌机速度，角度）
	_, targetRx, targetRY := geometry.CalcWeaponFireAngle(
		attacker.CurPos.RX, attacker.CurPos.RY, attacker.MaxSpeed,
		eState.CurPos.RX, eState.CurPos.RY, eState.CurSpeed, eState.CurRotation,
	)
	targetPos := obj.NewMapPosR(targetRx, targetRY)
	attacker.MoveTo(missionState.MissionMD.MapCfg, targetPos)
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
		plane.CurPos.RX, plane.CurPos.RY, plane.MaxSpeed,
		ship.CurPos.RX, ship.CurPos.RY, ship.CurSpeed, ship.CurRotation,
	)
	targetPos := obj.NewMapPosR(targetRx, targetRY)
	plane.MoveTo(missionState.MissionMD.MapCfg, targetPos)
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
