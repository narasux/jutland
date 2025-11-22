package instruction

import (
	"fmt"

	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/utils/grid"
)

// EnableWeapon 启用武器
type EnableWeapon struct {
	shipUid    string
	weaponType obj.WeaponType
	status     InstrStatus
}

// NewEnableWeapon ...
func NewEnableWeapon(shipUid string, weaponType obj.WeaponType) *EnableWeapon {
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
	weaponType obj.WeaponType
	status     InstrStatus
}

// NewDisableWeapon ...
func NewDisableWeapon(shipUid string, weaponType obj.WeaponType) *DisableWeapon {
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
	targetPos obj.MapPos
	status    InstrStatus
}

// NewShipMove ...
func NewShipMove(shipUid string, targetPos obj.MapPos) *ShipMove {
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
	curPos    obj.MapPos
	targetPos obj.MapPos
	path      []obj.MapPos
	curIdx    int
	status    InstrStatus
}

// NewShipMovePath ...
func NewShipMovePath(shipUid string, curPos, targetPos obj.MapPos) *ShipMovePath {
	return &ShipMovePath{shipUid: shipUid, curPos: curPos, targetPos: targetPos, status: Pending}
}

var _ Instruction = (*ShipMovePath)(nil)

// Exec ...
func (i *ShipMovePath) Exec(s *state.MissionState) error {
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
	ship, ok := s.Ships[i.shipUid]
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

// genPath 生成战舰移动的路径
func (i *ShipMovePath) genPath(misState *state.MissionState) {
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
