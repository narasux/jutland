package manager

import (
	"testing"

	"github.com/narasux/jutland/pkg/mission/faction"
	objMark "github.com/narasux/jutland/pkg/mission/object/mark"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
)

func TestSupportShipHealing(t *testing.T) {
	for _, supportType := range []objUnit.ShipType{
		objUnit.ShipTypeHospital,
		objUnit.ShipTypeRepair,
	} {
		t.Run(string(supportType), func(t *testing.T) {
			support := &objUnit.BattleShip{
				Uid:          "support",
				Type:         supportType,
				CurHP:        100,
				TotalHP:      100,
				CurPos:       objPos.New(10, 10),
				BelongPlayer: faction.HumanAlpha,
				Length:       30,
				Width:        12,
			}
			damaged := &objUnit.BattleShip{
				Uid:          "damaged",
				CurHP:        40,
				TotalHP:      100,
				CurPos:       objPos.New(12, 10),
				BelongPlayer: faction.HumanAlpha,
			}
			full := &objUnit.BattleShip{
				Uid:          "full",
				CurHP:        100,
				TotalHP:      100,
				CurPos:       objPos.New(11, 10),
				BelongPlayer: faction.HumanAlpha,
			}
			enemy := &objUnit.BattleShip{
				Uid:          "enemy",
				CurHP:        40,
				TotalHP:      100,
				CurPos:       objPos.New(11, 11),
				BelongPlayer: faction.ComputerAlpha,
			}
			far := &objUnit.BattleShip{
				Uid:          "far",
				CurHP:        40,
				TotalHP:      100,
				CurPos:       objPos.New(20, 10),
				BelongPlayer: faction.HumanAlpha,
			}

			manager := &MissionManager{state: &state.MissionState{
				Arena: state.MissionArenaState{
					Ships: map[string]*objUnit.BattleShip{
						support.Uid: support,
						damaged.Uid: damaged,
						full.Uid:    full,
						enemy.Uid:   enemy,
						far.Uid:     far,
					},
				},
				UI: state.MissionUIState{
					GameMarks: map[objMark.ID]*objMark.Mark{},
				},
			}}

			manager.updateHospitalShipHealing()

			if damaged.CurHP != damaged.TotalHP {
				t.Fatalf("nearby friendly ship HP = %.1f, want %.1f", damaged.CurHP, damaged.TotalHP)
			}
			if full.CurHP != full.TotalHP {
				t.Fatalf("full-health ship HP = %.1f, want %.1f", full.CurHP, full.TotalHP)
			}
			if enemy.CurHP != 40 {
				t.Fatalf("enemy ship was healed to %.1f", enemy.CurHP)
			}
			if far.CurHP != 40 {
				t.Fatalf("out-of-range ship was healed to %.1f", far.CurHP)
			}
			if support.LastHealAt == 0 {
				t.Fatal("support ship heal timestamp was not updated")
			}
			if len(manager.state.UI.GameMarks) != 1 {
				t.Fatalf("healing marks = %d, want 1", len(manager.state.UI.GameMarks))
			}

			manager.updateHospitalShipHealing()
			if damaged.CurHP != damaged.TotalHP {
				t.Fatalf("ship HP changed before the next heal interval: %.1f", damaged.CurHP)
			}
		})
	}
}
