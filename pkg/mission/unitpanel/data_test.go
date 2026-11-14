package unitpanel

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestWeaponRowsHideUnequippedAndReportMixedState(t *testing.T) {
	first := &objUnit.BattleShip{
		Uid: "a",
		Weapon: objUnit.ShipWeapon{
			MainGuns:        []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}},
			Torpedoes:       []*objUnit.TorpedoLauncher{{ReloadStartAt: 0, ReloadTime: 1}},
			TorpedoDisabled: true,
		},
		CurHP: 1,
	}
	second := &objUnit.BattleShip{
		Uid: "b",
		Weapon: objUnit.ShipWeapon{
			MainGuns:        []*objUnit.Gun{{ReloadStartAt: 0, ReloadTime: 1}},
			MainGunDisabled: true,
		},
		CurHP: 1,
	}
	missionState := panelTestState(first, second)

	rows := weaponRows(missionState, 10_000)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want only main guns and torpedoes", len(rows))
	}
	if rows[0].Type != objUnit.WeaponTypeMainGun || rows[0].Toggle != toggleMixed {
		t.Fatalf("main gun row = %+v", rows[0])
	}
	if rows[1].Type != objUnit.WeaponTypeTorpedo || rows[1].Toggle != toggleDisabled {
		t.Fatalf("torpedo row = %+v", rows[1])
	}
}

func TestWeaponActionEnablesMixedSelectionAndIgnoresUnequipped(t *testing.T) {
	first := &objUnit.BattleShip{Uid: "a", CurHP: 1, Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{{}}}}
	second := &objUnit.BattleShip{
		Uid:    "b",
		CurHP:  1,
		Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{{}}, MainGunDisabled: true},
	}
	third := &objUnit.BattleShip{Uid: "c", CurHP: 1}
	missionState := panelTestState(first, second, third)

	action := New().weaponAction(missionState, objUnit.WeaponTypeMainGun)
	if !action.Enable || len(action.ShipUids) != 2 {
		t.Fatalf("action = %+v, want enable for two equipped ships", action)
	}
}

func panelTestState(ships ...*objUnit.BattleShip) *state.MissionState {
	byUID := make(map[string]*objUnit.BattleShip, len(ships))
	selected := make([]string, 0, len(ships))
	for _, ship := range ships {
		byUID[ship.Uid] = ship
		selected = append(selected, ship.Uid)
	}
	return &state.MissionState{
		Interaction: state.MissionInteractionState{SelectedShips: selected},
		Arena:       state.MissionArenaState{Ships: byUID, Planes: map[string]*objUnit.Plane{}},
	}
}

// TestShipSpeedRowUsesCurrentDiveStateLimit 锁定单位面板航速行：
// 上限跟随当前深度状态，水下更快的潜艇不会显示成“当前航速 > 最大航速”。
func TestShipSpeedRowUsesCurrentDiveStateLimit(t *testing.T) {
	previous := config.G
	t.Cleanup(func() { config.G = previous })
	config.G = &config.GameSettings{SpeedMultiplier: 2}

	// U-2323：水面 9.7 节，水下倍率 1.29（约 12.5 节）。
	ship := &objUnit.BattleShip{
		Uid:                 "sub",
		Type:                objUnit.ShipTypeSubmarine,
		TotalHP:             100,
		CurHP:               100,
		MaxSpeed:            9.7 / objUnit.ShipSpeedScale,
		SubmergedSpeedRatio: 1.29,
		DiveState:           objUnit.DiveStateSubmerged,
		// CurSpeed 是世界系速度，已乘全局倍率。
		CurSpeed: 9.7 * 1.29 / objUnit.ShipSpeedScale * 2,
	}
	ms := panelTestState(ship)
	if got := speedRowValue(t, New().shipGroups(ms, ship)); got != "12.5 / 12.5" {
		t.Fatalf("submerged speed row = %q, want submerged cruise limit", got)
	}

	ship.DiveState = objUnit.DiveStateSurface
	ship.CurSpeed = 9.7 / objUnit.ShipSpeedScale * 2
	if got := speedRowValue(t, New().shipGroups(ms, ship)); got != "9.7 / 9.7" {
		t.Fatalf("surface speed row = %q, want surface speed", got)
	}
}

func speedRowValue(t *testing.T, groups []infoGroup) string {
	t.Helper()
	label := i18n.Text(i18n.MsgUnitPanelSpeed)
	for _, group := range groups {
		for _, item := range group.items {
			if item.label == label {
				return item.value
			}
		}
	}
	t.Fatalf("speed row %q not found", label)
	return ""
}
