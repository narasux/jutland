package unit

import (
	"testing"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/mission/object"
)

func TestPlaneWeaponReleaseIntervalUnmarshalsFractionalSeconds(t *testing.T) {
	var weapon PlaneWeapon
	if err := json5.Unmarshal([]byte(`{releaseInterval: 0.3}`), &weapon); err != nil {
		t.Fatalf("unmarshal releaseInterval: %v", err)
	}
	if weapon.ReleaseInterval != 0.3 {
		t.Fatalf("releaseInterval = %v, want 0.3", weapon.ReleaseInterval)
	}
}

func TestPlaneTypeAttacksShips(t *testing.T) {
	if PlaneTypeFighter.AttacksShips() {
		t.Fatal("fighter should not attack ships")
	}
	for _, planeType := range []PlaneType{
		PlaneTypeDiveBomber, PlaneTypeLevelBomber, PlaneTypeTorpedoBomber, PlaneTypeAttacker,
	} {
		if !planeType.AttacksShips() {
			t.Fatalf("%s should attack ships", planeType)
		}
	}
}

func TestGetPlaneTargetObjTypeAttacker(t *testing.T) {
	previous := PlaneMap
	t.Cleanup(func() { PlaneMap = previous })

	PlaneMap = map[string]*Plane{
		"Il-2": {
			Name:   "Il-2",
			Type:   PlaneTypeAttacker,
			Weapon: PlaneWeapon{Guns: []*Gun{{}}, Rockets: []*PlaneRocketLauncher{{}}},
		},
	}
	if got := GetPlaneTargetObjType("Il-2"); got != object.TypeShip {
		t.Fatalf("attacker target = %v, want ship", got)
	}
	if got := ResolvePlaneSight(PlaneTypeAttacker, 0); got != SightRangeBomber {
		t.Fatalf("attacker sight = %v, want %v", got, SightRangeBomber)
	}
}

func TestGetPlaneTargetObjTypeLevelBomber(t *testing.T) {
	previous := PlaneMap
	t.Cleanup(func() { PlaneMap = previous })

	PlaneMap = map[string]*Plane{
		"SeaOtter": {
			Name:   "SeaOtter",
			Type:   PlaneTypeLevelBomber,
			Weapon: PlaneWeapon{Bombs: []*Releaser{{}}},
		},
	}
	if got := GetPlaneTargetObjType("SeaOtter"); got != object.TypeShip {
		t.Fatalf("level bomber target = %v, want ship", got)
	}
}
