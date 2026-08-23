package unitpanel

import (
	"testing"

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
	second := &objUnit.BattleShip{Uid: "b", CurHP: 1, Weapon: objUnit.ShipWeapon{MainGuns: []*objUnit.Gun{{}}, MainGunDisabled: true}}
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
