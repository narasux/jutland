package instruction

import (
	"fmt"

	obj "github.com/narasux/jutland/pkg/mission/object"
	"github.com/narasux/jutland/pkg/mission/state"
)

// EnableWeapon 启用武器
type EnableWeapon struct {
	shipUid    string
	weaponType obj.WeaponType
	executed   bool
}

// NewEnableWeapon ...
func NewEnableWeapon(shipUid string, weaponType obj.WeaponType) *EnableWeapon {
	return &EnableWeapon{shipUid: shipUid, weaponType: weaponType}
}

var _ Instruction = (*EnableWeapon)(nil)

func (i *EnableWeapon) Exec(s *state.MissionState) error {
	// 战舰如果被摧毁了，直接修改指令为已完成
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		i.executed = true
		return nil
	}

	ship.EnableWeapon(i.weaponType)
	i.executed = true
	return nil
}

func (i *EnableWeapon) Executed() bool {
	return i.executed
}

func (i *EnableWeapon) Uid() string {
	return GenInstrUid(NameEnableWeapon, i.shipUid)
}

func (i *EnableWeapon) String() string {
	return fmt.Sprintf("Enable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

// DisableWeapon 禁用武器
type DisableWeapon struct {
	shipUid    string
	weaponType obj.WeaponType
	executed   bool
}

// NewDisableWeapon ...
func NewDisableWeapon(shipUid string, weaponType obj.WeaponType) *DisableWeapon {
	return &DisableWeapon{shipUid: shipUid, weaponType: weaponType}
}

var _ Instruction = (*DisableWeapon)(nil)

func (i *DisableWeapon) Exec(s *state.MissionState) error {
	// 战舰如果被摧毁了，直接修改指令为已完成
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		i.executed = true
		return nil
	}

	ship.DisableWeapon(i.weaponType)
	i.executed = true
	return nil
}

func (i *DisableWeapon) Executed() bool {
	return i.executed
}

func (i *DisableWeapon) Uid() string {
	return GenInstrUid(NameDisableWeapon, i.shipUid)
}

func (i *DisableWeapon) String() string {
	return fmt.Sprintf("Disable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

// ShipMove 移动
type ShipMove struct {
	shipUid   string
	targetPos obj.MapPos
	executed  bool
}

// NewShipMove ...
func NewShipMove(shipUid string, targetPos obj.MapPos) *ShipMove {
	return &ShipMove{shipUid: shipUid, targetPos: targetPos}
}

var _ Instruction = (*ShipMove)(nil)

func (i *ShipMove) Exec(s *state.MissionState) error {
	// 战舰如果被摧毁了，直接修改指令为已完成
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		i.executed = true
		return nil
	}

	if ship.MoveTo(s.MissionMD.MapCfg, i.targetPos, true) {
		i.executed = true
	}
	return nil
}

func (i *ShipMove) Executed() bool {
	return i.executed
}

func (i *ShipMove) Uid() string {
	return GenInstrUid(NameShipMove, i.shipUid)
}

func (i *ShipMove) String() string {
	return fmt.Sprintf("Ship %s move to %s", i.shipUid, i.targetPos.String())
}

// ShipMovePath 按照指定路径移动
type ShipMovePath struct {
	shipUid  string
	path     []obj.MapPos
	curIdx   int
	executed bool
}

// NewShipMovePath ...
func NewShipMovePath(shipUid string, path []obj.MapPos) *ShipMovePath {
	return &ShipMovePath{shipUid: shipUid, path: path}
}

var _ Instruction = (*ShipMovePath)(nil)

func (i *ShipMovePath) Exec(s *state.MissionState) error {
	// 战舰如果被摧毁了，直接修改指令为已完成
	ship, ok := s.Ships[i.shipUid]
	if !ok {
		i.executed = true
		return nil
	}

	if i.curIdx >= len(i.path) {
		i.executed = true
		return nil
	}
	if ship.MoveTo(s.MissionMD.MapCfg, i.path[i.curIdx], i.curIdx == len(i.path)-1) {
		i.curIdx++
	}
	return nil
}

func (i *ShipMovePath) Executed() bool {
	return i.executed
}

func (i *ShipMovePath) Uid() string {
	return GenInstrUid(NameShipMovePath, i.shipUid)
}

func (i *ShipMovePath) String() string {
	return fmt.Sprintf("Ship %s move with path %s", i.shipUid, i.path)
}
