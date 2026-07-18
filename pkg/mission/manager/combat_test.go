package manager

import (
	"strings"
	"testing"

	audioPlayer "github.com/narasux/jutland/pkg/audio/player"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/resources/mapcfg"
)

func TestTorpedoBomberSkipsReleaseWhenPathCrossesLand(t *testing.T) {
	oldSettings := config.G
	config.G = config.NewDefaultGameSettings()
	t.Cleanup(func() { config.G = oldSettings })

	const planeName = "test-land-check-torpedo-bomber"
	oldTemplate, hadTemplate := objUnit.PlaneMap[planeName]
	objUnit.PlaneMap[planeName] = &objUnit.Plane{
		Name: planeName,
		Type: objUnit.PlaneTypeTorpedoBomber,
		Weapon: objUnit.PlaneWeapon{
			Torpedoes: []*objUnit.Releaser{{Range: 3}},
		},
	}
	t.Cleanup(func() {
		if hadTemplate {
			objUnit.PlaneMap[planeName] = oldTemplate
		} else {
			delete(objUnit.PlaneMap, planeName)
		}
	})

	torpedo := &objUnit.Releaser{
		Range:          3,
		BulletSpeed:    0.25,
		RightFiringArc: objUnit.FiringArc{Start: 0, End: 180},
	}
	plane := &objUnit.Plane{
		Name:            planeName,
		Type:            objUnit.PlaneTypeTorpedoBomber,
		Uid:             "torpedo-bomber",
		CurHP:           100,
		CurPos:          objPos.NewR(0, 0.5),
		CurRotation:     0,
		BelongPlayer:    faction.HumanAlpha,
		CurAttackTarget: "unsafe-target",
		Weapon: objUnit.PlaneWeapon{
			Torpedoes:      []*objUnit.Releaser{torpedo},
			MaxToShipRange: 3,
		},
	}
	unsafeTarget := &objUnit.BattleShip{
		Uid:          "unsafe-target",
		CurHP:        100,
		CurPos:       objPos.NewR(2, 0.5),
		BelongPlayer: faction.ComputerAlpha,
	}
	alternateTarget := &objUnit.BattleShip{
		Uid:          "alternate-target",
		CurHP:        100,
		CurPos:       objPos.New(9, 1),
		BelongPlayer: faction.ComputerAlpha,
	}

	missionState := &state.MissionState{
		Core: state.MissionCoreState{MissionMD: metadata.MissionMetadata{
			MapCfg: &mapcfg.MapCfg{Map: mapcfg.MapData{
				"SCSSSSSSSS",
				"..........",
			}},
		}},
		Arena: state.MissionArenaState{
			Planes: map[string]*objUnit.Plane{plane.Uid: plane},
			Ships: map[string]*objUnit.BattleShip{
				unsafeTarget.Uid:    unsafeTarget,
				alternateTarget.Uid: alternateTarget,
			},
		},
	}
	instructionSet := NewInstructionSet()
	instructionSet.Add(instr.NewPlaneAttack(plane.Uid, unsafeTarget.ObjType(), unsafeTarget.Uid))
	manager := &MissionManager{
		state:            missionState,
		instructionSet:   instructionSet,
		weaponFirePlayer: audioPlayer.NewWeaponFire(),
	}
	if !plane.TorpedoPathCrossesLand(unsafeTarget, &missionState.Core.MissionMD.MapCfg.Map) {
		t.Fatal("expected land between torpedo bomber and target to block the path")
	}
	safeTerrain := mapcfg.MapData{"SSSSSSSSSS"}
	if plane.TorpedoPathCrossesLand(unsafeTarget, &safeTerrain) {
		t.Fatal("torpedo path over open water was reported as crossing land")
	}

	manager.updatePlaneWeaponFire()

	if torpedo.Released {
		t.Fatal("torpedo was released over land")
	}
	if len(missionState.Arena.ForwardingBullets) != 0 {
		t.Fatalf("forwarding bullets = %d, want 0", len(missionState.Arena.ForwardingBullets))
	}
	if plane.CurAttackTarget != alternateTarget.Uid {
		t.Fatalf("attack target = %q, want %q", plane.CurAttackTarget, alternateTarget.Uid)
	}
	attackInstruction := instructionSet.Items()[instr.GenInstrUid(instr.NamePlaneAttack, plane.Uid)]
	if attackInstruction == nil || !strings.Contains(attackInstruction.String(), alternateTarget.Uid) {
		t.Fatalf("attack instruction = %v, want target %q", attackInstruction, alternateTarget.Uid)
	}
}
