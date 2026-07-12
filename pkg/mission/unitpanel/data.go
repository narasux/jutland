package unitpanel

import (
	"sort"
	"time"

	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

type toggleState int

const (
	toggleAllowed toggleState = iota
	toggleDisabled
	toggleMixed
)

type weaponRow struct {
	Type            objUnit.WeaponType
	Equipped        int
	Ready           int
	Progress        float64
	RemainingMillis int64
	Toggle          toggleState
}

// selectedShips 返回仍存活的选中舰，并按展示名称和 Uid 稳定排序。
func selectedShips(ms *state.MissionState) []*objUnit.BattleShip {
	ships := make([]*objUnit.BattleShip, 0, len(ms.Interaction.SelectedShips))
	for _, uid := range ms.Interaction.SelectedShips {
		if ship := ms.Arena.Ships[uid]; ship != nil && ship.CurHP > 0 {
			ships = append(ships, ship)
		}
	}
	sort.Slice(ships, func(i, j int) bool {
		left, right := objUnit.GetShipDisplayName(ships[i].Name), objUnit.GetShipDisplayName(ships[j].Name)
		if left == right {
			return ships[i].Uid < ships[j].Uid
		}
		return left < right
	})
	return ships
}

// focusedShip 返回仍存活的焦点舰。
func focusedShip(ms *state.MissionState) *objUnit.BattleShip {
	ship := ms.Arena.Ships[ms.Interaction.FocusedShipUid]
	if ship == nil || ship.CurHP <= 0 {
		return nil
	}
	return ship
}

// hasWeapon 判断战舰是否实际装备指定类型武器。
func hasWeapon(ship *objUnit.BattleShip, weaponType objUnit.WeaponType) bool {
	switch weaponType {
	case objUnit.WeaponTypeMainGun:
		return len(ship.Weapon.MainGuns) > 0
	case objUnit.WeaponTypeSecondaryGun:
		return len(ship.Weapon.SecondaryGuns) > 0
	case objUnit.WeaponTypeAntiAircraftGun:
		return len(ship.Weapon.AntiAircraftGuns) > 0
	case objUnit.WeaponTypeTorpedo:
		return len(ship.Weapon.Torpedoes) > 0
	case objUnit.WeaponTypeRocket:
		return len(ship.Weapon.Rockets) > 0
	default:
		return false
	}
}

func weaponDisabled(ship *objUnit.BattleShip, weaponType objUnit.WeaponType) bool {
	switch weaponType {
	case objUnit.WeaponTypeMainGun:
		return ship.Weapon.MainGunDisabled
	case objUnit.WeaponTypeSecondaryGun:
		return ship.Weapon.SecondaryGunDisabled
	case objUnit.WeaponTypeAntiAircraftGun:
		return ship.Weapon.AntiAircraftGunDisabled
	case objUnit.WeaponTypeTorpedo:
		return ship.Weapon.TorpedoDisabled
	case objUnit.WeaponTypeRocket:
		return ship.Weapon.RocketDisabled
	default:
		return false
	}
}

func equippedWeaponTypes(ships []*objUnit.BattleShip) []objUnit.WeaponType {
	all := []objUnit.WeaponType{
		objUnit.WeaponTypeMainGun,
		objUnit.WeaponTypeSecondaryGun,
		objUnit.WeaponTypeAntiAircraftGun,
		objUnit.WeaponTypeTorpedo,
		objUnit.WeaponTypeRocket,
	}
	result := make([]objUnit.WeaponType, 0, len(all))
	for _, weaponType := range all {
		for _, ship := range ships {
			if hasWeapon(ship, weaponType) {
				result = append(result, weaponType)
				break
			}
		}
	}
	return result
}

// weaponRows 汇总多选舰队的武器就绪数、下一次装填进度和开关状态。
func weaponRows(ms *state.MissionState, nowMillis int64) []weaponRow {
	ships := selectedShips(ms)
	rows := make([]weaponRow, 0, 5)
	for _, weaponType := range equippedWeaponTypes(ships) {
		row := weaponRow{Type: weaponType, Progress: 1}
		disabledCount, applicableCount := 0, 0
		nextRemaining := int64(0)
		for _, ship := range ships {
			if !hasWeapon(ship, weaponType) {
				continue
			}
			applicableCount++
			status := ship.Weapon.ReloadStatus(weaponType, nowMillis)
			row.Equipped += status.Equipped
			row.Ready += status.Ready
			if status.Disabled {
				disabledCount++
			}
			if status.RemainingMillis > 0 && (nextRemaining == 0 || status.RemainingMillis < nextRemaining) {
				nextRemaining = status.RemainingMillis
				row.RemainingMillis = status.RemainingMillis
				row.Progress = status.Progress
			}
		}
		switch {
		case disabledCount == 0:
			row.Toggle = toggleAllowed
		case disabledCount == applicableCount:
			row.Toggle = toggleDisabled
		default:
			row.Toggle = toggleMixed
		}
		rows = append(rows, row)
	}
	return rows
}

func allWeaponsToggle(ships []*objUnit.BattleShip) toggleState {
	total, disabled := 0, 0
	for _, weaponType := range equippedWeaponTypes(ships) {
		for _, ship := range ships {
			if !hasWeapon(ship, weaponType) {
				continue
			}
			total++
			if weaponDisabled(ship, weaponType) {
				disabled++
			}
		}
	}
	switch {
	case disabled == 0:
		return toggleAllowed
	case disabled == total:
		return toggleDisabled
	default:
		return toggleMixed
	}
}

func aircraftToggle(ships []*objUnit.BattleShip) toggleState {
	total, disabled := 0, 0
	for _, ship := range ships {
		if !ship.Aircraft.HasPlane {
			continue
		}
		total++
		if ship.Aircraft.Disable {
			disabled++
		}
	}
	switch {
	case disabled == 0:
		return toggleAllowed
	case disabled == total:
		return toggleDisabled
	default:
		return toggleMixed
	}
}

// aircraftRows 按机型合并选区内全部航空联队，并生成合计行。
func aircraftRows(ms *state.MissionState) ([]objUnit.AircraftGroupStatus, objUnit.AircraftGroupStatus) {
	byName := map[string]objUnit.AircraftGroupStatus{}
	for _, ship := range selectedShips(ms) {
		if !ship.Aircraft.HasPlane {
			continue
		}
		status := ship.Aircraft.Status(ship.Uid, ms.Arena.Planes)
		for _, row := range status.Groups {
			current := byName[row.Name]
			current.Name = row.Name
			current.Standby += row.Standby
			current.InCombat += row.InCombat
			current.Returning += row.Returning
			current.Lost += row.Lost
			byName[row.Name] = current
		}
	}
	rows := make([]objUnit.AircraftGroupStatus, 0, len(byName))
	for _, row := range byName {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return objUnit.GetPlaneDisplayName(rows[i].Name) < objUnit.GetPlaneDisplayName(rows[j].Name)
	})
	total := objUnit.AircraftGroupStatus{Name: "total"}
	for _, row := range rows {
		total.Standby += row.Standby
		total.InCombat += row.InCombat
		total.Returning += row.Returning
		total.Lost += row.Lost
	}
	return rows, total
}

func nowMillis() int64 { return time.Now().UnixMilli() }
