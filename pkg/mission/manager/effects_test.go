package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objTrail "github.com/narasux/jutland/pkg/mission/object/trail"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	textureImg "github.com/narasux/jutland/pkg/resources/images/texture"
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

func newWakeTrail(ownerUID string) *objTrail.Trail {
	trail := objTrail.New(
		objPos.NewR(10, 10), textureImg.TrailShapeCircle,
		10, 1,
		100, 1,
		0, 0, nil,
	)
	trail.OwnerUid = ownerUID
	return trail
}

func TestStoppedShipWakeFadesOutInsteadOfLeavingIsolatedDots(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	trail := newWakeTrail("ship")
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{
					"ship": {
						Uid:          "ship",
						CurHP:        100,
						TotalHP:      100,
						CurSpeed:     0,
						CurPos:       objPos.NewR(10, 10),
						BelongPlayer: faction.HumanAlpha,
					},
				},
				Trails: []*objTrail.Trail{trail},
			},
		},
	}

	manager.updateObjectTrails()
	if trail.LifeReductionRate <= 1 {
		t.Fatalf("stopped wake fade rate = %v, want faster than normal", trail.LifeReductionRate)
	}
	for range stoppedShipWakeFadeFrames {
		manager.updateObjectTrails()
	}
	if len(manager.state.Arena.Trails) != 0 {
		t.Fatalf("stopped wake trails = %d, want 0", len(manager.state.Arena.Trails))
	}
}

func TestMovingShipWakeKeepsNormalFade(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	trail := newWakeTrail("ship")
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Ships: map[string]*objUnit.BattleShip{
					"ship": {
						Uid:          "ship",
						CurHP:        100,
						TotalHP:      100,
						CurSpeed:     0.1,
						CurPos:       objPos.NewR(10, 10),
						BelongPlayer: faction.HumanAlpha,
					},
				},
				Trails: []*objTrail.Trail{trail},
			},
		},
	}

	manager.updateObjectTrails()
	if trail.LifeReductionRate != 1 {
		t.Fatalf("moving wake fade rate = %v, want 1", trail.LifeReductionRate)
	}
}
