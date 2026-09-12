package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestPlaneFireTargetPrioritizesCurrentAttackTarget(t *testing.T) {
	fighter := &objUnit.Plane{
		Uid:             "fighter",
		Name:            "A6M5",
		Type:            objUnit.PlaneTypeFighter,
		BelongPlayer:    faction.HumanAlpha,
		CurPos:          objPos.NewR(10, 10),
		CurRotation:     0,
		CurAttackTarget: "tail-target",
		Weapon: objUnit.PlaneWeapon{
			MaxToPlaneRange: 5,
		},
	}
	tailTarget := &objUnit.Plane{
		Uid:          "tail-target",
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(10, 9.5),
	}
	flankTarget := &objUnit.Plane{
		Uid:          "flank-target",
		BelongPlayer: faction.ComputerAlpha,
		CurPos:       objPos.NewR(12, 10),
	}
	manager := &MissionManager{
		state: &state.MissionState{
			Arena: state.MissionArenaState{
				Planes: map[string]*objUnit.Plane{
					fighter.Uid:     fighter,
					tailTarget.Uid:  tailTarget,
					flankTarget.Uid: flankTarget,
				},
			},
		},
	}

	target := manager.planeFireTarget(fighter)
	if target == nil || target.ID() != tailTarget.Uid {
		t.Fatalf("fire target = %v, want current tail target %q", target, tailTarget.Uid)
	}
}
