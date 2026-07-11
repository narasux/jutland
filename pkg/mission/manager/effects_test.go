package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func TestDestroyedLandingPlaneReleasesSlot(t *testing.T) {
	ship := &objUnit.BattleShip{Uid: "carrier"}
	first := &objUnit.Plane{
		Uid:        "first",
		BelongShip: ship.Uid,
		CurHP:      0,
		CurPos:     objPos.NewR(10, 10),
	}
	second := &objUnit.Plane{
		Uid:        "second",
		BelongShip: ship.Uid,
		CurHP:      10,
	}
	ship.Aircraft.RequestLanding(first.Uid)
	ship.Aircraft.RequestLanding(second.Uid)

	manager := &MissionManager{state: &state.MissionState{
		Core: state.MissionCoreState{
			MissionMD: metadata.MissionMetadata{
				MapCfg: &mapcfg.MapCfg{Width: 100, Height: 100},
			},
		},
		Arena: state.MissionArenaState{
			Ships: map[string]*objUnit.BattleShip{ship.Uid: ship},
			Planes: map[string]*objUnit.Plane{
				first.Uid:  first,
				second.Uid: second,
			},
		},
	}}
	manager.updateMissionPlanes()

	if _, exists := manager.state.Arena.Planes[first.Uid]; exists {
		t.Fatalf("destroyed plane was not removed from arena")
	}
	if slot := ship.Aircraft.RequestLanding("replacement"); slot != 0 {
		t.Fatalf("destroyed plane slot was not released: replacement slot = %d", slot)
	}
}
