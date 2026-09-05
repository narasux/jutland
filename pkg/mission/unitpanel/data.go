package unitpanel

import (
	"sort"
	"time"

	objBuilding "github.com/narasux/jutland/pkg/mission/object/building"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

// selectedAirfield 返回仍存在的选中机场；未选中时返回 nil。
func selectedAirfield(ms *state.MissionState) *objBuilding.Airfield {
	if ms.Interaction.SelectedAirfieldUid == "" {
		return nil
	}
	for _, af := range ms.Arena.Airfields {
		if af.Uid == ms.Interaction.SelectedAirfieldUid {
			return af
		}
	}
	return nil
}

// airfieldSquadRow 是驻场机队单机型的展示行（兼作生产机型选择行）。
type airfieldSquadRow struct {
	Name      string
	Stock     int64 // 机库库存
	Max       int64 // 编制上限
	Flying    int64 // 空中数量
	Lost      int64 // 累计损失
	Producing bool  // 是否为当前生产机型（满编自动顺延后的实际生产机型）
	Progress  int   // 生产进度百分比（0-100，非生产机型为 0）
}

// airfieldSquadRows 统计选中机场各机型的库存、出击与损失数量及生产进度。
func airfieldSquadRows(ms *state.MissionState, af *objBuilding.Airfield) []airfieldSquadRow {
	flying := map[string]int64{}
	for _, plane := range ms.Arena.Planes {
		if plane.BelongShip == af.Uid && plane.CurHP > 0 {
			flying[plane.Name]++
		}
	}
	target := af.ProducingTargetIdx(flying)
	rows := make([]airfieldSquadRow, 0, len(af.Aircraft.Groups))
	for idx, group := range af.Aircraft.Groups {
		rows = append(rows, airfieldSquadRow{
			Name:   group.Name,
			Stock:  group.CurCount,
			Max:    group.MaxCount,
			Flying: flying[group.Name],
			Lost:   af.Losses[group.Name],
			// 待命 + 出击 < 上限才生产（只补充损失）；指定机型满编后
			// 自动顺延到下一个未满编机型，全部满编则无人标记
			Producing: idx == target,
			Progress:  af.ProductionProgress(group.Name),
		})
	}
	return rows
}

// airfieldCapacityFull 全部机型满编（待命 + 出击 >= 上限）：无可生产机型。
func airfieldCapacityFull(rows []airfieldSquadRow) bool {
	for _, row := range rows {
		if row.Stock+row.Flying < row.Max {
			return false
		}
	}
	return len(rows) > 0
}

// fundsLow 判断己方机场是否因资金不足而无法开工 / 暂停生产（与生产逻辑
// 一致：按当前应生产机型的单价判断，满编顺延后即顺延后的机型；全部满编时
// 无生产，不提示；敌方机场不提示）。
func fundsLow(ms *state.MissionState, af *objBuilding.Airfield) bool {
	if af.BelongPlayer != ms.Player.CurPlayer || len(af.Aircraft.Groups) == 0 {
		return false
	}
	flying := map[string]int64{}
	for _, plane := range ms.Arena.Planes {
		if plane.BelongShip == af.Uid && plane.CurHP > 0 {
			flying[plane.Name]++
		}
	}
	target := af.ProducingTargetIdx(flying)
	// 全部满编：没有生产，不需要资金提示
	if target < 0 {
		return false
	}
	group := af.Aircraft.Groups[target]
	fundsCost, _ := objUnit.GetPlaneCost(group.Name)
	return ms.Player.CurFunds < fundsCost
}

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
