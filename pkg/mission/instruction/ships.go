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

func (i *EnableWeapon) IsExecuted() bool {
	return i.executed
}

func (i *EnableWeapon) GetObjUid() string {
	return i.shipUid
}

func (i *EnableWeapon) String() string {
	return fmt.Sprintf("Enable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

func (i *EnableWeapon) Name() string {
	return NameEnableWeapon
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

func (i *DisableWeapon) IsExecuted() bool {
	return i.executed
}

func (i *DisableWeapon) GetObjUid() string {
	return i.shipUid
}

func (i *DisableWeapon) String() string {
	return fmt.Sprintf("Disable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

func (i *DisableWeapon) Name() string {
	return NameDisableWeapon
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

	borderX, borderY := s.MissionMD.MapCfg.Width, s.MissionMD.MapCfg.Height
	if ship.MoveTo(i.targetPos, borderX, borderY) {
		i.executed = true
	}
	return nil
}

func (i *ShipMove) IsExecuted() bool {
	return i.executed
}

func (i *ShipMove) GetObjUid() string {
	return i.shipUid
}

func (i *ShipMove) String() string {
	return fmt.Sprintf("Ship %s move to %s", i.shipUid, i.targetPos.String())
}

func (i *ShipMove) Name() string {
	return NameShipMove
}
