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

func (i *EnableWeapon) Exec(s *state.MissionState) error {
	s.Ships[i.shipUid].EnableWeapon(i.weaponType)
	i.executed = true
	return nil
}

func (i *EnableWeapon) String() string {
	return fmt.Sprintf("Enable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

func (i *EnableWeapon) IsExecuted() bool {
	return i.executed
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

func (i *DisableWeapon) Exec(s *state.MissionState) error {
	s.Ships[i.shipUid].DisableWeapon(i.weaponType)
	i.executed = true
	return nil
}

func (i *DisableWeapon) String() string {
	return fmt.Sprintf("Disable ship %s weapon %s", i.shipUid, string(i.weaponType))
}

func (i *DisableWeapon) IsExecuted() bool {
	return i.executed
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

func (i *ShipMove) Exec(s *state.MissionState) error {
	borderX, borderY := s.MissionMD.MapCfg.Width, s.MissionMD.MapCfg.Height
	if s.Ships[i.shipUid].MoveTo(i.targetPos, borderX, borderY) {
		i.executed = true
	}
	return nil
}

func (i *ShipMove) String() string {
	return fmt.Sprintf("Ship %s move to %s", i.shipUid, i.targetPos)
}

func (i *ShipMove) IsExecuted() bool {
	return i.executed
}
